// Package reports is the API module for publishing neighbor reports.
package reports

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/imaging"
)

const (
	minPhotos       = 1
	maxPhotos       = 3
	maxTitle        = 100
	maxDetails      = 1000
	maxReference    = 150
	maxAuthorName   = 40
	rateLimitCount  = 5
	rateLimitWindow = 10 * time.Minute
	cleanupTimeout  = 10 * time.Second
	photoLinkTTL    = time.Hour

	// MaxPhotoBytes is the largest photo, as received, that is accepted.
	MaxPhotoBytes = 10 << 20
)

// Repository is what the service needs from the database; implemented by
// repository.PgRepository.
type Repository interface {
	// CategoryExists tells whether slug is one of the fixed categories.
	CategoryExists(ctx context.Context, slug string) (bool, error)
	// CountRecent summarizes the reports authorID published since the given time.
	CountRecent(ctx context.Context, authorID uuid.UUID, since time.Time) (RecentReports, error)
	// Create stores the report and its photos in one transaction.
	Create(ctx context.Context, report Report) error
	// Get returns the report with its photos in order, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (Report, error)
}

// PhotoStore is the file storage the service needs; implemented by
// storage.S3.
type PhotoStore interface {
	Put(ctx context.Context, key string, data []byte) error
	// Delete removes the files; keys that do not exist are not an error.
	Delete(ctx context.Context, keys ...string) error
	// URL returns a signed link to read the file until ttl passes.
	URL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type Service struct {
	repo   Repository
	photos PhotoStore
	now    func() time.Time
}

func NewService(repo Repository, photos PhotoStore) *Service {
	return &Service{repo: repo, photos: photos, now: time.Now}
}

// Create validates and publishes a report. It either stores the report with
// all its photos or stores nothing visible: photos are uploaded first, the
// report is inserted afterwards, and the uploads are removed if anything
// fails in between.
func (s *Service) Create(ctx context.Context, in CreateReportInput) (CreatedReportDTO, error) {
	report, err := s.validate(in)
	if err != nil {
		return CreatedReportDTO{}, err
	}

	ok, err := s.repo.CategoryExists(ctx, report.CategorySlug)
	if err != nil {
		return CreatedReportDTO{}, fmt.Errorf("check category: %w", err)
	}
	if !ok {
		return CreatedReportDTO{}, invalid("unknown category")
	}

	if err := s.checkRate(ctx, report.AuthorID); err != nil {
		return CreatedReportDTO{}, err
	}

	cleaned := make([][]byte, len(in.Photos))
	for i, raw := range in.Photos {
		cleaned[i], err = imaging.Clean(raw)
		if errors.Is(err, imaging.ErrInvalidImage) {
			return CreatedReportDTO{}, invalid("photo %d is not a valid JPEG or PNG image", i+1)
		}
		if err != nil {
			return CreatedReportDTO{}, fmt.Errorf("clean photo %d: %w", i+1, err)
		}
	}

	report.ID = uuid.New()
	for i := range cleaned {
		report.Photos = append(report.Photos, Photo{
			Position:   i + 1,
			StorageKey: fmt.Sprintf("reports/%s/%d.jpg", report.ID, i+1),
		})
	}

	if err := s.upload(ctx, report.Photos, cleaned); err != nil {
		s.removePhotos(ctx, report.Photos)
		return CreatedReportDTO{}, err
	}
	if err := s.repo.Create(ctx, report); err != nil {
		s.removePhotos(ctx, report.Photos)
		return CreatedReportDTO{}, fmt.Errorf("create report: %w", err)
	}
	return CreatedReportDTO{ID: report.ID, Status: StatusPublished}, nil
}

// Get returns a report with its photos as signed links that expire after one
// hour. An id that is not a valid UUID is reported as ErrNotFound, like an
// unknown one.
func (s *Service) Get(ctx context.Context, id string) (ReportDTO, error) {
	reportID, err := uuid.Parse(id)
	if err != nil {
		return ReportDTO{}, ErrNotFound
	}
	r, err := s.repo.Get(ctx, reportID)
	if err != nil {
		return ReportDTO{}, fmt.Errorf("get report: %w", err)
	}

	photos := make([]PhotoDTO, len(r.Photos))
	for i, p := range r.Photos {
		link, err := s.photos.URL(ctx, p.StorageKey, photoLinkTTL)
		if err != nil {
			return ReportDTO{}, fmt.Errorf("sign photo %d: %w", p.Position, err)
		}
		photos[i] = PhotoDTO{URL: link}
	}
	return ReportDTO{
		ID:            r.ID,
		Title:         r.Title,
		Details:       r.Details,
		CategorySlug:  r.CategorySlug,
		CategoryLabel: r.CategoryLabel,
		Status:        r.Status,
		Latitude:      r.Latitude,
		Longitude:     r.Longitude,
		Reference:     r.Reference,
		AuthorName:    r.AuthorName,
		CreatedAt:     r.CreatedAt,
		Photos:        photos,
	}, nil
}

// validate checks the data sent by the neighbor and returns the report to
// store, with text fields trimmed.
func (s *Service) validate(in CreateReportInput) (Report, error) {
	r := Report{
		AuthorID:     in.AuthorID,
		CategorySlug: strings.TrimSpace(in.CategorySlug),
		Title:        strings.TrimSpace(in.Title),
		Details:      strings.TrimSpace(in.Details),
		Reference:    strings.TrimSpace(in.Reference),
		Latitude:     in.Latitude,
		Longitude:    in.Longitude,
		AuthorName:   strings.TrimSpace(in.AuthorName),
	}

	switch {
	case r.CategorySlug == "":
		return Report{}, invalid("category is required")
	case r.Title == "":
		return Report{}, invalid("title is required")
	case utf8.RuneCountInString(r.Title) > maxTitle:
		return Report{}, invalid("title must have at most %d characters", maxTitle)
	case utf8.RuneCountInString(r.Details) > maxDetails:
		return Report{}, invalid("details must have at most %d characters", maxDetails)
	case utf8.RuneCountInString(r.Reference) > maxReference:
		return Report{}, invalid("reference must have at most %d characters", maxReference)
	case utf8.RuneCountInString(r.AuthorName) > maxAuthorName:
		return Report{}, invalid("author name must have at most %d characters", maxAuthorName)
	case !inRange(r.Latitude, 90):
		return Report{}, invalid("latitude must be between -90 and 90")
	case !inRange(r.Longitude, 180):
		return Report{}, invalid("longitude must be between -180 and 180")
	case len(in.Photos) < minPhotos || len(in.Photos) > maxPhotos:
		return Report{}, invalid("a report needs between %d and %d photos", minPhotos, maxPhotos)
	}
	for _, p := range in.Photos {
		if len(p) > MaxPhotoBytes {
			return Report{}, ErrPhotoTooLarge
		}
	}
	return r, nil
}

func inRange(v, limit float64) bool {
	return !math.IsNaN(v) && math.Abs(v) <= limit
}

// checkRate rejects the call when the author already published the maximum
// number of reports inside the window.
func (s *Service) checkRate(ctx context.Context, authorID uuid.UUID) error {
	now := s.now()
	recent, err := s.repo.CountRecent(ctx, authorID, now.Add(-rateLimitWindow))
	if err != nil {
		return fmt.Errorf("count recent reports: %w", err)
	}
	if recent.Count < rateLimitCount {
		return nil
	}
	wait := recent.Oldest.Add(rateLimitWindow).Sub(now)
	if wait < time.Second {
		wait = time.Second
	}
	return &TooManyReportsError{RetryAfter: wait}
}

func (s *Service) upload(ctx context.Context, photos []Photo, data [][]byte) error {
	for i, p := range photos {
		if err := s.photos.Put(ctx, p.StorageKey, data[i]); err != nil {
			return fmt.Errorf("store photo %d: %w", p.Position, err)
		}
	}
	return nil
}

// removePhotos deletes the files of a report that was not created. A failure
// is only logged: the keys are namespaced by report id, so a leftover file is
// unreachable and never shows up as a report.
func (s *Service) removePhotos(ctx context.Context, photos []Photo) {
	keys := make([]string, len(photos))
	for i, p := range photos {
		keys[i] = p.StorageKey
	}
	// The request may already be cancelled; the cleanup must still run.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	if err := s.photos.Delete(ctx, keys...); err != nil {
		slog.ErrorContext(ctx, "remove photos of unpublished report", "error", err, "keys", keys)
	}
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidReport, fmt.Sprintf(format, args...))
}

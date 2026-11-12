// Package reports is the API module for publishing neighbor reports.
package reports

import (
	"context"
	"encoding/base64"
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
	defaultPageSize = 20
	maxPageSize     = 50

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
	// List returns up to params.Limit reports, newest first, continuing after
	// the key in params.
	List(ctx context.Context, params ListParams) ([]ReportSummary, error)
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

// List returns one page of the feed, newest first. Each cover photo is a
// signed link that expires after one hour. The cursor in the page continues
// right after its last item; it is empty on the last page.
func (s *Service) List(ctx context.Context, in ListReportsInput) (ReportPageDTO, error) {
	limit := in.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maxPageSize {
		return ReportPageDTO{}, invalid("limit must be between 1 and %d", maxPageSize)
	}
	if (in.Latitude == nil) != (in.Longitude == nil) {
		return ReportPageDTO{}, invalid("latitude and longitude must be sent together")
	}
	if in.Latitude != nil {
		if !inRange(*in.Latitude, 90) {
			return ReportPageDTO{}, invalid("latitude must be between -90 and 90")
		}
		if !inRange(*in.Longitude, 180) {
			return ReportPageDTO{}, invalid("longitude must be between -180 and 180")
		}
	}

	// One more than the page, to know whether there is a next page.
	params := ListParams{Limit: limit + 1, Latitude: in.Latitude, Longitude: in.Longitude}
	if in.Cursor != "" {
		after, id, err := decodeCursor(in.Cursor)
		if err != nil {
			return ReportPageDTO{}, err
		}
		params.HasAfter, params.After, params.AfterID = true, after, id
	}
	rows, err := s.repo.List(ctx, params)
	if err != nil {
		return ReportPageDTO{}, fmt.Errorf("list reports: %w", err)
	}

	page := ReportPageDTO{Items: []ReportSummaryDTO{}}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	for _, r := range rows {
		link, err := s.photos.URL(ctx, r.CoverKey, photoLinkTTL)
		if err != nil {
			return ReportPageDTO{}, fmt.Errorf("sign cover of report %s: %w", r.ID, err)
		}
		page.Items = append(page.Items, ReportSummaryDTO{
			ID:             r.ID,
			Title:          r.Title,
			CategorySlug:   r.CategorySlug,
			CategoryLabel:  r.CategoryLabel,
			Status:         r.Status,
			Reference:      r.Reference,
			CreatedAt:      r.CreatedAt,
			CoverPhoto:     PhotoDTO{URL: link},
			DistanceMeters: r.Distance,
		})
	}
	return page, nil
}

// encodeCursor builds the opaque cursor that continues after a report: the
// base64url of "<created_at RFC3339Nano>|<id>". The full time precision is
// kept so that reports with the same second are still told apart.
func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor is the inverse of encodeCursor; anything else is
// ErrInvalidCursor.
func decodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	ts, idText, ok := strings.Cut(string(raw), "|")
	if !ok {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	return createdAt, id, nil
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

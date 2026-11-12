// Package reports is the API module for publishing neighbor reports.
package reports

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
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
	maxQuery        = 100
	maxAuthorName   = 40
	rateLimitCount  = 5
	rateLimitWindow = 10 * time.Minute
	cleanupTimeout  = 10 * time.Second
	photoLinkTTL    = time.Hour
	defaultPageSize = 20
	maxPageSize     = 50
	maxCategories   = 5
	maxMapPins      = 500

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
	// Search returns up to params.Limit reports that contain every word, in
	// the order of params, continuing after the key in params.
	Search(ctx context.Context, params SearchParams) ([]SearchHit, error)
	// ListPins returns up to params.Limit map points, newest first.
	ListPins(ctx context.Context, params MapParams) ([]ReportPin, error)
	// UnknownCategories returns the slugs that are not one of the fixed
	// categories.
	UnknownCategories(ctx context.Context, slugs []string) ([]string, error)
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
	limit, err := checkPaging(in.Limit, in.Latitude, in.Longitude)
	if err != nil {
		return ReportPageDTO{}, err
	}

	categories, err := s.checkFilters(ctx, in.Categories, in.Status)
	if err != nil {
		return ReportPageDTO{}, err
	}

	// One more than the page, to know whether there is a next page.
	params := ListParams{
		Limit:      limit + 1,
		Latitude:   in.Latitude,
		Longitude:  in.Longitude,
		Categories: categories,
		Status:     in.Status,
	}
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

	page := ReportPageDTO{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	if page.Items, err = s.cards(ctx, rows); err != nil {
		return ReportPageDTO{}, err
	}
	return page, nil
}

// Map returns the map points: the newest reports that match the filters, up to
// maxMapPins, without pages.
func (s *Service) Map(ctx context.Context, in MapInput) ([]ReportPinDTO, error) {
	categories, err := s.checkFilters(ctx, in.Categories, in.Status)
	if err != nil {
		return nil, err
	}
	pins, err := s.repo.ListPins(ctx, MapParams{Categories: categories, Status: in.Status, Limit: maxMapPins})
	if err != nil {
		return nil, fmt.Errorf("list report pins: %w", err)
	}
	items := make([]ReportPinDTO, len(pins))
	for i, p := range pins {
		items[i] = ReportPinDTO{
			ID:            p.ID,
			Title:         p.Title,
			CategorySlug:  p.CategorySlug,
			CategoryLabel: p.CategoryLabel,
			Status:        p.Status,
			Latitude:      p.Latitude,
			Longitude:     p.Longitude,
		}
	}
	return items, nil
}

// checkFilters validates the category and status filters of the feed and the
// map, and returns the categories without duplicates. An empty status means
// any status.
func (s *Service) checkFilters(ctx context.Context, categories []string, status string) ([]string, error) {
	if status != "" && !slices.Contains(Statuses, status) {
		return nil, invalid("unknown status")
	}
	unique := make([]string, 0, len(categories))
	for _, c := range categories {
		if !slices.Contains(unique, c) {
			unique = append(unique, c)
		}
	}
	if len(unique) > maxCategories {
		return nil, invalid("at most %d categories can be sent", maxCategories)
	}
	if len(unique) > 0 {
		unknown, err := s.repo.UnknownCategories(ctx, unique)
		if err != nil {
			return nil, fmt.Errorf("check categories: %w", err)
		}
		if len(unknown) > 0 {
			return nil, invalid("unknown category")
		}
	}
	return unique, nil
}

// Search returns one page of the search. A report matches when its title or
// details contain every word of the query, ignoring case and accents. The
// order is newest first, or nearest first with Sort "distance", which needs a
// location. A query without words is only accepted when sorting by distance.
// Covers are signed as in List. The cursor continues right after the last
// item and only works with the order it was issued for; it is empty on the
// last page.
func (s *Service) Search(ctx context.Context, in SearchReportsInput) (ReportPageDTO, error) {
	if utf8.RuneCountInString(in.Query) > maxQuery {
		return ReportPageDTO{}, invalid("q must have at most %d characters", maxQuery)
	}
	var order SearchOrder
	switch in.Sort {
	case "", string(OrderRecent):
		order = OrderRecent
	case string(OrderDistance):
		order = OrderDistance
	default:
		return ReportPageDTO{}, invalid("sort must be recent or distance")
	}
	limit, err := checkPaging(in.Limit, in.Latitude, in.Longitude)
	if err != nil {
		return ReportPageDTO{}, err
	}
	words := searchWords(in.Query)
	if order == OrderRecent && len(words) == 0 {
		return ReportPageDTO{}, invalid("q is required unless sorting by distance")
	}
	if order == OrderDistance && in.Latitude == nil {
		return ReportPageDTO{}, invalid("sorting by distance needs latitude and longitude")
	}

	params := SearchParams{
		Order:     order,
		Words:     words,
		Limit:     limit + 1, // one more than the page, to know whether there is a next page
		Latitude:  in.Latitude,
		Longitude: in.Longitude,
	}
	if in.Cursor != "" {
		key, err := decodeSearchCursor(in.Cursor, order)
		if err != nil {
			return ReportPageDTO{}, err
		}
		params.HasAfter, params.After, params.AfterID, params.AfterDistance = true, key.CreatedAt, key.ID, key.Distance
	}
	hits, err := s.repo.Search(ctx, params)
	if err != nil {
		return ReportPageDTO{}, fmt.Errorf("search reports: %w", err)
	}

	page := ReportPageDTO{}
	if len(hits) > limit {
		hits = hits[:limit]
		last := hits[limit-1]
		page.NextCursor = encodeSearchCursor(order, last)
	}
	rows := make([]ReportSummary, len(hits))
	for i, h := range hits {
		rows[i] = h.ReportSummary
	}
	if page.Items, err = s.cards(ctx, rows); err != nil {
		return ReportPageDTO{}, err
	}
	return page, nil
}

// checkPaging validates the page size and the optional location shared by the
// feed and the search, and returns the page size to use.
func checkPaging(limit int, lat, lng *float64) (int, error) {
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maxPageSize {
		return 0, invalid("limit must be between 1 and %d", maxPageSize)
	}
	if (lat == nil) != (lng == nil) {
		return 0, invalid("latitude and longitude must be sent together")
	}
	if lat != nil {
		if !inRange(*lat, 90) {
			return 0, invalid("latitude must be between -90 and 90")
		}
		if !inRange(*lng, 180) {
			return 0, invalid("longitude must be between -180 and 180")
		}
	}
	return limit, nil
}

// cards turns rows into cards, signing each cover for one hour.
func (s *Service) cards(ctx context.Context, rows []ReportSummary) ([]ReportSummaryDTO, error) {
	items := make([]ReportSummaryDTO, 0, len(rows))
	for _, r := range rows {
		link, err := s.photos.URL(ctx, r.CoverKey, photoLinkTTL)
		if err != nil {
			return nil, fmt.Errorf("sign cover of report %s: %w", r.ID, err)
		}
		items = append(items, ReportSummaryDTO{
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
	return items, nil
}

// searchWords splits the query on whitespace and escapes \, % and _ in each
// word so that ILIKE treats them as plain text.
func searchWords(q string) []string {
	fields := strings.Fields(q)
	words := make([]string, len(fields))
	for i, f := range fields {
		words[i] = likeEscaper.Replace(f)
	}
	return words
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// searchCursor is the key a search page continues after.
type searchCursor struct {
	Order     SearchOrder `json:"o"`
	CreatedAt time.Time   `json:"t"`
	ID        uuid.UUID   `json:"i"`
	Distance  float64     `json:"d,omitempty"` // exact meters; only for OrderDistance
}

// encodeSearchCursor builds the opaque cursor that continues after hit: the
// base64url of a JSON object that carries the order it belongs to.
func encodeSearchCursor(order SearchOrder, hit SearchHit) string {
	c := searchCursor{Order: order, CreatedAt: hit.CreatedAt.UTC(), ID: hit.ID}
	if order == OrderDistance {
		c.Distance = hit.ExactDistance
	}
	raw, _ := json.Marshal(c) // only strings, a time and a float: cannot fail
	return base64.RawURLEncoding.EncodeToString(raw)
}

// decodeSearchCursor is the inverse of encodeSearchCursor. A malformed value,
// a feed cursor, or a cursor issued for another order is ErrInvalidCursor.
func decodeSearchCursor(cursor string, order SearchOrder) (searchCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return searchCursor{}, ErrInvalidCursor
	}
	var c searchCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return searchCursor{}, ErrInvalidCursor
	}
	if c.Order != order || c.ID == uuid.Nil || c.CreatedAt.IsZero() ||
		math.IsNaN(c.Distance) || math.IsInf(c.Distance, 0) || c.Distance < 0 {
		return searchCursor{}, ErrInvalidCursor
	}
	return c, nil
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

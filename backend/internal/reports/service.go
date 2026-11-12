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
	"github.com/pgvector/pgvector-go"

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

	maxContribution       = 1200
	contributionRateCount = 10
	contributionRateWin   = 10 * time.Minute

	// publishEmbedTimeout bounds how long publishing waits for the embedding;
	// past it the report is stored without one and the backfill job embeds it.
	publishEmbedTimeout = 3 * time.Second
	// similarEmbedTimeout bounds the embedding of a draft that is being checked.
	similarEmbedTimeout = 5 * time.Second
	// backfillEmbedTimeout bounds one embedding of the backfill job.
	backfillEmbedTimeout = 10 * time.Second
	backfillInterval     = 5 * time.Minute
	backfillBatch        = 20

	// maxSimilarDistance is the largest cosine distance (0 is the same meaning,
	// 1 is unrelated) at which a nearby report is offered as similar.
	//
	// Calibrated with voyage-4 on 15 invented Spanish pairs of short report
	// texts: pairs of the same problem in other words measured 0.10 to 0.26;
	// different problems of the same category 0.23 to 0.42; different
	// categories 0.33 to 0.52. 0.3 keeps every same-problem pair and drops every
	// pair of another category, but lets through some related problems of the
	// same category (for example a water leak and a clogged drain), which is
	// fine: the dialog only suggests, and «No es ninguno, seguir» dismisses it.
	// The sample is small, so retune this with real reports; the scale changes
	// with the model (voyage-3.5-lite needed about 0.2).
	maxSimilarDistance = 0.3

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
	// Get returns the report with its photos in order and its relation to
	// caller (author or supporter), or ErrNotFound.
	Get(ctx context.Context, id, caller uuid.UUID) (Report, error)
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

	// GetAuthor returns the id of the session that published the report, or
	// ErrNotFound. It is only for decisions inside the service.
	GetAuthor(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	// FindSimilar returns the open reports within 300 m of the point whose
	// meaning is close to the embedding, best match first, at most 5.
	FindSimilar(ctx context.Context, params SimilarParams) ([]SimilarReport, error)
	// Support marks "También me afecta" for userID; marking twice changes
	// nothing. Unsupport removes it; removing it twice changes nothing.
	Support(ctx context.Context, reportID, userID uuid.UUID) error
	Unsupport(ctx context.Context, reportID, userID uuid.UUID) error
	// CountAffected returns the author plus every other neighbor who supports
	// the report.
	CountAffected(ctx context.Context, reportID uuid.UUID) (int, error)
	// CountRecentContributions summarizes the aportes authorID added since the
	// given time.
	CountRecentContributions(ctx context.Context, authorID uuid.UUID, since time.Time) (RecentReports, error)
	// CreateContribution stores the aporte and, when markSupport is set, the
	// support of its author, in one transaction. It returns the aporte as stored.
	CreateContribution(ctx context.Context, c Contribution, markSupport bool) (Contribution, error)
	// ListContributions returns up to params.Limit aportes of a report, oldest
	// first, continuing after the key in params.
	ListContributions(ctx context.Context, params ContributionParams) ([]Contribution, error)

	BackfillRepository
}

// BackfillRepository is what the embedding backfill job needs; Repository
// includes it.
type BackfillRepository interface {
	// ListMissingEmbedding returns up to limit reports that have no embedding,
	// oldest first.
	ListMissingEmbedding(ctx context.Context, limit int) ([]ReportText, error)
	// SetEmbedding stores the embedding of a report that has none.
	SetEmbedding(ctx context.Context, id uuid.UUID, embedding pgvector.Vector) error
}

// Embedder turns a text into a vector that captures its meaning; implemented by
// embedding.Voyage. Only comparableText may be passed to it.
type Embedder interface {
	Embed(ctx context.Context, text string) (pgvector.Vector, error)
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
	repo     Repository
	photos   PhotoStore
	embedder Embedder
	now      func() time.Time
}

func NewService(repo Repository, photos PhotoStore, embedder Embedder) *Service {
	return &Service{repo: repo, photos: photos, embedder: embedder, now: time.Now}
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

	report.Embedding = s.embedReport(ctx, report)

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
// hour, and the caller's relation to it. An id that is not a valid UUID is
// reported as ErrNotFound, like an unknown one.
func (s *Service) Get(ctx context.Context, id string, caller uuid.UUID) (ReportDTO, error) {
	reportID, err := uuid.Parse(id)
	if err != nil {
		return ReportDTO{}, ErrNotFound
	}
	r, err := s.repo.Get(ctx, reportID, caller)
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

		AffectedCount:     r.AffectedCount,
		ContributionCount: r.ContributionCount,
		Support:           supportState(r.IsAuthor, r.Supported),
	}, nil
}

// supportState tells the caller's relation to a report. The author always
// counts as affected, whatever the support rows say.
func supportState(isAuthor, supported bool) string {
	switch {
	case isAuthor:
		return SupportAuthor
	case supported:
		return SupportSupporter
	default:
		return SupportNone
	}
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
		// Same fields as the entity, none of them the author: a plain conversion.
		items[i] = ReportPinDTO(p)
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
			AffectedCount:  r.AffectedCount,
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
	return &TooManyReportsError{RetryAfter: retryAfter(recent.Oldest, rateLimitWindow, now)}
}

// retryAfter is how long to wait until the oldest action inside the window
// leaves it, at least one second.
func retryAfter(oldest time.Time, window time.Duration, now time.Time) time.Duration {
	wait := oldest.Add(window).Sub(now)
	if wait < time.Second {
		wait = time.Second
	}
	return wait
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

// comparableText is the only text that leaves the system to be compared: the
// title, and the details when there are any. Never the author name, the
// reference, the location nor any user data. Callers pass trimmed values.
func comparableText(title, details string) string {
	if details == "" {
		return title
	}
	return title + "\n" + details
}

// embedReport returns the embedding of a report about to be published, or nil
// when it cannot be obtained within publishEmbedTimeout: publishing never
// fails because of the comparison service, and the backfill job embeds the
// report later.
func (s *Service) embedReport(ctx context.Context, r Report) *pgvector.Vector {
	ctx, cancel := context.WithTimeout(ctx, publishEmbedTimeout)
	defer cancel()
	vec, err := s.embedder.Embed(ctx, comparableText(r.Title, r.Details))
	if err != nil {
		// The error never carries the text.
		slog.WarnContext(ctx, "embed report on publish; the backfill will retry", "error", err)
		return nil
	}
	return &vec
}

// Similar returns the open reports within 300 m of the draft's location whose
// meaning is close to its title and details, best match first, at most 5, with
// each cover as a signed link that expires after one hour. When the comparison
// service fails it returns ErrComparisonUnavailable.
func (s *Service) Similar(ctx context.Context, in SimilarReportsInput) ([]SimilarReportDTO, error) {
	title, details := strings.TrimSpace(in.Title), strings.TrimSpace(in.Details)
	switch {
	case title == "":
		return nil, invalid("title is required")
	case utf8.RuneCountInString(title) > maxTitle:
		return nil, invalid("title must have at most %d characters", maxTitle)
	case utf8.RuneCountInString(details) > maxDetails:
		return nil, invalid("details must have at most %d characters", maxDetails)
	case !inRange(in.Latitude, 90):
		return nil, invalid("latitude must be between -90 and 90")
	case !inRange(in.Longitude, 180):
		return nil, invalid("longitude must be between -180 and 180")
	}

	embedCtx, cancel := context.WithTimeout(ctx, similarEmbedTimeout)
	defer cancel()
	vec, err := s.embedder.Embed(embedCtx, comparableText(title, details))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrComparisonUnavailable, err)
	}

	found, err := s.repo.FindSimilar(ctx, SimilarParams{
		Embedding:   vec,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		MaxDistance: maxSimilarDistance,
	})
	if err != nil {
		return nil, fmt.Errorf("find similar reports: %w", err)
	}

	items := make([]SimilarReportDTO, 0, len(found))
	for _, r := range found {
		link, err := s.photos.URL(ctx, r.CoverKey, photoLinkTTL)
		if err != nil {
			return nil, fmt.Errorf("sign cover of report %s: %w", r.ID, err)
		}
		items = append(items, SimilarReportDTO{
			ID:             r.ID,
			Title:          r.Title,
			CategorySlug:   r.CategorySlug,
			CategoryLabel:  r.CategoryLabel,
			Status:         r.Status,
			CreatedAt:      r.CreatedAt,
			CoverPhoto:     PhotoDTO{URL: link},
			DistanceMeters: r.Distance,
			AffectedCount:  r.AffectedCount,
		})
	}
	return items, nil
}

// Support marks "También me afecta" for the caller. It is idempotent. The
// author of the report cannot change their support (ErrIsAuthor), and an
// unknown or malformed id is ErrNotFound.
func (s *Service) Support(ctx context.Context, id string, caller uuid.UUID) (ReportSupportDTO, error) {
	return s.changeSupport(ctx, id, caller, true)
}

// Unsupport removes the caller's support. It is idempotent, and fails as
// Support does.
func (s *Service) Unsupport(ctx context.Context, id string, caller uuid.UUID) (ReportSupportDTO, error) {
	return s.changeSupport(ctx, id, caller, false)
}

// changeSupport is not a transaction: the count is read after the write, so a
// concurrent change by another neighbor only shifts it by one.
func (s *Service) changeSupport(ctx context.Context, id string, caller uuid.UUID, mark bool) (ReportSupportDTO, error) {
	reportID, err := uuid.Parse(id)
	if err != nil {
		return ReportSupportDTO{}, ErrNotFound
	}
	author, err := s.repo.GetAuthor(ctx, reportID)
	if err != nil {
		return ReportSupportDTO{}, fmt.Errorf("get report author: %w", err)
	}
	if author == caller {
		return ReportSupportDTO{}, ErrIsAuthor
	}

	state := SupportNone
	if mark {
		err, state = s.repo.Support(ctx, reportID, caller), SupportSupporter
	} else {
		err = s.repo.Unsupport(ctx, reportID, caller)
	}
	if err != nil {
		return ReportSupportDTO{}, fmt.Errorf("change support: %w", err)
	}

	count, err := s.repo.CountAffected(ctx, reportID)
	if err != nil {
		return ReportSupportDTO{}, fmt.Errorf("count affected: %w", err)
	}
	return ReportSupportDTO{AffectedCount: count, Support: state}, nil
}

// Contribute adds an aporte to a report, which is also joining it: the caller's
// support is marked in the same transaction, unless the caller is the author or
// already supports it, which leave the count as it was. The photo, when there
// is one, is cleaned and stored first, and removed again if the aporte cannot
// be saved.
func (s *Service) Contribute(ctx context.Context, in ContributionInput) (CreatedContributionDTO, error) {
	text := strings.TrimSpace(in.Text)
	name := strings.TrimSpace(in.AuthorName)
	switch {
	case text == "":
		return CreatedContributionDTO{}, invalid("text is required")
	case utf8.RuneCountInString(text) > maxContribution:
		return CreatedContributionDTO{}, invalid("text must have at most %d characters", maxContribution)
	case utf8.RuneCountInString(name) > maxAuthorName:
		return CreatedContributionDTO{}, invalid("author name must have at most %d characters", maxAuthorName)
	case len(in.Photo) > MaxPhotoBytes:
		return CreatedContributionDTO{}, ErrPhotoTooLarge
	}

	reportID, err := uuid.Parse(in.ReportID)
	if err != nil {
		return CreatedContributionDTO{}, ErrNotFound
	}
	author, err := s.repo.GetAuthor(ctx, reportID)
	if err != nil {
		return CreatedContributionDTO{}, fmt.Errorf("get report author: %w", err)
	}

	if err := s.checkContributionRate(ctx, in.AuthorID); err != nil {
		return CreatedContributionDTO{}, err
	}

	c := Contribution{
		ID:         uuid.New(),
		ReportID:   reportID,
		AuthorID:   in.AuthorID,
		AuthorName: name,
		Text:       text,
	}
	var stored []Photo
	if in.Photo != nil {
		cleaned, err := imaging.Clean(in.Photo)
		if errors.Is(err, imaging.ErrInvalidImage) {
			return CreatedContributionDTO{}, invalid("photo is not a valid JPEG or PNG image")
		}
		if err != nil {
			return CreatedContributionDTO{}, fmt.Errorf("clean photo: %w", err)
		}
		c.PhotoKey = fmt.Sprintf("contributions/%s.jpg", c.ID)
		stored = []Photo{{Position: 1, StorageKey: c.PhotoKey}}
		if err := s.upload(ctx, stored, [][]byte{cleaned}); err != nil {
			s.removePhotos(ctx, stored)
			return CreatedContributionDTO{}, err
		}
	}

	isAuthor := author == in.AuthorID
	saved, err := s.repo.CreateContribution(ctx, c, !isAuthor)
	if err != nil {
		s.removePhotos(ctx, stored)
		return CreatedContributionDTO{}, fmt.Errorf("create contribution: %w", err)
	}

	count, err := s.repo.CountAffected(ctx, reportID)
	if err != nil {
		return CreatedContributionDTO{}, fmt.Errorf("count affected: %w", err)
	}
	dto, err := s.contributionDTO(ctx, saved)
	if err != nil {
		return CreatedContributionDTO{}, err
	}
	state := SupportSupporter
	if isAuthor {
		state = SupportAuthor
	}
	return CreatedContributionDTO{Contribution: dto, AffectedCount: count, Support: state}, nil
}

// Contributions returns one page of the aportes of a report, oldest first, with
// each photo as a signed link that expires after one hour. The cursor continues
// right after the last item; it is empty on the last page.
func (s *Service) Contributions(ctx context.Context, in ListContributionsInput) (ContributionPageDTO, error) {
	reportID, err := uuid.Parse(in.ReportID)
	if err != nil {
		return ContributionPageDTO{}, ErrNotFound
	}
	if _, err := s.repo.GetAuthor(ctx, reportID); err != nil {
		return ContributionPageDTO{}, fmt.Errorf("get report author: %w", err)
	}
	limit, err := checkPaging(in.Limit, nil, nil)
	if err != nil {
		return ContributionPageDTO{}, err
	}

	// One more than the page, to know whether there is a next page.
	params := ContributionParams{ReportID: reportID, Limit: limit + 1}
	if in.Cursor != "" {
		after, id, err := decodeCursor(in.Cursor)
		if err != nil {
			return ContributionPageDTO{}, err
		}
		params.HasAfter, params.After, params.AfterID = true, after, id
	}
	rows, err := s.repo.ListContributions(ctx, params)
	if err != nil {
		return ContributionPageDTO{}, fmt.Errorf("list contributions: %w", err)
	}

	page := ContributionPageDTO{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	page.Items = make([]ContributionDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := s.contributionDTO(ctx, row)
		if err != nil {
			return ContributionPageDTO{}, err
		}
		page.Items = append(page.Items, dto)
	}
	return page, nil
}

// contributionDTO turns an aporte into its public form, signing its photo.
func (s *Service) contributionDTO(ctx context.Context, c Contribution) (ContributionDTO, error) {
	dto := ContributionDTO{ID: c.ID, Text: c.Text, AuthorName: c.AuthorName, CreatedAt: c.CreatedAt}
	if c.PhotoKey != "" {
		link, err := s.photos.URL(ctx, c.PhotoKey, photoLinkTTL)
		if err != nil {
			return ContributionDTO{}, fmt.Errorf("sign photo of contribution %s: %w", c.ID, err)
		}
		dto.PhotoURL = link
	}
	return dto, nil
}

// checkContributionRate rejects the call when the author already added the
// maximum number of aportes inside the window, across all reports.
func (s *Service) checkContributionRate(ctx context.Context, authorID uuid.UUID) error {
	now := s.now()
	recent, err := s.repo.CountRecentContributions(ctx, authorID, now.Add(-contributionRateWin))
	if err != nil {
		return fmt.Errorf("count recent contributions: %w", err)
	}
	if recent.Count < contributionRateCount {
		return nil
	}
	return &TooManyContributionsError{RetryAfter: retryAfter(recent.Oldest, contributionRateWin, now)}
}

// RunEmbeddingBackfill embeds the reports that were published while the
// comparison service was unavailable. It runs a round right away and then every
// five minutes, until ctx is done. Each round embeds up to 20 reports, oldest
// first, and stops at the first failure to embed, since the service is likely
// still down.
func RunEmbeddingBackfill(ctx context.Context, repo BackfillRepository, embedder Embedder, logger *slog.Logger) {
	ticker := time.NewTicker(backfillInterval)
	defer ticker.Stop()
	for {
		backfillRound(ctx, repo, embedder, logger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// backfillRound runs one round and returns how many reports it embedded.
func backfillRound(ctx context.Context, repo BackfillRepository, embedder Embedder, logger *slog.Logger) int {
	pending, err := repo.ListMissingEmbedding(ctx, backfillBatch)
	if err != nil {
		logger.WarnContext(ctx, "list reports without embedding", "error", err)
		return 0
	}
	done := 0
	for _, r := range pending {
		vec, err := embedWithTimeout(ctx, embedder, comparableText(r.Title, r.Details))
		if err != nil {
			logger.WarnContext(ctx, "embed report in backfill; stopping the round", "error", err)
			return done
		}
		if err := repo.SetEmbedding(ctx, r.ID, vec); err != nil {
			logger.WarnContext(ctx, "store embedding in backfill", "report", r.ID, "error", err)
			return done
		}
		done++
	}
	return done
}

func embedWithTimeout(ctx context.Context, embedder Embedder, text string) (pgvector.Vector, error) {
	ctx, cancel := context.WithTimeout(ctx, backfillEmbedTimeout)
	defer cancel()
	return embedder.Embed(ctx, text)
}

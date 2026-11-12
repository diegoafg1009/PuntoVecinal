package reports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/embedding"
)

type fakeRepo struct {
	stored     map[uuid.UUID]Report
	getErr     error
	categories map[string]bool
	recent     RecentReports
	countErr   error
	createErr  error
	created    []Report
	counted    time.Time
	listRows   []ReportSummary
	listErr    error
	listParams ListParams

	searchHits   []SearchHit
	searchErr    error
	searchParams SearchParams

	pins          []ReportPin
	pinsErr       error
	pinsParams    MapParams
	unknownErr    error
	unknownCalled int

	// Participation.
	authors       map[uuid.UUID]uuid.UUID // report id -> author id; a missing report is ErrNotFound
	authorErr     error
	similar       []SimilarReport
	similarErr    error
	similarParams SimilarParams
	supports      map[uuid.UUID]map[uuid.UUID]bool // report id -> user ids
	supportErr    error
	contribs      []Contribution
	contribRecent RecentReports
	contribCount  error
	contribErr    error
	contribMarked bool // markSupport of the last CreateContribution
	contribParams ContributionParams
	listContribs  error
	missing       []ReportText
	missingErr    error
	embedded      map[uuid.UUID]pgvector.Vector
	setErr        error
}

func (f *fakeRepo) ListPins(_ context.Context, p MapParams) ([]ReportPin, error) {
	f.pinsParams = p
	if f.pinsErr != nil {
		return nil, f.pinsErr
	}
	return f.pins, nil
}

func (f *fakeRepo) UnknownCategories(_ context.Context, slugs []string) ([]string, error) {
	f.unknownCalled++
	if f.unknownErr != nil {
		return nil, f.unknownErr
	}
	var unknown []string
	for _, s := range slugs {
		if !f.categories[s] {
			unknown = append(unknown, s)
		}
	}
	return unknown, nil
}

func (f *fakeRepo) CategoryExists(_ context.Context, slug string) (bool, error) {
	return f.categories[slug], nil
}

func (f *fakeRepo) CountRecent(_ context.Context, _ uuid.UUID, since time.Time) (RecentReports, error) {
	f.counted = since
	return f.recent, f.countErr
}

func (f *fakeRepo) Create(_ context.Context, r Report) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, r)
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id, _ uuid.UUID) (Report, error) {
	if f.getErr != nil {
		return Report{}, f.getErr
	}
	r, ok := f.stored[id]
	if !ok {
		return Report{}, ErrNotFound
	}
	return r, nil
}

func (f *fakeRepo) List(_ context.Context, p ListParams) ([]ReportSummary, error) {
	f.listParams = p
	if f.listErr != nil {
		return nil, f.listErr
	}
	rows := f.listRows
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
	}
	return rows, nil
}

func (f *fakeRepo) Search(_ context.Context, p SearchParams) ([]SearchHit, error) {
	f.searchParams = p
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	hits := f.searchHits
	if len(hits) > p.Limit {
		hits = hits[:p.Limit]
	}
	return hits, nil
}

func (f *fakeRepo) GetAuthor(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	if f.authorErr != nil {
		return uuid.Nil, f.authorErr
	}
	author, ok := f.authors[id]
	if !ok {
		return uuid.Nil, ErrNotFound
	}
	return author, nil
}

func (f *fakeRepo) FindSimilar(_ context.Context, p SimilarParams) ([]SimilarReport, error) {
	f.similarParams = p
	return f.similar, f.similarErr
}

func (f *fakeRepo) Support(_ context.Context, reportID, userID uuid.UUID) error {
	if f.supportErr != nil {
		return f.supportErr
	}
	if f.supports == nil {
		f.supports = map[uuid.UUID]map[uuid.UUID]bool{}
	}
	if f.supports[reportID] == nil {
		f.supports[reportID] = map[uuid.UUID]bool{}
	}
	f.supports[reportID][userID] = true
	return nil
}

func (f *fakeRepo) Unsupport(_ context.Context, reportID, userID uuid.UUID) error {
	if f.supportErr != nil {
		return f.supportErr
	}
	delete(f.supports[reportID], userID)
	return nil
}

// CountAffected is the author plus the supporters, as the real query counts.
func (f *fakeRepo) CountAffected(_ context.Context, reportID uuid.UUID) (int, error) {
	total := 1
	for user := range f.supports[reportID] {
		if user != f.authors[reportID] {
			total++
		}
	}
	return total, nil
}

func (f *fakeRepo) CountRecentContributions(_ context.Context, _ uuid.UUID, since time.Time) (RecentReports, error) {
	f.counted = since
	return f.contribRecent, f.contribCount
}

// CreateContribution is all or nothing, like the real transaction: on error
// neither the aporte nor the support is stored.
func (f *fakeRepo) CreateContribution(ctx context.Context, c Contribution, markSupport bool) (Contribution, error) {
	f.contribMarked = markSupport
	if f.contribErr != nil {
		return Contribution{}, f.contribErr
	}
	c.CreatedAt = time.Date(2026, 11, 12, 10, 0, len(f.contribs), 0, time.UTC)
	f.contribs = append(f.contribs, c)
	if markSupport {
		_ = f.Support(ctx, c.ReportID, c.AuthorID)
	}
	c.AuthorID = uuid.Nil // read back without the author
	return c, nil
}

// ListContributions pages the stored aportes of the report oldest first, with
// the same keyset as the real query.
func (f *fakeRepo) ListContributions(_ context.Context, p ContributionParams) ([]Contribution, error) {
	f.contribParams = p
	if f.listContribs != nil {
		return nil, f.listContribs
	}
	var rows []Contribution
	for _, c := range f.contribs {
		after := !p.HasAfter || c.CreatedAt.After(p.After) || (c.CreatedAt.Equal(p.After) && c.ID.String() > p.AfterID.String())
		if c.ReportID == p.ReportID && after {
			c.AuthorID = uuid.Nil
			rows = append(rows, c)
		}
	}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
	}
	return rows, nil
}

func (f *fakeRepo) ListMissingEmbedding(_ context.Context, limit int) ([]ReportText, error) {
	if f.missingErr != nil {
		return nil, f.missingErr
	}
	rows := f.missing
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (f *fakeRepo) SetEmbedding(_ context.Context, id uuid.UUID, e pgvector.Vector) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.embedded == nil {
		f.embedded = map[uuid.UUID]pgvector.Vector{}
	}
	f.embedded[id] = e
	return nil
}

// fakeEmbedder records the texts it receives. Calls from number failFrom on
// (1-based; 0 means never) fail as the real client does when Voyage is down.
type fakeEmbedder struct {
	texts    []string
	failFrom int
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) (pgvector.Vector, error) {
	f.texts = append(f.texts, text)
	if f.failFrom > 0 && len(f.texts) >= f.failFrom {
		return pgvector.Vector{}, embedding.ErrUnavailable
	}
	return pgvector.NewVector([]float32{1, 0, 0}), nil
}

type fakeStore struct {
	files     map[string][]byte
	deleted   []string
	deleteErr error
	urlErr    error
}

func newFakeStore() *fakeStore { return &fakeStore{files: map[string][]byte{}} }

func (f *fakeStore) Put(_ context.Context, key string, data []byte) error {
	f.files[key] = data
	return nil
}

// URL returns a fake signed link that records the key and the expiry.
func (f *fakeStore) URL(_ context.Context, key string, ttl time.Duration) (string, error) {
	if f.urlErr != nil {
		return "", f.urlErr
	}
	return fmt.Sprintf("https://storage.test/%s?expires=%d", key, int(ttl.Seconds())), nil
}

func (f *fakeStore) Delete(_ context.Context, keys ...string) error {
	f.deleted = append(f.deleted, keys...)
	for _, k := range keys {
		delete(f.files, k)
	}
	return f.deleteErr
}

func photo(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func validInput(t *testing.T) CreateReportInput {
	return CreateReportInput{
		AuthorID:     uuid.New(),
		CategorySlug: "limpieza",
		Title:        "Basura acumulada",
		Latitude:     -11.93,
		Longitude:    -77.05,
		Photos:       [][]byte{photo(t), photo(t)},
	}
}

func setup() (*Service, *fakeRepo, *fakeStore) {
	repo := &fakeRepo{categories: map[string]bool{"limpieza": true}}
	store := newFakeStore()
	return NewService(repo, store, &fakeEmbedder{}), repo, store
}

// embedderOf returns the fake embedder of a service built by setup.
func embedderOf(svc *Service) *fakeEmbedder { return svc.embedder.(*fakeEmbedder) }

func TestCreatePublishesCompleteReport(t *testing.T) {
	svc, repo, store := setup()
	in := validInput(t)

	got, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Status != "publicado" || got.ID == uuid.Nil {
		t.Fatalf("result = %+v", got)
	}
	if len(repo.created) != 1 {
		t.Fatalf("created = %d, want 1", len(repo.created))
	}
	r := repo.created[0]
	if r.ID != got.ID || r.AuthorID != in.AuthorID {
		t.Errorf("stored id/author = %v/%v", r.ID, r.AuthorID)
	}
	if len(r.Photos) != 2 || r.Photos[0].Position != 1 || r.Photos[1].Position != 2 {
		t.Fatalf("photos = %+v", r.Photos)
	}
	for _, p := range r.Photos {
		if _, ok := store.files[p.StorageKey]; !ok {
			t.Errorf("photo %q was not stored", p.StorageKey)
		}
		if !strings.HasPrefix(p.StorageKey, "reports/"+got.ID.String()+"/") {
			t.Errorf("key %q is not namespaced by report id", p.StorageKey)
		}
	}
}

func TestCreateOptionalFieldsLeftOut(t *testing.T) {
	svc, repo, _ := setup()
	in := validInput(t)
	in.Details, in.Reference, in.AuthorName = "  ", "", "   "

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	r := repo.created[0]
	if r.Details != "" || r.Reference != "" || r.AuthorName != "" {
		t.Errorf("optional fields = %q %q %q, want empty", r.Details, r.Reference, r.AuthorName)
	}
}

func TestCreateTrimsAuthorName(t *testing.T) {
	svc, repo, _ := setup()
	in := validInput(t)
	in.AuthorName = "  Rosa "

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := repo.created[0].AuthorName; got != "Rosa" {
		t.Errorf("author name = %q, want Rosa", got)
	}
}

func TestCreateRejectsInvalidData(t *testing.T) {
	tests := map[string]func(*CreateReportInput){
		"unknown category":  func(in *CreateReportInput) { in.CategorySlug = "no-existe" },
		"empty category":    func(in *CreateReportInput) { in.CategorySlug = "" },
		"blank title":       func(in *CreateReportInput) { in.Title = "   " },
		"long title":        func(in *CreateReportInput) { in.Title = strings.Repeat("a", 101) },
		"long details":      func(in *CreateReportInput) { in.Details = strings.Repeat("a", 1001) },
		"long reference":    func(in *CreateReportInput) { in.Reference = strings.Repeat("a", 151) },
		"long author name":  func(in *CreateReportInput) { in.AuthorName = strings.Repeat("a", 41) },
		"latitude too high": func(in *CreateReportInput) { in.Latitude = 90.5 },
		"longitude too low": func(in *CreateReportInput) { in.Longitude = -180.1 },
		"no photos":         func(in *CreateReportInput) { in.Photos = nil },
		"four photos":       func(in *CreateReportInput) { in.Photos = append(in.Photos, in.Photos[0], in.Photos[0]) },
		"file is not image": func(in *CreateReportInput) { in.Photos[1] = []byte("not an image") },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			svc, repo, store := setup()
			in := validInput(t)
			mutate(&in)

			_, err := svc.Create(context.Background(), in)
			if !errors.Is(err, ErrInvalidReport) {
				t.Fatalf("err = %v, want ErrInvalidReport", err)
			}
			if len(repo.created) != 0 || len(store.files) != 0 {
				t.Errorf("created %d reports and %d files, want none", len(repo.created), len(store.files))
			}
		})
	}
}

func TestCreateAcceptsBoundaryValues(t *testing.T) {
	svc, _, _ := setup()
	in := validInput(t)
	in.Title = strings.Repeat("ñ", 100) // counted in characters, not bytes
	in.AuthorName = strings.Repeat("a", 40)
	in.Latitude, in.Longitude = 90, -180

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreatePhotoTooLarge(t *testing.T) {
	svc, repo, store := setup()
	in := validInput(t)
	in.Photos[0] = make([]byte, MaxPhotoBytes+1)

	_, err := svc.Create(context.Background(), in)
	if !errors.Is(err, ErrPhotoTooLarge) {
		t.Fatalf("err = %v, want ErrPhotoTooLarge", err)
	}
	if len(repo.created) != 0 || len(store.files) != 0 {
		t.Error("something was created")
	}
}

func TestCreateRateLimit(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	svc, repo, store := setup()
	svc.now = func() time.Time { return now }
	repo.recent = RecentReports{Count: 5, Oldest: now.Add(-7 * time.Minute)}

	_, err := svc.Create(context.Background(), validInput(t))
	var limit *TooManyReportsError
	if !errors.As(err, &limit) || !errors.Is(err, ErrTooManyReports) {
		t.Fatalf("err = %v, want TooManyReportsError", err)
	}
	if limit.RetryAfter != 3*time.Minute {
		t.Errorf("RetryAfter = %v, want 3m", limit.RetryAfter)
	}
	if want := now.Add(-10 * time.Minute); !repo.counted.Equal(want) {
		t.Errorf("window starts at %v, want %v", repo.counted, want)
	}
	if len(repo.created) != 0 || len(store.files) != 0 {
		t.Error("something was created")
	}
}

func TestCreateBelowRateLimit(t *testing.T) {
	svc, repo, _ := setup()
	repo.recent = RecentReports{Count: 4, Oldest: time.Now().Add(-time.Minute)}

	if _, err := svc.Create(context.Background(), validInput(t)); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreateRetryAfterIsAtLeastOneSecond(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	svc, repo, _ := setup()
	svc.now = func() time.Time { return now }
	repo.recent = RecentReports{Count: 5, Oldest: now.Add(-rateLimitWindow)}

	_, err := svc.Create(context.Background(), validInput(t))
	var limit *TooManyReportsError
	if !errors.As(err, &limit) || limit.RetryAfter < time.Second {
		t.Fatalf("err = %v, want RetryAfter >= 1s", err)
	}
}

func TestCreateStorageFailureLeavesNothing(t *testing.T) {
	svc, repo, store := setup()
	in := validInput(t)
	// Fail the second upload: the first must be removed.
	svc.photos = &failSecondPut{fakeStore: store}

	_, err := svc.Create(context.Background(), in)
	if err == nil || errors.Is(err, ErrInvalidReport) {
		t.Fatalf("err = %v, want a server error", err)
	}
	if len(repo.created) != 0 {
		t.Error("a report was created")
	}
	if len(store.files) != 0 {
		t.Errorf("files left behind: %d", len(store.files))
	}
}

type failSecondPut struct {
	*fakeStore
	puts int
}

func (f *failSecondPut) Put(ctx context.Context, key string, data []byte) error {
	f.puts++
	if f.puts == 2 {
		return errors.New("storage down")
	}
	return f.fakeStore.Put(ctx, key, data)
}

func TestCreateDatabaseFailureRemovesPhotos(t *testing.T) {
	svc, repo, store := setup()
	repo.createErr = errors.New("db down")

	_, err := svc.Create(context.Background(), validInput(t))
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("err = %v, want the database error", err)
	}
	if len(store.files) != 0 {
		t.Errorf("files left behind: %d", len(store.files))
	}
	if len(store.deleted) != 2 {
		t.Errorf("deleted = %v, want both photos", store.deleted)
	}
}

func TestCreateCleanupFailureStillReportsOriginalError(t *testing.T) {
	svc, repo, store := setup()
	repo.createErr = errors.New("db down")
	store.deleteErr = errors.New("storage down")

	_, err := svc.Create(context.Background(), validInput(t))
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("err = %v, want the database error", err)
	}
}

func TestCreateCleanupRunsWhenRequestIsCancelled(t *testing.T) {
	svc, repo, store := setup()
	ctx, cancel := context.WithCancel(context.Background())
	repo.createErr = errors.New("cancelled")
	svc.repo = &cancellingRepo{fakeRepo: repo, cancel: cancel}

	if _, err := svc.Create(ctx, validInput(t)); err == nil {
		t.Fatal("expected an error")
	}
	if len(store.deleted) == 0 {
		t.Error("photos were not removed after cancellation")
	}
}

type cancellingRepo struct {
	*fakeRepo
	cancel context.CancelFunc
}

func (c *cancellingRepo) Create(ctx context.Context, r Report) error {
	c.cancel()
	return c.fakeRepo.Create(ctx, r)
}

func storedReport() Report {
	return Report{
		ID:            uuid.MustParse("3f2a1c9e-8b7d-4e6f-a5c4-1b2d3e4f5a6b"),
		AuthorID:      uuid.New(),
		CategorySlug:  "limpieza",
		CategoryLabel: "Limpieza",
		Status:        "publicado",
		CreatedAt:     time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC),
		Title:         "Basura acumulada",
		Details:       "Hace una semana",
		Latitude:      -11.93,
		Longitude:     -77.05,
		Photos: []Photo{
			{Position: 1, StorageKey: "reports/r/1.jpg"},
			{Position: 2, StorageKey: "reports/r/2.jpg"},
		},
	}
}

func TestGetReturnsTheReport(t *testing.T) {
	svc, repo, _ := setup()
	r := storedReport()
	repo.stored = map[uuid.UUID]Report{r.ID: r}

	got, err := svc.Get(context.Background(), r.ID.String(), uuid.New())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != r.ID || got.Title != r.Title || got.Details != r.Details || got.Status != "publicado" {
		t.Errorf("report = %+v", got)
	}
	if got.CategorySlug != "limpieza" || got.CategoryLabel != "Limpieza" || got.Latitude != -11.93 || got.Longitude != -77.05 {
		t.Errorf("category or location = %+v", got)
	}
	if !got.CreatedAt.Equal(r.CreatedAt) || got.Reference != "" || got.AuthorName != "" {
		t.Errorf("date, reference or author = %+v", got)
	}
}

func TestGetSignsPhotosInOrderForOneHour(t *testing.T) {
	svc, repo, _ := setup()
	r := storedReport()
	repo.stored = map[uuid.UUID]Report{r.ID: r}

	got, err := svc.Get(context.Background(), r.ID.String(), uuid.New())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := []string{
		"https://storage.test/reports/r/1.jpg?expires=3600",
		"https://storage.test/reports/r/2.jpg?expires=3600",
	}
	if len(got.Photos) != len(want) {
		t.Fatalf("photos = %+v", got.Photos)
	}
	for i, p := range got.Photos {
		if p.URL != want[i] {
			t.Errorf("photo %d = %q, want %q", i+1, p.URL, want[i])
		}
	}
}

func TestGetUnknownOrMalformedID(t *testing.T) {
	svc, repo, _ := setup()
	repo.stored = map[uuid.UUID]Report{}

	for _, id := range []string{uuid.NewString(), "not-a-uuid", ""} {
		if _, err := svc.Get(context.Background(), id, uuid.New()); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestGetFailures(t *testing.T) {
	svc, repo, store := setup()
	r := storedReport()
	repo.stored = map[uuid.UUID]Report{r.ID: r}

	store.urlErr = errors.New("cannot sign")
	if _, err := svc.Get(context.Background(), r.ID.String(), uuid.New()); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("signing failure: err = %v, want a server error", err)
	}

	store.urlErr = nil
	repo.getErr = errors.New("db down")
	if _, err := svc.Get(context.Background(), r.ID.String(), uuid.New()); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("database failure: err = %v, want a server error", err)
	}
}

func TestReportDTOCarriesNoAuthorID(t *testing.T) {
	typ := reflect.TypeOf(ReportDTO{})
	for i := range typ.NumField() {
		if f := typ.Field(i); f.Type == reflect.TypeOf(uuid.UUID{}) && f.Name != "ID" {
			t.Errorf("ReportDTO.%s is a user-like id; the publisher must never be returned", f.Name)
		}
	}
	if _, ok := typ.FieldByName("AuthorID"); ok {
		t.Error("ReportDTO has AuthorID")
	}
}

func summaries(n int) []ReportSummary {
	base := time.Date(2026, 11, 12, 12, 0, 0, 123456000, time.UTC)
	rows := make([]ReportSummary, n)
	for i := range rows {
		rows[i] = ReportSummary{
			ID:            uuid.New(),
			Title:         fmt.Sprintf("Report %d", i),
			CategorySlug:  "limpieza",
			CategoryLabel: "Limpieza",
			Status:        "publicado",
			CreatedAt:     base.Add(-time.Duration(i) * time.Minute),
			CoverKey:      fmt.Sprintf("reports/%d/1.jpg", i),
		}
	}
	return rows
}

func TestListDefaultPageSizeAndNextCursor(t *testing.T) {
	svc, repo, _ := setup()
	repo.listRows = summaries(25)

	page, err := svc.List(context.Background(), ListReportsInput{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.listParams.Limit != 21 {
		t.Errorf("asked the repository for %d rows, want 21", repo.listParams.Limit)
	}
	if len(page.Items) != 20 {
		t.Fatalf("items = %d, want 20", len(page.Items))
	}
	if page.NextCursor == "" {
		t.Fatal("want a next cursor")
	}
	at, id, err := decodeCursor(page.NextCursor)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	last := repo.listRows[19]
	if !at.Equal(last.CreatedAt) || id != last.ID {
		t.Errorf("cursor = %v/%v, want the 20th item %v/%v", at, id, last.CreatedAt, last.ID)
	}
}

func TestListLastPageHasNoCursor(t *testing.T) {
	svc, repo, _ := setup()
	repo.listRows = summaries(5)

	page, err := svc.List(context.Background(), ListReportsInput{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 5 || page.NextCursor != "" {
		t.Errorf("items = %d, cursor = %q", len(page.Items), page.NextCursor)
	}
}

func TestListEmpty(t *testing.T) {
	svc, _, _ := setup()
	page, err := svc.List(context.Background(), ListReportsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.NextCursor != "" {
		t.Errorf("page = %+v", page)
	}
}

func TestListCursorRoundTripReachesTheRepository(t *testing.T) {
	svc, repo, _ := setup()
	repo.listRows = summaries(3)
	first, err := svc.List(context.Background(), ListReportsInput{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.List(context.Background(), ListReportsInput{Cursor: first.NextCursor, Limit: 2}); err != nil {
		t.Fatal(err)
	}
	p := repo.listParams
	second := repo.listRows[1]
	if !p.HasAfter || !p.After.Equal(second.CreatedAt) || p.AfterID != second.ID {
		t.Errorf("params = %+v, want to continue after %v/%v", p, second.CreatedAt, second.ID)
	}
}

func TestListInvalidInput(t *testing.T) {
	lat, lng := -11.93, -77.05
	badLat, badLng := 91.0, 181.0
	tests := map[string]struct {
		in  ListReportsInput
		err error
	}{
		"limit below range":  {ListReportsInput{Limit: -1}, ErrInvalidReport},
		"limit above range":  {ListReportsInput{Limit: 51}, ErrInvalidReport},
		"only latitude":      {ListReportsInput{Latitude: &lat}, ErrInvalidReport},
		"only longitude":     {ListReportsInput{Longitude: &lng}, ErrInvalidReport},
		"latitude too big":   {ListReportsInput{Latitude: &badLat, Longitude: &lng}, ErrInvalidReport},
		"longitude too big":  {ListReportsInput{Latitude: &lat, Longitude: &badLng}, ErrInvalidReport},
		"cursor not base64":  {ListReportsInput{Cursor: "***"}, ErrInvalidCursor},
		"cursor wrong shape": {ListReportsInput{Cursor: "aGVsbG8"}, ErrInvalidCursor},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			svc, _, _ := setup()
			if _, err := svc.List(context.Background(), tc.in); !errors.Is(err, tc.err) {
				t.Errorf("err = %v, want %v", err, tc.err)
			}
		})
	}
}

func TestListLimitZeroMeansDefaultAndBoundariesAreAccepted(t *testing.T) {
	for _, limit := range []int{0, 1, 50} {
		svc, _, _ := setup()
		if _, err := svc.List(context.Background(), ListReportsInput{Limit: limit}); err != nil {
			t.Errorf("limit %d: %v", limit, err)
		}
	}
}

func TestListPassesTheLocationOnlyWhenGiven(t *testing.T) {
	svc, repo, _ := setup()
	if _, err := svc.List(context.Background(), ListReportsInput{}); err != nil {
		t.Fatal(err)
	}
	if repo.listParams.Latitude != nil || repo.listParams.Longitude != nil {
		t.Errorf("params = %+v, want no location", repo.listParams)
	}

	lat, lng := -11.93, -77.05
	repo.listRows = summaries(1)
	d := 120
	repo.listRows[0].Distance = &d
	page, err := svc.List(context.Background(), ListReportsInput{Latitude: &lat, Longitude: &lng})
	if err != nil {
		t.Fatal(err)
	}
	if repo.listParams.Latitude == nil || *repo.listParams.Latitude != lat || *repo.listParams.Longitude != lng {
		t.Errorf("params = %+v, want the location", repo.listParams)
	}
	if got := page.Items[0].DistanceMeters; got == nil || *got != 120 {
		t.Errorf("distance = %v, want 120", got)
	}
}

func TestListSignsCoversForOneHour(t *testing.T) {
	svc, repo, _ := setup()
	repo.listRows = summaries(1)
	page, err := svc.List(context.Background(), ListReportsInput{})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://storage.test/reports/0/1.jpg?expires=3600"
	if got := page.Items[0].CoverPhoto.URL; got != want {
		t.Errorf("cover = %q, want %q", got, want)
	}
	if page.Items[0].CategoryLabel != "Limpieza" || page.Items[0].Status != "publicado" {
		t.Errorf("item = %+v", page.Items[0])
	}
}

func TestListFailures(t *testing.T) {
	svc, repo, store := setup()
	repo.listRows = summaries(1)
	store.urlErr = errors.New("signer down")
	if _, err := svc.List(context.Background(), ListReportsInput{}); err == nil {
		t.Error("a signing failure must return an error")
	}

	svc, repo, _ = setup()
	repo.listErr = errors.New("db down")
	if _, err := svc.List(context.Background(), ListReportsInput{}); err == nil {
		t.Error("a repository failure must return an error")
	}
}

func hits(n int) []SearchHit {
	rows := summaries(n)
	out := make([]SearchHit, n)
	for i, r := range rows {
		out[i] = SearchHit{ReportSummary: r, ExactDistance: float64(10 * (i + 1))}
	}
	return out
}

func TestSearchSplitsAndEscapesWords(t *testing.T) {
	tests := map[string]struct {
		q    string
		want []string
	}{
		"one word":         {"poste", []string{"poste"}},
		"extra spaces":     {"  fuga   agua \t", []string{"fuga", "agua"}},
		"percent":          {"100%", []string{`100\%`}},
		"underscore":       {"a_b", []string{`a\_b`}},
		"backslash":        {`a\b`, []string{`a\\b`}},
		"accents are kept": {"agüa", []string{"agüa"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			svc, repo, _ := setup()
			if _, err := svc.Search(context.Background(), SearchReportsInput{Query: tc.q}); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(repo.searchParams.Words, tc.want) {
				t.Errorf("words = %q, want %q", repo.searchParams.Words, tc.want)
			}
		})
	}
}

func TestSearchRejectsInvalidInput(t *testing.T) {
	lat, lng := -11.93, -77.05
	badLat := 91.0
	tests := map[string]SearchReportsInput{
		"no query, recent":      {},
		"blank query, recent":   {Query: "   "},
		"distance no location":  {Query: "a", Sort: "distance"},
		"distance only lat":     {Sort: "distance", Latitude: &lat},
		"query too long":        {Query: strings.Repeat("a", 101)},
		"unknown sort":          {Query: "a", Sort: "oldest"},
		"limit below range":     {Query: "a", Limit: -1},
		"limit above range":     {Query: "a", Limit: 51},
		"only longitude":        {Query: "a", Longitude: &lng},
		"latitude out of range": {Query: "a", Latitude: &badLat, Longitude: &lng},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			svc, _, _ := setup()
			if _, err := svc.Search(context.Background(), in); !errors.Is(err, ErrInvalidReport) {
				t.Errorf("err = %v, want ErrInvalidReport", err)
			}
		})
	}
}

func TestSearchAcceptsBoundaries(t *testing.T) {
	lat, lng := -11.93, -77.05
	for name, in := range map[string]SearchReportsInput{
		"100 characters":        {Query: strings.Repeat("ñ", 100)},
		"nearby without words":  {Sort: "distance", Latitude: &lat, Longitude: &lng},
		"nearby blank words":    {Query: "  ", Sort: "distance", Latitude: &lat, Longitude: &lng},
		"recent with location":  {Query: "a", Latitude: &lat, Longitude: &lng},
		"limit 1":               {Query: "a", Limit: 1},
		"limit 50":              {Query: "a", Limit: 50},
		"explicit recent order": {Query: "a", Sort: "recent"},
	} {
		t.Run(name, func(t *testing.T) {
			svc, _, _ := setup()
			if _, err := svc.Search(context.Background(), in); err != nil {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestSearchOrderAndLocationReachTheRepository(t *testing.T) {
	svc, repo, _ := setup()
	lat, lng := -11.93, -77.05
	if _, err := svc.Search(context.Background(), SearchReportsInput{Query: "a"}); err != nil {
		t.Fatal(err)
	}
	p := repo.searchParams
	if p.Order != OrderRecent || p.Latitude != nil || p.Limit != defaultPageSize+1 || p.HasAfter {
		t.Errorf("params = %+v", p)
	}
	if _, err := svc.Search(context.Background(), SearchReportsInput{Sort: "distance", Latitude: &lat, Longitude: &lng}); err != nil {
		t.Fatal(err)
	}
	p = repo.searchParams
	if p.Order != OrderDistance || *p.Latitude != lat || *p.Longitude != lng || len(p.Words) != 0 {
		t.Errorf("params = %+v", p)
	}
}

func TestSearchNextCursorOnlyWhenThereAreMoreRows(t *testing.T) {
	svc, repo, _ := setup()
	repo.searchHits = hits(3)

	page, err := svc.Search(context.Background(), SearchReportsInput{Query: "a", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 || page.NextCursor != "" {
		t.Errorf("items = %d, cursor = %q, want 3 and none", len(page.Items), page.NextCursor)
	}

	page, err = svc.Search(context.Background(), SearchReportsInput{Query: "a", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Errorf("items = %d, cursor = %q, want 2 and a cursor", len(page.Items), page.NextCursor)
	}
}

func TestSearchEmptyResult(t *testing.T) {
	svc, _, _ := setup()
	page, err := svc.Search(context.Background(), SearchReportsInput{Query: "xyz"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.NextCursor != "" {
		t.Errorf("page = %+v", page)
	}
}

func TestSearchCursorRoundTripRecent(t *testing.T) {
	svc, repo, _ := setup()
	repo.searchHits = hits(3)
	first, err := svc.Search(context.Background(), SearchReportsInput{Query: "a", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Search(context.Background(), SearchReportsInput{Query: "a", Limit: 2, Cursor: first.NextCursor}); err != nil {
		t.Fatal(err)
	}
	p, second := repo.searchParams, repo.searchHits[1]
	if !p.HasAfter || !p.After.Equal(second.CreatedAt) || p.AfterID != second.ID {
		t.Errorf("params = %+v, want to continue after %v/%v", p, second.CreatedAt, second.ID)
	}
}

func TestSearchCursorRoundTripDistanceKeepsTheExactValue(t *testing.T) {
	svc, repo, _ := setup()
	lat, lng := -11.93, -77.05
	repo.searchHits = hits(3)
	exact := 123.45678901234567
	repo.searchHits[1].ExactDistance = exact
	in := SearchReportsInput{Sort: "distance", Latitude: &lat, Longitude: &lng, Limit: 2}
	first, err := svc.Search(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	in.Cursor = first.NextCursor
	if _, err := svc.Search(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	p, second := repo.searchParams, repo.searchHits[1]
	if !p.HasAfter || p.AfterDistance != exact || p.AfterID != second.ID || !p.After.Equal(second.CreatedAt) {
		t.Errorf("params = %+v, want the key of %v at %v m", p, second.ID, exact)
	}
}

func TestSearchDistanceTieAcrossPageBoundaryIsNotSkipped(t *testing.T) {
	// Two reports at the same exact distance on both sides of the boundary: the
	// cursor carries the exact distance and the (created_at, id) of the last
	// item, so the repository can tell them apart.
	svc, repo, _ := setup()
	lat, lng := -11.93, -77.05
	repo.searchHits = hits(3)
	for i := range repo.searchHits {
		repo.searchHits[i].ExactDistance = 50
	}
	in := SearchReportsInput{Sort: "distance", Latitude: &lat, Longitude: &lng, Limit: 1}
	first, err := svc.Search(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	in.Cursor = first.NextCursor
	if _, err := svc.Search(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	p, last := repo.searchParams, repo.searchHits[0]
	if p.AfterDistance != 50 || p.AfterID != last.ID || !p.After.Equal(last.CreatedAt) {
		t.Errorf("params = %+v, want the full tie-breaking key of %v", p, last.ID)
	}
}

func TestSearchRejectsCursorsOfAnotherOrderOrOfTheFeed(t *testing.T) {
	svc, repo, _ := setup()
	lat, lng := -11.93, -77.05
	repo.searchHits = hits(3)
	recent, err := svc.Search(context.Background(), SearchReportsInput{Query: "a", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	distance, err := svc.Search(context.Background(), SearchReportsInput{Sort: "distance", Latitude: &lat, Longitude: &lng, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	repo.listRows = summaries(3)
	feed, err := svc.List(context.Background(), ListReportsInput{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]SearchReportsInput{
		"recent cursor with distance": {Sort: "distance", Latitude: &lat, Longitude: &lng, Cursor: recent.NextCursor},
		"distance cursor with recent": {Query: "a", Cursor: distance.NextCursor},
		"feed cursor":                 {Query: "a", Cursor: feed.NextCursor},
		"not base64":                  {Query: "a", Cursor: "***"},
		"not json":                    {Query: "a", Cursor: "aGVsbG8"},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Search(context.Background(), in); !errors.Is(err, ErrInvalidCursor) {
				t.Errorf("err = %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func TestSearchCarriesDistanceAndSignsCovers(t *testing.T) {
	svc, repo, _ := setup()
	repo.searchHits = hits(1)
	d := 80
	repo.searchHits[0].Distance = &d
	page, err := svc.Search(context.Background(), SearchReportsInput{Query: "a"})
	if err != nil {
		t.Fatal(err)
	}
	item := page.Items[0]
	if item.DistanceMeters == nil || *item.DistanceMeters != 80 {
		t.Errorf("distance = %v, want 80", item.DistanceMeters)
	}
	if want := "https://storage.test/reports/0/1.jpg?expires=3600"; item.CoverPhoto.URL != want {
		t.Errorf("cover = %q, want %q", item.CoverPhoto.URL, want)
	}
}

func TestSearchFailures(t *testing.T) {
	svc, repo, store := setup()
	repo.searchErr = errors.New("db down")
	if _, err := svc.Search(context.Background(), SearchReportsInput{Query: "a"}); err == nil || errors.Is(err, ErrInvalidReport) {
		t.Errorf("a repository failure must be a plain error, got %v", err)
	}
	repo.searchErr = nil
	repo.searchHits = hits(1)
	store.urlErr = errors.New("signer down")
	if _, err := svc.Search(context.Background(), SearchReportsInput{Query: "a"}); err == nil {
		t.Error("a signing failure must return an error")
	}
}

func TestListPassesTheFiltersToTheRepository(t *testing.T) {
	svc, repo, _ := setup()
	repo.categories["alumbrado"] = true

	_, err := svc.List(context.Background(), ListReportsInput{Categories: []string{"limpieza", "alumbrado"}, Status: "publicado"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := []string{"limpieza", "alumbrado"}; !reflect.DeepEqual(repo.listParams.Categories, want) {
		t.Errorf("Categories = %v, want %v", repo.listParams.Categories, want)
	}
	if repo.listParams.Status != "publicado" {
		t.Errorf("Status = %q, want publicado", repo.listParams.Status)
	}
}

func TestFiltersDropDuplicateCategories(t *testing.T) {
	svc, repo, _ := setup()

	if _, err := svc.List(context.Background(), ListReportsInput{Categories: []string{"limpieza", "limpieza"}}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := []string{"limpieza"}; !reflect.DeepEqual(repo.listParams.Categories, want) {
		t.Errorf("Categories = %v, want %v", repo.listParams.Categories, want)
	}
}

func TestFiltersAcceptFiveCategoriesAndRejectMore(t *testing.T) {
	svc, repo, _ := setup()
	five := []string{"a", "b", "c", "d", "e"}
	for _, s := range five {
		repo.categories[s] = true
	}
	if _, err := svc.List(context.Background(), ListReportsInput{Categories: five}); err != nil {
		t.Errorf("5 categories: %v", err)
	}
	if _, err := svc.Map(context.Background(), MapInput{Categories: append(five, "f")}); !errors.Is(err, ErrInvalidReport) {
		t.Errorf("6 categories: err = %v, want ErrInvalidReport", err)
	}
}

func TestFiltersRejectUnknownValues(t *testing.T) {
	svc, _, _ := setup()
	ctx := context.Background()

	for name, in := range map[string]ListReportsInput{
		"unknown category": {Categories: []string{"limpieza", "nope"}},
		"unknown status":   {Status: "nope"},
	} {
		if _, err := svc.List(ctx, in); !errors.Is(err, ErrInvalidReport) {
			t.Errorf("List %s: err = %v, want ErrInvalidReport", name, err)
		}
		if _, err := svc.Map(ctx, MapInput{Categories: in.Categories, Status: in.Status}); !errors.Is(err, ErrInvalidReport) {
			t.Errorf("Map %s: err = %v, want ErrInvalidReport", name, err)
		}
	}
}

func TestFiltersAcceptEveryStatus(t *testing.T) {
	svc, _, _ := setup()
	for _, status := range Statuses {
		if _, err := svc.Map(context.Background(), MapInput{Status: status}); err != nil {
			t.Errorf("status %q: %v", status, err)
		}
	}
}

func TestFiltersDoNotCheckCategoriesWhenThereAreNone(t *testing.T) {
	svc, repo, _ := setup()
	ctx := context.Background()

	if _, err := svc.List(ctx, ListReportsInput{Status: "publicado"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Map(ctx, MapInput{}); err != nil {
		t.Fatal(err)
	}
	if repo.unknownCalled != 0 {
		t.Errorf("UnknownCategories called %d times, want 0", repo.unknownCalled)
	}
}

func TestFilterFailures(t *testing.T) {
	svc, repo, _ := setup()
	repo.unknownErr = errors.New("db down")
	_, err := svc.List(context.Background(), ListReportsInput{Categories: []string{"limpieza"}})
	if err == nil || errors.Is(err, ErrInvalidReport) {
		t.Errorf("err = %v, want an internal error", err)
	}
}

func TestMapAsksForTheCapAndMapsPinsWithoutTheAuthor(t *testing.T) {
	svc, repo, _ := setup()
	id := uuid.New()
	repo.pins = []ReportPin{{
		ID: id, Title: "Poste sin luz", CategorySlug: "alumbrado", CategoryLabel: "Alumbrado",
		Status: "publicado", Latitude: -11.93, Longitude: -77.05,
	}}

	got, err := svc.Map(context.Background(), MapInput{Status: "publicado"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if repo.pinsParams.Limit != 500 {
		t.Errorf("Limit = %d, want 500", repo.pinsParams.Limit)
	}
	if repo.pinsParams.Status != "publicado" {
		t.Errorf("Status = %q, want publicado", repo.pinsParams.Status)
	}
	want := []ReportPinDTO{{
		ID: id, Title: "Poste sin luz", CategorySlug: "alumbrado", CategoryLabel: "Alumbrado",
		Status: "publicado", Latitude: -11.93, Longitude: -77.05,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pins = %+v, want %+v", got, want)
	}
}

func TestMapEmptyIsNotNil(t *testing.T) {
	svc, _, _ := setup()
	got, err := svc.Map(context.Background(), MapInput{})
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("Map = %v, %v; want an empty non-nil slice", got, err)
	}
}

func TestMapRepositoryFailure(t *testing.T) {
	svc, repo, _ := setup()
	repo.pinsErr = errors.New("db down")
	if _, err := svc.Map(context.Background(), MapInput{}); err == nil || errors.Is(err, ErrInvalidReport) {
		t.Errorf("err = %v, want an internal error", err)
	}
}

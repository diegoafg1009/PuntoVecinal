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

func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (Report, error) {
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
	return NewService(repo, store), repo, store
}

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

	got, err := svc.Get(context.Background(), r.ID.String())
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

	got, err := svc.Get(context.Background(), r.ID.String())
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
		if _, err := svc.Get(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestGetFailures(t *testing.T) {
	svc, repo, store := setup()
	r := storedReport()
	repo.stored = map[uuid.UUID]Report{r.ID: r}

	store.urlErr = errors.New("cannot sign")
	if _, err := svc.Get(context.Background(), r.ID.String()); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("signing failure: err = %v, want a server error", err)
	}

	store.urlErr = nil
	repo.getErr = errors.New("db down")
	if _, err := svc.Get(context.Background(), r.ID.String()); err == nil || errors.Is(err, ErrNotFound) {
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

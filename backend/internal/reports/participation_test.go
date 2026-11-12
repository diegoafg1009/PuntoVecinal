package reports

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// authoredReport registers a report by author in the fake repository.
func authoredReport(repo *fakeRepo, author uuid.UUID) uuid.UUID {
	id := uuid.New()
	if repo.authors == nil {
		repo.authors = map[uuid.UUID]uuid.UUID{}
	}
	repo.authors[id] = author
	return id
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// --- Publishing and backfill ---

func TestCreateEmbedsOnlyTheTitleAndDetails(t *testing.T) {
	svc, repo, _ := setup()
	in := validInput(t)
	in.Title, in.Details = "  Poste sin luz ", " Desde el lunes "
	in.AuthorName, in.Reference = "Rosa", "Jr. Ejemplo cdra. 3"

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	texts := embedderOf(svc).texts
	if want := []string{"Poste sin luz\nDesde el lunes"}; !reflect.DeepEqual(texts, want) {
		t.Fatalf("embedder received %q, want %q", texts, want)
	}
	if strings.Contains(texts[0], "Rosa") || strings.Contains(texts[0], "Jr. Ejemplo") {
		t.Errorf("the text carries the author name or the reference: %q", texts[0])
	}
	if repo.created[0].Embedding == nil {
		t.Error("the report was stored without its embedding")
	}
}

func TestCreateWithoutDetailsEmbedsTheTitleAlone(t *testing.T) {
	svc, _, _ := setup()
	in := validInput(t)
	in.Title = "Poste sin luz"

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if want := []string{"Poste sin luz"}; !reflect.DeepEqual(embedderOf(svc).texts, want) {
		t.Errorf("embedder received %q, want %q", embedderOf(svc).texts, want)
	}
}

func TestCreateWithAFailingEmbedderStillPublishes(t *testing.T) {
	svc, repo, store := setup()
	embedderOf(svc).failFrom = 1

	got, err := svc.Create(context.Background(), validInput(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Status != "publicado" || len(repo.created) != 1 || len(store.files) != 2 {
		t.Fatalf("result %+v, %d reports, %d files", got, len(repo.created), len(store.files))
	}
	if repo.created[0].Embedding != nil {
		t.Error("the report has an embedding although the embedder failed")
	}
}

func TestBackfillRoundEmbedsTheMissingReports(t *testing.T) {
	repo := &fakeRepo{missing: []ReportText{
		{ID: uuid.New(), Title: "Poste sin luz", Details: "Desde el lunes"},
		{ID: uuid.New(), Title: "Basura"},
	}}
	emb := &fakeEmbedder{}

	if got := backfillRound(context.Background(), repo, emb, discardLogger()); got != 2 {
		t.Fatalf("embedded %d reports, want 2", got)
	}
	if want := []string{"Poste sin luz\nDesde el lunes", "Basura"}; !reflect.DeepEqual(emb.texts, want) {
		t.Errorf("embedder received %q, want %q", emb.texts, want)
	}
	for _, r := range repo.missing {
		if _, ok := repo.embedded[r.ID]; !ok {
			t.Errorf("report %q was not given an embedding", r.Title)
		}
	}
}

func TestBackfillRoundStopsAtTheFirstUnavailable(t *testing.T) {
	repo := &fakeRepo{missing: []ReportText{
		{ID: uuid.New(), Title: "uno"}, {ID: uuid.New(), Title: "dos"}, {ID: uuid.New(), Title: "tres"},
	}}
	emb := &fakeEmbedder{failFrom: 2}

	if got := backfillRound(context.Background(), repo, emb, discardLogger()); got != 1 {
		t.Fatalf("embedded %d reports, want 1", got)
	}
	if len(emb.texts) != 2 {
		t.Errorf("embedder was called %d times, want 2 (it must not go on after the failure)", len(emb.texts))
	}
	if len(repo.embedded) != 1 {
		t.Errorf("%d reports were given an embedding, want 1", len(repo.embedded))
	}
}

func TestBackfillRoundTakesAtMostOneBatch(t *testing.T) {
	repo := &fakeRepo{}
	for i := 0; i < backfillBatch+5; i++ {
		repo.missing = append(repo.missing, ReportText{ID: uuid.New(), Title: "x"})
	}
	if got := backfillRound(context.Background(), repo, &fakeEmbedder{}, discardLogger()); got != backfillBatch {
		t.Errorf("embedded %d reports, want %d", got, backfillBatch)
	}
}

func TestBackfillRoundSurvivesDatabaseErrors(t *testing.T) {
	repo := &fakeRepo{missingErr: errors.New("db down")}
	if got := backfillRound(context.Background(), repo, &fakeEmbedder{}, discardLogger()); got != 0 {
		t.Errorf("embedded %d reports, want 0", got)
	}

	repo = &fakeRepo{missing: []ReportText{{ID: uuid.New(), Title: "x"}}, setErr: errors.New("db down")}
	if got := backfillRound(context.Background(), repo, &fakeEmbedder{}, discardLogger()); got != 0 {
		t.Errorf("embedded %d reports, want 0", got)
	}
}

// --- Similar ---

func similarInput() SimilarReportsInput {
	return SimilarReportsInput{Title: "Poste sin luz", Details: "Desde el lunes", Latitude: -11.93, Longitude: -77.05}
}

func TestSimilarRejectsInvalidInput(t *testing.T) {
	tests := map[string]func(*SimilarReportsInput){
		"blank title":        func(in *SimilarReportsInput) { in.Title = "   " },
		"long title":         func(in *SimilarReportsInput) { in.Title = strings.Repeat("a", 101) },
		"long details":       func(in *SimilarReportsInput) { in.Details = strings.Repeat("a", 1001) },
		"latitude too high":  func(in *SimilarReportsInput) { in.Latitude = 91 },
		"longitude too high": func(in *SimilarReportsInput) { in.Longitude = 181 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			svc, repo, _ := setup()
			in := similarInput()
			mutate(&in)

			if _, err := svc.Similar(context.Background(), in); !errors.Is(err, ErrInvalidReport) {
				t.Fatalf("err = %v, want ErrInvalidReport", err)
			}
			if n := len(embedderOf(svc).texts); n != 0 {
				t.Errorf("the embedder was called %d times for invalid input", n)
			}
			if repo.similarParams.MaxDistance != 0 {
				t.Error("the repository was queried for invalid input")
			}
		})
	}
}

func TestSimilarWithAnUnavailableEmbedderIsComparisonUnavailable(t *testing.T) {
	svc, repo, _ := setup()
	embedderOf(svc).failFrom = 1

	if _, err := svc.Similar(context.Background(), similarInput()); !errors.Is(err, ErrComparisonUnavailable) {
		t.Fatalf("err = %v, want ErrComparisonUnavailable", err)
	}
	if repo.similarParams.MaxDistance != 0 {
		t.Error("the repository was queried although nothing could be compared")
	}
}

func TestSimilarSendsOnlyTheTextAndSearchesNearby(t *testing.T) {
	svc, repo, _ := setup()
	in := similarInput()
	in.Title, in.Details = " Poste sin luz ", " Desde el lunes "
	repo.similar = []SimilarReport{{
		ID: uuid.New(), Title: "No hay alumbrado", CategorySlug: "alumbrado", CategoryLabel: "Alumbrado",
		Status: "publicado", CreatedAt: time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC),
		CoverKey: "reports/a/1.jpg", Distance: 80, AffectedCount: 14,
	}}

	got, err := svc.Similar(context.Background(), in)
	if err != nil {
		t.Fatalf("Similar: %v", err)
	}

	if want := []string{"Poste sin luz\nDesde el lunes"}; !reflect.DeepEqual(embedderOf(svc).texts, want) {
		t.Errorf("embedder received %q, want %q", embedderOf(svc).texts, want)
	}
	p := repo.similarParams
	if p.Latitude != -11.93 || p.Longitude != -77.05 || p.MaxDistance != maxSimilarDistance {
		t.Errorf("params = %+v", p)
	}
	if len(got) != 1 {
		t.Fatalf("items = %+v", got)
	}
	item := got[0]
	if item.ID != repo.similar[0].ID || item.CategoryLabel != "Alumbrado" || item.DistanceMeters != 80 || item.AffectedCount != 14 {
		t.Errorf("item = %+v", item)
	}
	if want := "https://storage.test/reports/a/1.jpg?expires=3600"; item.CoverPhoto.URL != want {
		t.Errorf("cover = %q, want %q", item.CoverPhoto.URL, want)
	}
}

func TestSimilarWithoutResultsIsAnEmptyList(t *testing.T) {
	svc, _, _ := setup()
	got, err := svc.Similar(context.Background(), similarInput())
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("got %v, %v; want an empty, non-nil list", got, err)
	}
}

func TestSimilarFailures(t *testing.T) {
	svc, repo, store := setup()
	repo.similar = []SimilarReport{{ID: uuid.New(), CoverKey: "reports/a/1.jpg"}}

	store.urlErr = errors.New("cannot sign")
	if _, err := svc.Similar(context.Background(), similarInput()); err == nil || errors.Is(err, ErrComparisonUnavailable) {
		t.Errorf("signing failure: err = %v, want a server error", err)
	}

	store.urlErr = nil
	repo.similarErr = errors.New("db down")
	if _, err := svc.Similar(context.Background(), similarInput()); err == nil || errors.Is(err, ErrComparisonUnavailable) {
		t.Errorf("database failure: err = %v, want a server error", err)
	}
}

// --- Support ---

func TestSupportUnknownOrMalformedReport(t *testing.T) {
	svc, _, _ := setup()
	for _, id := range []string{uuid.NewString(), "not-a-uuid", ""} {
		if _, err := svc.Support(context.Background(), id, uuid.New()); !errors.Is(err, ErrNotFound) {
			t.Errorf("Support(%q) err = %v, want ErrNotFound", id, err)
		}
		if _, err := svc.Unsupport(context.Background(), id, uuid.New()); !errors.Is(err, ErrNotFound) {
			t.Errorf("Unsupport(%q) err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestTheAuthorCannotChangeSupport(t *testing.T) {
	svc, repo, _ := setup()
	author := uuid.New()
	id := authoredReport(repo, author)

	if _, err := svc.Support(context.Background(), id.String(), author); !errors.Is(err, ErrIsAuthor) {
		t.Errorf("Support err = %v, want ErrIsAuthor", err)
	}
	if _, err := svc.Unsupport(context.Background(), id.String(), author); !errors.Is(err, ErrIsAuthor) {
		t.Errorf("Unsupport err = %v, want ErrIsAuthor", err)
	}
	if len(repo.supports[id]) != 0 {
		t.Errorf("supports = %v, want none", repo.supports[id])
	}
}

func TestSupportAndUnsupportAreIdempotent(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())
	neighbor := uuid.New()
	ctx := context.Background()

	steps := []struct {
		name  string
		do    func() (ReportSupportDTO, error)
		count int
		state string
	}{
		{"mark", func() (ReportSupportDTO, error) { return svc.Support(ctx, id.String(), neighbor) }, 2, "supporter"},
		{"mark again", func() (ReportSupportDTO, error) { return svc.Support(ctx, id.String(), neighbor) }, 2, "supporter"},
		{"unmark", func() (ReportSupportDTO, error) { return svc.Unsupport(ctx, id.String(), neighbor) }, 1, "none"},
		{"unmark again", func() (ReportSupportDTO, error) { return svc.Unsupport(ctx, id.String(), neighbor) }, 1, "none"},
	}
	for _, step := range steps {
		got, err := step.do()
		if err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if got.AffectedCount != step.count || got.Support != step.state {
			t.Errorf("%s = %+v, want count %d and state %q", step.name, got, step.count, step.state)
		}
	}
}

func TestTwoNeighborsMakeThreeAffected(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())

	if _, err := svc.Support(context.Background(), id.String(), uuid.New()); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Support(context.Background(), id.String(), uuid.New())
	if err != nil || got.AffectedCount != 3 {
		t.Errorf("got %+v, %v; want 3 affected", got, err)
	}
}

func TestSupportFailures(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())

	repo.supportErr = errors.New("db down")
	if _, err := svc.Support(context.Background(), id.String(), uuid.New()); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want a server error", err)
	}
	repo.supportErr, repo.authorErr = nil, errors.New("db down")
	if _, err := svc.Support(context.Background(), id.String(), uuid.New()); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want a server error", err)
	}
}

// --- Get: the caller's support state ---

func TestGetTellsTheSupportState(t *testing.T) {
	tests := map[string]struct {
		isAuthor, supported bool
		want                string
	}{
		"author":    {true, false, SupportAuthor},
		"supporter": {false, true, SupportSupporter},
		"none":      {false, false, SupportNone},
		// The author always counts, whatever the support rows say.
		"author with a support row": {true, true, SupportAuthor},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			svc, repo, _ := setup()
			r := storedReport()
			r.IsAuthor, r.Supported, r.AffectedCount, r.ContributionCount = tc.isAuthor, tc.supported, 4, 2
			repo.stored = map[uuid.UUID]Report{r.ID: r}

			got, err := svc.Get(context.Background(), r.ID.String(), uuid.New())
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.Support != tc.want || got.AffectedCount != 4 || got.ContributionCount != 2 {
				t.Errorf("got support %q, affected %d, aportes %d", got.Support, got.AffectedCount, got.ContributionCount)
			}
		})
	}
}

// --- Contribute ---

func contributionInput(t *testing.T, report uuid.UUID, author uuid.UUID) ContributionInput {
	t.Helper()
	return ContributionInput{ReportID: report.String(), AuthorID: author, Text: "Hoy en la mañana seguía igual"}
}

func TestContributeRejectsInvalidData(t *testing.T) {
	tests := map[string]func(*ContributionInput){
		"empty text":     func(in *ContributionInput) { in.Text = "" },
		"blank text":     func(in *ContributionInput) { in.Text = "  \n " },
		"long text":      func(in *ContributionInput) { in.Text = strings.Repeat("a", 1201) },
		"long name":      func(in *ContributionInput) { in.AuthorName = strings.Repeat("a", 41) },
		"not an image":   func(in *ContributionInput) { in.Photo = []byte("not an image") },
		"empty photo":    func(in *ContributionInput) { in.Photo = []byte{} },
		"photo is a gif": func(in *ContributionInput) { in.Photo = []byte("GIF89a....") },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			svc, repo, store := setup()
			in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
			mutate(&in)

			if _, err := svc.Contribute(context.Background(), in); !errors.Is(err, ErrInvalidReport) {
				t.Fatalf("err = %v, want ErrInvalidReport", err)
			}
			if len(repo.contribs) != 0 || len(store.files) != 0 || len(repo.supports) != 0 {
				t.Errorf("stored %d aportes, %d files and %d supports; want nothing", len(repo.contribs), len(store.files), len(repo.supports))
			}
		})
	}
}

func TestContributeAcceptsTheLimits(t *testing.T) {
	svc, repo, _ := setup()
	in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
	in.Text, in.AuthorName = strings.Repeat("á", 1200), strings.Repeat("ñ", 40)

	if _, err := svc.Contribute(context.Background(), in); err != nil {
		t.Fatalf("Contribute: %v", err)
	}
}

func TestContributePhotoTooLarge(t *testing.T) {
	svc, repo, store := setup()
	in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
	in.Photo = make([]byte, MaxPhotoBytes+1)

	if _, err := svc.Contribute(context.Background(), in); !errors.Is(err, ErrPhotoTooLarge) {
		t.Fatalf("err = %v, want ErrPhotoTooLarge", err)
	}
	if len(repo.contribs) != 0 || len(store.files) != 0 {
		t.Error("something was stored")
	}
}

func TestContributeUnknownOrMalformedReport(t *testing.T) {
	for _, id := range []string{uuid.NewString(), "not-a-uuid", ""} {
		svc, repo, store := setup()
		in := ContributionInput{ReportID: id, AuthorID: uuid.New(), Text: "hola"}
		in.Photo = photo(t)

		if _, err := svc.Contribute(context.Background(), in); !errors.Is(err, ErrNotFound) {
			t.Errorf("Contribute(%q) err = %v, want ErrNotFound", id, err)
		}
		if len(repo.contribs) != 0 || len(store.files) != 0 {
			t.Errorf("Contribute(%q) stored something", id)
		}
	}
}

func TestContributeRateLimit(t *testing.T) {
	svc, repo, store := setup()
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	repo.contribRecent = RecentReports{Count: 10, Oldest: now.Add(-4 * time.Minute)}
	in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
	in.Photo = photo(t)

	_, err := svc.Contribute(context.Background(), in)
	var limit *TooManyContributionsError
	if !errors.As(err, &limit) || !errors.Is(err, ErrTooManyContributions) {
		t.Fatalf("err = %v, want TooManyContributionsError", err)
	}
	if limit.RetryAfter != 6*time.Minute {
		t.Errorf("RetryAfter = %s, want 6m", limit.RetryAfter)
	}
	if want := now.Add(-10 * time.Minute); !repo.counted.Equal(want) {
		t.Errorf("window starts at %s, want %s", repo.counted, want)
	}
	if len(repo.contribs) != 0 || len(store.files) != 0 {
		t.Error("something was stored over the limit")
	}

	// Nine in the window still leave room for one.
	repo.contribRecent = RecentReports{Count: 9, Oldest: now.Add(-4 * time.Minute)}
	if _, err := svc.Contribute(context.Background(), in); err != nil {
		t.Errorf("below the limit: %v", err)
	}
}

func TestContributeStoresACleanedPhotoUnderContributions(t *testing.T) {
	svc, repo, store := setup()
	id := authoredReport(repo, uuid.New())
	in := contributionInput(t, id, uuid.New())
	in.Photo = photo(t)

	got, err := svc.Contribute(context.Background(), in)
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	if len(store.files) != 1 {
		t.Fatalf("stored %d files, want 1", len(store.files))
	}
	stored := repo.contribs[0]
	if want := "contributions/" + stored.ID.String() + ".jpg"; stored.PhotoKey != want {
		t.Errorf("photo key = %q, want %q", stored.PhotoKey, want)
	}
	data, ok := store.files[stored.PhotoKey]
	if !ok || len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Errorf("the stored file is not a re-encoded JPEG")
	}
	if want := "https://storage.test/" + stored.PhotoKey + "?expires=3600"; got.Contribution.PhotoURL != want {
		t.Errorf("photo link = %q, want %q", got.Contribution.PhotoURL, want)
	}
}

func TestContributeWithoutPhoto(t *testing.T) {
	svc, repo, store := setup()
	in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
	in.AuthorName = "  Rosa "

	got, err := svc.Contribute(context.Background(), in)
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	if len(store.files) != 0 || repo.contribs[0].PhotoKey != "" || got.Contribution.PhotoURL != "" {
		t.Errorf("a photo appeared: %d files, key %q, link %q", len(store.files), repo.contribs[0].PhotoKey, got.Contribution.PhotoURL)
	}
	if got.Contribution.AuthorName != "Rosa" || got.Contribution.Text != in.Text || got.Contribution.ID == uuid.Nil {
		t.Errorf("contribution = %+v", got.Contribution)
	}
}

func TestContributionWithoutNameIsAnonymous(t *testing.T) {
	svc, repo, _ := setup()
	in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
	in.AuthorName = "   "

	got, err := svc.Contribute(context.Background(), in)
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	if got.Contribution.AuthorName != "" || repo.contribs[0].AuthorName != "" {
		t.Errorf("author name = %q / %q, want none", got.Contribution.AuthorName, repo.contribs[0].AuthorName)
	}
	if repo.contribs[0].AuthorID != in.AuthorID {
		t.Error("the session that added the aporte is not recorded")
	}
}

func TestAFailedInsertRemovesThePhotoAndMarksNoSupport(t *testing.T) {
	svc, repo, store := setup()
	id := authoredReport(repo, uuid.New())
	in := contributionInput(t, id, uuid.New())
	in.Photo = photo(t)
	repo.contribErr = errors.New("db down")

	if _, err := svc.Contribute(context.Background(), in); err == nil {
		t.Fatal("err = nil, want a server error")
	}
	if len(store.files) != 0 || len(store.deleted) != 1 || !strings.HasPrefix(store.deleted[0], "contributions/") {
		t.Errorf("files = %d, deleted = %v; want the stored photo removed", len(store.files), store.deleted)
	}
	if len(repo.contribs) != 0 || len(repo.supports[id]) != 0 {
		t.Error("the aporte or the support was saved")
	}
}

func TestEveryContributionMarksSupportForANonAuthor(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())
	// Three neighbors already support it: 4 are affected.
	for i := 0; i < 3; i++ {
		_ = repo.Support(context.Background(), id, uuid.New())
	}
	neighbor := uuid.New()

	first, err := svc.Contribute(context.Background(), contributionInput(t, id, neighbor))
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	if first.AffectedCount != 5 || first.Support != SupportSupporter || !repo.contribMarked {
		t.Errorf("first = %+v (marked %v), want 5 affected and «supporter»", first, repo.contribMarked)
	}

	// Already supporting: the aporte is added and the count stays.
	second, err := svc.Contribute(context.Background(), contributionInput(t, id, neighbor))
	if err != nil {
		t.Fatalf("Contribute again: %v", err)
	}
	if second.AffectedCount != 5 || second.Support != SupportSupporter || len(repo.contribs) != 2 {
		t.Errorf("second = %+v with %d aportes, want 5 affected and 2 aportes", second, len(repo.contribs))
	}
}

func TestAnAuthorsContributionInsertsNoSupport(t *testing.T) {
	svc, repo, _ := setup()
	author := uuid.New()
	id := authoredReport(repo, author)

	got, err := svc.Contribute(context.Background(), contributionInput(t, id, author))
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	if repo.contribMarked || len(repo.supports[id]) != 0 {
		t.Error("the author's aporte inserted a support")
	}
	if got.AffectedCount != 1 || got.Support != SupportAuthor || len(repo.contribs) != 1 {
		t.Errorf("got %+v with %d aportes, want 1 affected, «author» and 1 aporte", got, len(repo.contribs))
	}
}

func TestContributeSigningFailure(t *testing.T) {
	svc, repo, store := setup()
	in := contributionInput(t, authoredReport(repo, uuid.New()), uuid.New())
	in.Photo = photo(t)
	store.urlErr = errors.New("cannot sign")

	if _, err := svc.Contribute(context.Background(), in); err == nil {
		t.Error("err = nil, want a server error")
	}
}

// --- Contributions ---

func seedContributions(repo *fakeRepo, report uuid.UUID, n int) {
	for i := 0; i < n; i++ {
		repo.contribs = append(repo.contribs, Contribution{
			ID:        uuid.New(),
			ReportID:  report,
			Text:      "aporte",
			CreatedAt: time.Date(2026, 11, 12, 10, 0, i, 0, time.UTC),
		})
	}
}

func TestContributionsPagesWithACursor(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())
	seedContributions(repo, id, 25)
	seedContributions(repo, uuid.New(), 3) // another report's aportes

	first, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String()})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Items) != 20 || first.NextCursor == "" {
		t.Fatalf("first page has %d items and cursor %q, want 20 and a cursor", len(first.Items), first.NextCursor)
	}

	second, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String(), Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 5 || second.NextCursor != "" {
		t.Fatalf("second page has %d items and cursor %q, want 5 and none", len(second.Items), second.NextCursor)
	}

	// Oldest first, every aporte exactly once.
	var all []ContributionDTO
	all = append(append(all, first.Items...), second.Items...)
	for i, c := range all {
		if c.ID != repo.contribs[i].ID {
			t.Fatalf("item %d is %v, want %v", i, c.ID, repo.contribs[i].ID)
		}
	}
}

func TestContributionsEmpty(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())

	page, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String()})
	if err != nil || page.Items == nil || len(page.Items) != 0 || page.NextCursor != "" {
		t.Errorf("page = %+v, err = %v; want an empty list and no cursor", page, err)
	}
}

func TestContributionsInvalidPaging(t *testing.T) {
	svc, repo, _ := setup()
	id := authoredReport(repo, uuid.New())

	for _, limit := range []int{-1, 51} {
		_, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String(), Limit: limit})
		if !errors.Is(err, ErrInvalidReport) {
			t.Errorf("limit %d: err = %v, want ErrInvalidReport", limit, err)
		}
	}
	for _, cursor := range []string{"not a cursor", "bm90LWEtY3Vyc29y"} {
		_, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String(), Cursor: cursor})
		if !errors.Is(err, ErrInvalidCursor) {
			t.Errorf("cursor %q: err = %v, want ErrInvalidCursor", cursor, err)
		}
	}
}

func TestContributionsUnknownOrMalformedReport(t *testing.T) {
	svc, _, _ := setup()
	for _, id := range []string{uuid.NewString(), "not-a-uuid", ""} {
		if _, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id}); !errors.Is(err, ErrNotFound) {
			t.Errorf("Contributions(%q) err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestContributionsSignPhotosAndFailures(t *testing.T) {
	svc, repo, store := setup()
	id := authoredReport(repo, uuid.New())
	seedContributions(repo, id, 2)
	repo.contribs[0].PhotoKey = "contributions/a.jpg"
	repo.contribs[0].AuthorName = "Rosa"

	page, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String()})
	if err != nil {
		t.Fatalf("Contributions: %v", err)
	}
	if want := "https://storage.test/contributions/a.jpg?expires=3600"; page.Items[0].PhotoURL != want {
		t.Errorf("photo link = %q, want %q", page.Items[0].PhotoURL, want)
	}
	if page.Items[0].AuthorName != "Rosa" || page.Items[1].AuthorName != "" || page.Items[1].PhotoURL != "" {
		t.Errorf("items = %+v", page.Items)
	}

	store.urlErr = errors.New("cannot sign")
	if _, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String()}); err == nil {
		t.Error("signing failure: err = nil, want a server error")
	}
	store.urlErr, repo.listContribs = nil, errors.New("db down")
	if _, err := svc.Contributions(context.Background(), ListContributionsInput{ReportID: id.String()}); err == nil {
		t.Error("database failure: err = nil, want a server error")
	}
}

// --- Privacy ---

func TestParticipationDTOsCarryNoUserID(t *testing.T) {
	for _, v := range []any{SimilarReportDTO{}, ReportSupportDTO{}, ContributionDTO{}, CreatedContributionDTO{}, ContributionPageDTO{}} {
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			if name := strings.ToLower(typ.Field(i).Name); strings.Contains(name, "author") && name != "authorname" {
				t.Errorf("%s has the field %s", typ.Name(), typ.Field(i).Name)
			}
		}
	}
	// A similar report never carries the author name nor the details either.
	typ := reflect.TypeOf(SimilarReportDTO{})
	for _, name := range []string{"AuthorName", "Details"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("SimilarReportDTO has the field %s", name)
		}
	}
}

func TestComparableText(t *testing.T) {
	tests := []struct{ title, details, want string }{
		{"Poste sin luz", "", "Poste sin luz"},
		{"Poste sin luz", "Desde el lunes", "Poste sin luz\nDesde el lunes"},
	}
	for _, tc := range tests {
		if got := comparableText(tc.title, tc.details); got != tc.want {
			t.Errorf("comparableText(%q, %q) = %q, want %q", tc.title, tc.details, got, tc.want)
		}
	}
}

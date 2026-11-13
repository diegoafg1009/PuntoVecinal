package reports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// send makes a request with an optional body and content type.
func send(h http.Handler, method, path, authorization, contentType string, body []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// --- POST /reports/similar ---

func similarBody() []byte {
	return []byte(`{"title":"Poste sin luz","details":"Desde el lunes","latitude":-11.93,"longitude":-77.05}`)
}

func postSimilar(h http.Handler, authorization string, body []byte) *httptest.ResponseRecorder {
	return send(h, http.MethodPost, "/reports/similar", authorization, "application/json", body)
}

func TestFindSimilarReportsReturnsItems(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.similar = []SimilarReport{
		{ID: uuid.New(), Title: "No hay alumbrado", CategorySlug: "alumbrado", CategoryLabel: "Alumbrado",
			Status: "publicado", CreatedAt: time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC),
			CoverKey: "reports/a/1.jpg", Distance: 80, AffectedCount: 14},
		{ID: uuid.New(), Title: "Poste caído", CategorySlug: "alumbrado", CategoryLabel: "Alumbrado",
			Status: "derivado", CoverKey: "reports/b/1.jpg", Distance: 120, AffectedCount: 1},
	}

	w := postSimilar(h, "Bearer good", similarBody())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var got struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(got.Items))
	}
	first := got.Items[0]
	if first["id"] != repo.similar[0].ID.String() || first["title"] != "No hay alumbrado" || first["status"] != "publicado" {
		t.Errorf("first = %v", first)
	}
	if first["distanceMeters"] != float64(80) || first["affectedCount"] != float64(14) {
		t.Errorf("distance %v, affected %v", first["distanceMeters"], first["affectedCount"])
	}
	if first["coverPhoto"] != "https://storage.test/reports/a/1.jpg?expires=3600" {
		t.Errorf("coverPhoto = %v", first["coverPhoto"])
	}
	if first["createdAt"] != "2026-11-12T08:00:00Z" {
		t.Errorf("createdAt = %v", first["createdAt"])
	}
	if cat := first["category"].(map[string]any); cat["slug"] != "alumbrado" || cat["label"] != "Alumbrado" {
		t.Errorf("category = %v", cat)
	}
	for _, item := range got.Items {
		for _, key := range []string{"authorName", "details", "authorId", "reference"} {
			if _, ok := item[key]; ok {
				t.Errorf("item carries %q: %v", key, item)
			}
		}
	}
	if strings.Contains(w.Body.String(), testUserID) {
		t.Errorf("response reveals a user id: %s", w.Body)
	}
	if p := repo.similarParams; p.Latitude != -11.93 || p.Longitude != -77.05 {
		t.Errorf("location = %v,%v", p.Latitude, p.Longitude)
	}
}

func TestFindSimilarReportsWithoutResultsIsAnEmptyList(t *testing.T) {
	h, _, _ := newRouter(t)
	w := postSimilar(h, "Bearer good", similarBody())
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"items":[]}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestFindSimilarReportsUnavailableIs503(t *testing.T) {
	h, repo, _ := newRouterWith(t, &fakeEmbedder{failFrom: 1})
	repo.similar = []SimilarReport{{ID: uuid.New()}}

	w := postSimilar(h, "Bearer good", similarBody())
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503, body = %s", w.Code, w.Body)
	}
	if strings.TrimSpace(w.Body.String()) != "comparison unavailable" {
		t.Errorf("body = %q", w.Body)
	}
}

func TestFindSimilarReportsBadRequests(t *testing.T) {
	long := strings.Repeat("a", 101)
	tests := map[string]string{
		"blank title":       `{"title":"  ","latitude":-11.93,"longitude":-77.05}`,
		"long title":        `{"title":"` + long + `","latitude":-11.93,"longitude":-77.05}`,
		"long details":      `{"title":"x","details":"` + strings.Repeat("a", 1001) + `","latitude":-11.93,"longitude":-77.05}`,
		"latitude range":    `{"title":"x","latitude":91,"longitude":-77.05}`,
		"longitude range":   `{"title":"x","latitude":-11.93,"longitude":181}`,
		"no location":       `{"title":"x"}`,
		"malformed json":    `{"title":`,
		"empty body":        ``,
		"no title at all":   `{"latitude":-11.93,"longitude":-77.05}`,
		"title is a number": `{"title":5,"latitude":-11.93,"longitude":-77.05}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			h, _, _ := newRouter(t)
			w := postSimilar(h, "Bearer good", []byte(body))
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400, body = %s", w.Code, w.Body)
			}
		})
	}
}

func TestSimilarRouteIsNotTakenAsAReportID(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.getErr = errors.New("Get must not be called")

	if w := postSimilar(h, "Bearer good", similarBody()); w.Code != http.StatusOK {
		t.Fatalf("POST /reports/similar: status = %d, body = %s", w.Code, w.Body)
	}
	// A report id still reaches GetReport.
	repo.getErr = nil
	if w := getReport(h, "Bearer good", uuid.NewString()); w.Code != http.StatusNotFound {
		t.Errorf("GET /reports/{id}: status = %d, want 404", w.Code)
	}
}

func TestFindSimilarReportsRequiresAValidToken(t *testing.T) {
	h, _, _ := newRouter(t)
	for _, authorization := range []string{"", "Bearer bad"} {
		if w := postSimilar(h, authorization, similarBody()); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
	}
}

func TestFindSimilarReportsServerError(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.similarErr = errors.New("db down")
	if w := postSimilar(h, "Bearer good", similarBody()); w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestFindSimilarReportsBodyIsCapped(t *testing.T) {
	h, _, _ := newRouter(t)
	huge := `{"title":"x","details":"` + strings.Repeat("a", 2*maxJSONBytes) + `","latitude":-11.93,"longitude":-77.05}`
	if w := postSimilar(h, "Bearer good", []byte(huge)); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a body over the cap", w.Code)
	}
}

// --- PUT / DELETE /reports/{id}/support ---

func supportRequest(h http.Handler, method, authorization, id string) *httptest.ResponseRecorder {
	return send(h, method, "/reports/"+id+"/support", authorization, "", nil)
}

type supportBody struct {
	AffectedCount int    `json:"affectedCount"`
	Support       string `json:"support"`
}

func TestSupportAndUnsupportReturnTheCountAndState(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	_ = repo.Support(context.Background(), id, uuid.New())

	tests := []struct {
		method string
		count  int
		state  string
	}{
		{http.MethodPut, 3, "supporter"},
		{http.MethodPut, 3, "supporter"},
		{http.MethodDelete, 2, "none"},
		{http.MethodDelete, 2, "none"},
	}
	for _, tc := range tests {
		w := supportRequest(h, tc.method, "Bearer good", id.String())
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", tc.method, w.Code, w.Body)
		}
		var got supportBody
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.AffectedCount != tc.count || got.Support != tc.state {
			t.Errorf("%s: got %+v, want count %d and state %q", tc.method, got, tc.count, tc.state)
		}
		if strings.Contains(w.Body.String(), testUserID) {
			t.Errorf("%s: response reveals a user id: %s", tc.method, w.Body)
		}
	}
}

func TestSupportUnknownReportIs404(t *testing.T) {
	h, _, _ := newRouter(t)
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
			if w := supportRequest(h, method, "Bearer good", id); w.Code != http.StatusNotFound {
				t.Errorf("%s %q: status = %d, want 404", method, id, w.Code)
			}
		}
	}
}

func TestTheAuthorGetsA409OnSupport(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.MustParse(testUserID))

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if w := supportRequest(h, method, "Bearer good", id.String()); w.Code != http.StatusConflict {
			t.Errorf("%s: status = %d, want 409, body = %s", method, w.Code, w.Body)
		}
	}
	if len(repo.supports[id]) != 0 {
		t.Error("the author's request changed the support")
	}
}

func TestSupportRequiresAValidToken(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		for _, authorization := range []string{"", "Bearer bad"} {
			if w := supportRequest(h, method, authorization, id.String()); w.Code != http.StatusUnauthorized {
				t.Errorf("%s with %q: status = %d, want 401", method, authorization, w.Code)
			}
		}
	}
	if len(repo.supports[id]) != 0 {
		t.Error("an unauthenticated request changed the support")
	}
}

func TestSupportServerError(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	repo.supportErr = errors.New("db down")
	if w := supportRequest(h, http.MethodPut, "Bearer good", id.String()); w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// --- POST /reports/{id}/contributions ---

type contributionPart struct{ name, value string }

// contributionForm builds a multipart body with the text fields and then the
// given photos, all under the field name "photo".
func contributionForm(t *testing.T, fields []contributionPart, photos ...[]byte) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, f := range fields {
		if err := mw.WriteField(f.name, f.value); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range photos {
		fw, err := mw.CreateFormFile("photo", "photo.jpg")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write(p)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes(), mw.FormDataContentType()
}

func postContribution(h http.Handler, authorization, id string, body []byte, contentType string) *httptest.ResponseRecorder {
	return send(h, http.MethodPost, "/reports/"+id+"/contributions", authorization, contentType, body)
}

type contributionBody struct {
	Contribution  map[string]any `json:"contribution"`
	AffectedCount int            `json:"affectedCount"`
	Support       string         `json:"support"`
}

func TestCreateContributionWithTextOnly(t *testing.T) {
	h, repo, store := newRouter(t)
	id := authoredReport(repo, uuid.New())
	body, ct := contributionForm(t, []contributionPart{{"text", "Hoy en la mañana seguía igual"}})

	w := postContribution(h, "Bearer good", id.String(), body, ct)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var got contributionBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	c := got.Contribution
	if c["text"] != "Hoy en la mañana seguía igual" || c["id"] == "" || c["createdAt"] == nil {
		t.Errorf("contribution = %v", c)
	}
	if v, ok := c["photoUrl"]; !ok || v != nil {
		t.Errorf("photoUrl = %v (present %v), want null", v, ok)
	}
	if v, ok := c["authorName"]; !ok || v != nil {
		t.Errorf("authorName = %v (present %v), want null", v, ok)
	}
	if got.AffectedCount != 2 || got.Support != "supporter" {
		t.Errorf("affected %d, support %q; want 2 and «supporter»", got.AffectedCount, got.Support)
	}
	if len(store.files) != 0 || len(repo.contribs) != 1 || repo.contribs[0].AuthorID.String() != testUserID {
		t.Errorf("stored %d files and %d aportes", len(store.files), len(repo.contribs))
	}
	if strings.Contains(w.Body.String(), testUserID) {
		t.Errorf("response reveals the user id: %s", w.Body)
	}
}

func TestCreateContributionWithPhotoAndName(t *testing.T) {
	h, repo, store := newRouter(t)
	id := authoredReport(repo, uuid.New())
	body, ct := contributionForm(t, []contributionPart{{"text", "Se ve en la foto"}, {"authorName", "Rosa"}}, photo(t))

	w := postContribution(h, "Bearer good", id.String(), body, ct)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var got contributionBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Contribution["authorName"] != "Rosa" {
		t.Errorf("authorName = %v", got.Contribution["authorName"])
	}
	url, _ := got.Contribution["photoUrl"].(string)
	if !strings.HasPrefix(url, "https://storage.test/contributions/") || !strings.HasSuffix(url, ".jpg?expires=3600") {
		t.Errorf("photoUrl = %q", url)
	}
	if len(store.files) != 1 {
		t.Errorf("stored %d files, want 1", len(store.files))
	}
}

func TestAnAuthorsContributionKeepsTheCount(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.MustParse(testUserID))
	body, ct := contributionForm(t, []contributionPart{{"text", "Sigue igual"}})

	w := postContribution(h, "Bearer good", id.String(), body, ct)
	var got contributionBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusCreated {
		t.Fatalf("status = %d, err = %v, body = %s", w.Code, err, w.Body)
	}
	if got.AffectedCount != 1 || got.Support != "author" {
		t.Errorf("affected %d, support %q; want 1 and «author»", got.AffectedCount, got.Support)
	}
}

func TestCreateContributionBadRequests(t *testing.T) {
	type request struct {
		body []byte
		ct   string
	}
	build := func(fields []contributionPart, photos ...[]byte) request {
		body, ct := contributionForm(t, fields, photos...)
		return request{body, ct}
	}
	tests := map[string]request{
		"two photos":    build([]contributionPart{{"text", "hola"}}, photo(t), photo(t)),
		"no text":       build(nil, photo(t)),
		"blank text":    build([]contributionPart{{"text", "   "}}),
		"long text":     build([]contributionPart{{"text", strings.Repeat("a", 1201)}}),
		"long name":     build([]contributionPart{{"text", "hola"}, {"authorName", strings.Repeat("a", 41)}}),
		"not an image":  build([]contributionPart{{"text", "hola"}}, []byte("not an image")),
		"not multipart": {[]byte(`{"text":"hola"}`), "application/json"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h, repo, store := newRouter(t)
			id := authoredReport(repo, uuid.New())

			w := postContribution(h, "Bearer good", id.String(), tc.body, tc.ct)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body)
			}
			if len(repo.contribs) != 0 || len(store.files) != 0 || len(repo.supports) != 0 {
				t.Error("something was stored")
			}
		})
	}
}

func TestCreateContributionPhotoTooLarge(t *testing.T) {
	h, repo, store := newRouter(t)
	id := authoredReport(repo, uuid.New())
	body, ct := contributionForm(t, []contributionPart{{"text", "hola"}}, make([]byte, MaxPhotoBytes+1))

	w := postContribution(h, "Bearer good", id.String(), body, ct)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413, body = %s", w.Code, w.Body)
	}
	if len(repo.contribs) != 0 || len(store.files) != 0 {
		t.Error("something was stored")
	}
}

func TestCreateContributionRequestTooLarge(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	// Over the cap of an aporte (one photo), although each part is small.
	body, ct := contributionForm(t, []contributionPart{{"text", "hola"}}, make([]byte, maxContributionBytes), make([]byte, 1))

	w := postContribution(h, "Bearer good", id.String(), body, ct)
	if w.Code != http.StatusRequestEntityTooLarge && w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 413 or 400", w.Code)
	}
	if len(repo.contribs) != 0 {
		t.Error("an aporte was stored")
	}
}

func TestCreateContributionRateLimited(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	repo.contribRecent = RecentReports{Count: 10, Oldest: time.Now().Add(-9 * time.Minute)}
	body, ct := contributionForm(t, []contributionPart{{"text", "hola"}})

	w := postContribution(h, "Bearer good", id.String(), body, ct)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429, body = %s", w.Code, w.Body)
	}
	if ra := w.Header().Get("Retry-After"); ra != "60" && ra != "61" {
		t.Errorf("Retry-After = %q, want about 60", ra)
	}
	if len(repo.contribs) != 0 {
		t.Error("an aporte was stored over the limit")
	}
}

func TestCreateContributionUnknownReportIs404(t *testing.T) {
	h, repo, store := newRouter(t)
	body, ct := contributionForm(t, []contributionPart{{"text", "hola"}}, photo(t))

	for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
		if w := postContribution(h, "Bearer good", id, body, ct); w.Code != http.StatusNotFound {
			t.Errorf("id %q: status = %d, want 404", id, w.Code)
		}
	}
	if len(repo.contribs) != 0 || len(store.files) != 0 {
		t.Error("something was stored")
	}
}

func TestCreateContributionRequiresAValidToken(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	body, ct := contributionForm(t, []contributionPart{{"text", "hola"}})

	for _, authorization := range []string{"", "Bearer bad"} {
		if w := postContribution(h, authorization, id.String(), body, ct); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
	}
	if len(repo.contribs) != 0 || len(repo.supports) != 0 {
		t.Error("an unauthenticated request stored something")
	}
}

func TestCreateContributionServerError(t *testing.T) {
	h, repo, store := newRouter(t)
	id := authoredReport(repo, uuid.New())
	repo.contribErr = errors.New("db down")
	body, ct := contributionForm(t, []contributionPart{{"text", "hola"}}, photo(t))

	if w := postContribution(h, "Bearer good", id.String(), body, ct); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if len(store.files) != 0 {
		t.Error("the photo was left behind")
	}
}

// --- GET /reports/{id}/contributions ---

func getContributions(h http.Handler, authorization, id, query string) *httptest.ResponseRecorder {
	return send(h, http.MethodGet, "/reports/"+id+"/contributions"+query, authorization, "", nil)
}

func TestListContributionsReturnsThemOldestFirst(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.MustParse(testUserID))
	seedContributions(repo, id, 3)
	repo.contribs[0].AuthorName = "Rosa"
	repo.contribs[0].PhotoKey = "contributions/a.jpg"
	repo.contribs[0].AuthorID = uuid.MustParse(testUserID)

	w := getContributions(h, "Bearer good", id.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var got listBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 || got.NextCursor != nil {
		t.Fatalf("items = %d, nextCursor = %v", len(got.Items), got.NextCursor)
	}
	for i, item := range got.Items {
		if item["id"] != repo.contribs[i].ID.String() {
			t.Errorf("item %d id = %v, want %v", i, item["id"], repo.contribs[i].ID)
		}
	}
	first, second := got.Items[0], got.Items[1]
	if first["authorName"] != "Rosa" || first["photoUrl"] != "https://storage.test/contributions/a.jpg?expires=3600" {
		t.Errorf("first = %v", first)
	}
	if v, ok := second["authorName"]; !ok || v != nil {
		t.Errorf("authorName = %v (present %v), want null", v, ok)
	}
	if v, ok := second["photoUrl"]; !ok || v != nil {
		t.Errorf("photoUrl = %v (present %v), want null", v, ok)
	}
	if strings.Contains(w.Body.String(), testUserID) || strings.Contains(strings.ToLower(w.Body.String()), "authorid") {
		t.Errorf("response reveals the author id: %s", w.Body)
	}
}

func TestListContributionsPagesWithACursor(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	seedContributions(repo, id, 25)

	var first listBody
	w := getContributions(h, "Bearer good", id.String(), "")
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil || len(first.Items) != 20 || first.NextCursor == nil {
		t.Fatalf("first page: err = %v, %+v", err, first)
	}
	var second listBody
	w = getContributions(h, "Bearer good", id.String(), "?cursor="+*first.NextCursor)
	if err := json.Unmarshal(w.Body.Bytes(), &second); err != nil || len(second.Items) != 5 || second.NextCursor != nil {
		t.Fatalf("second page: err = %v, %+v", err, second)
	}
}

func TestListContributionsEmpty(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	w := getContributions(h, "Bearer good", id.String(), "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"items":[],"nextCursor":null}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestListContributionsBadRequestsAndNotFound(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())

	for _, query := range []string{"?cursor=garbage", "?limit=0", "?limit=51"} {
		if w := getContributions(h, "Bearer good", id.String(), query); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, w.Code)
		}
	}
	for _, other := range []string{uuid.NewString(), "not-a-uuid"} {
		if w := getContributions(h, "Bearer good", other, ""); w.Code != http.StatusNotFound {
			t.Errorf("id %q: status = %d, want 404", other, w.Code)
		}
	}
}

func TestListContributionsRequiresAValidToken(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	for _, authorization := range []string{"", "Bearer bad"} {
		if w := getContributions(h, authorization, id.String(), ""); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
	}
}

func TestListContributionsServerError(t *testing.T) {
	h, repo, _ := newRouter(t)
	id := authoredReport(repo, uuid.New())
	repo.listContribs = errors.New("db down")
	if w := getContributions(h, "Bearer good", id.String(), ""); w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// --- The report and the cards carry the affected count ---

func TestGetReportCarriesTheParticipationFields(t *testing.T) {
	tests := map[string]struct {
		isAuthor, supported bool
		want                string
	}{
		"author":    {true, false, "author"},
		"supporter": {false, true, "supporter"},
		"none":      {false, false, "none"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h, repo, _ := newRouter(t)
			stored := storedReport()
			stored.AffectedCount, stored.ContributionCount = 4, 2
			stored.IsAuthor, stored.Supported = tc.isAuthor, tc.supported
			repo.stored = map[uuid.UUID]Report{stored.ID: stored}

			w := getReport(h, "Bearer good", stored.ID.String())
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body)
			}
			var got map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got["affectedCount"] != float64(4) || got["contributionCount"] != float64(2) || got["support"] != tc.want {
				t.Errorf("affected %v, aportes %v, support %v", got["affectedCount"], got["contributionCount"], got["support"])
			}
		})
	}
}

func TestFeedAndSearchItemsCarryTheAffectedCount(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.listRows = summaries(2)
	repo.listRows[0].AffectedCount, repo.listRows[1].AffectedCount = 1, 14
	repo.searchHits = hits(2)
	repo.searchHits[0].AffectedCount, repo.searchHits[1].AffectedCount = 6, 1

	for name, tc := range map[string]struct {
		w    *httptest.ResponseRecorder
		want []float64
	}{
		"feed":   {listReports(h, "Bearer good", ""), []float64{1, 14}},
		"search": {searchReports(h, "Bearer good", "?q=poste"), []float64{6, 1}},
	} {
		if tc.w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", name, tc.w.Code, tc.w.Body)
		}
		var got listBody
		if err := json.Unmarshal(tc.w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		for i, item := range got.Items {
			if item["affectedCount"] != tc.want[i] {
				t.Errorf("%s item %d: affectedCount = %v, want %v", name, i, item["affectedCount"], tc.want[i])
			}
		}
	}
}

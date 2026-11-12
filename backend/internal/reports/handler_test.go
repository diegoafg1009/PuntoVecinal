package reports

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

const testUserID = "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f"

// fakeVerifier accepts only the token "good".
type fakeVerifier struct{}

func (fakeVerifier) VerifyRequest(r *http.Request) (platformauth.Claims, error) {
	if r.Header.Get("Authorization") != "Bearer good" {
		return platformauth.Claims{}, platformauth.ErrUnauthorized
	}
	return platformauth.Claims{UserID: testUserID, IsAnonymous: true}, nil
}

func newRouter(t *testing.T) (http.Handler, *fakeRepo, *fakeStore) {
	t.Helper()
	repo := &fakeRepo{categories: map[string]bool{"limpieza": true}}
	store := newFakeStore()
	r := chi.NewRouter()
	if err := Register(r, fakeVerifier{}, repo, store); err != nil {
		t.Fatal(err)
	}
	return r, repo, store
}

type part struct{ name, value string }

// form builds a multipart body: text fields first, then the photos.
func form(t *testing.T, fields []part, photos ...[]byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, f := range fields {
		if err := mw.WriteField(f.name, f.value); err != nil {
			t.Fatal(err)
		}
	}
	for i, p := range photos {
		fw, err := mw.CreateFormFile("photos", "photo"+string(rune('1'+i))+".jpg")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write(p)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, mw.FormDataContentType()
}

func validFields() []part {
	return []part{
		{"category", "limpieza"},
		{"title", "Basura acumulada"},
		{"latitude", "-11.93"},
		{"longitude", "-77.05"},
	}
}

func post(h http.Handler, authorization string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/reports", body)
	r.Header.Set("Content-Type", contentType)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func without(fields []part, name string) []part {
	var out []part
	for _, f := range fields {
		if f.name != name {
			out = append(out, f)
		}
	}
	return out
}

func TestCreateReportPublishes(t *testing.T) {
	h, repo, store := newRouter(t)
	fields := append(validFields(), part{"authorName", "Rosa"}, part{"details", "Hace una semana"})
	body, ct := form(t, fields, photo(t), photo(t))

	w := post(h, "Bearer good", body, ct)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "publicado" || got["id"] == "" {
		t.Errorf("body = %v", got)
	}
	if _, leaked := got["authorId"]; leaked || strings.Contains(w.Body.String(), testUserID) {
		t.Errorf("response reveals the author id: %s", w.Body.String())
	}

	if len(repo.created) != 1 || len(store.files) != 2 {
		t.Fatalf("created %d reports and %d files", len(repo.created), len(store.files))
	}
	r := repo.created[0]
	if r.AuthorID.String() != testUserID || r.AuthorName != "Rosa" || r.Details != "Hace una semana" {
		t.Errorf("stored report = %+v", r)
	}
	if r.Latitude != -11.93 || r.Longitude != -77.05 {
		t.Errorf("location = %v,%v", r.Latitude, r.Longitude)
	}
}

func TestCreateReportOptionalFieldsLeftOut(t *testing.T) {
	h, _, _ := newRouter(t)
	body, ct := form(t, validFields(), photo(t))

	if w := post(h, "Bearer good", body, ct); w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
	}
}

func TestCreateReportRequiresAValidToken(t *testing.T) {
	for _, authorization := range []string{"", "Bearer bad"} {
		h, repo, store := newRouter(t)
		body, ct := form(t, validFields(), photo(t))

		if w := post(h, authorization, body, ct); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
		if len(repo.created) != 0 || len(store.files) != 0 {
			t.Errorf("Authorization %q: something was created", authorization)
		}
	}
}

func TestCreateReportBadRequests(t *testing.T) {
	big := func() []part { return append(without(validFields(), "title"), part{"title", strings.Repeat("a", 101)}) }
	tests := map[string]struct {
		fields []part
		photos [][]byte
	}{
		"unknown category":  {append(without(validFields(), "category"), part{"category", "no-existe"}), [][]byte{photo(t)}},
		"blank title":       {append(without(validFields(), "title"), part{"title", "   "}), [][]byte{photo(t)}},
		"missing title":     {without(validFields(), "title"), [][]byte{photo(t)}},
		"title too long":    {big(), [][]byte{photo(t)}},
		"missing latitude":  {without(validFields(), "latitude"), [][]byte{photo(t)}},
		"latitude not num":  {append(without(validFields(), "latitude"), part{"latitude", "abc"}), [][]byte{photo(t)}},
		"latitude too high": {append(without(validFields(), "latitude"), part{"latitude", "91"}), [][]byte{photo(t)}},
		"no photos":         {validFields(), nil},
		"four photos":       {validFields(), [][]byte{photo(t), photo(t), photo(t), photo(t)}},
		"not an image":      {validFields(), [][]byte{[]byte("hello")}},
		"name too long":     {append(validFields(), part{"authorName", strings.Repeat("a", 41)}), [][]byte{photo(t)}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h, repo, store := newRouter(t)
			body, ct := form(t, tt.fields, tt.photos...)

			w := post(h, "Bearer good", body, ct)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", w.Code, w.Body.String())
			}
			if len(repo.created) != 0 || len(store.files) != 0 {
				t.Error("something was created")
			}
		})
	}
}

func TestCreateReportNotMultipart(t *testing.T) {
	h, _, _ := newRouter(t)
	w := post(h, "Bearer good", bytes.NewBufferString(`{"title":"x"}`), "application/json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateReportPhotoTooLarge(t *testing.T) {
	h, repo, store := newRouter(t)
	body, ct := form(t, validFields(), make([]byte, MaxPhotoBytes+1))

	if w := post(h, "Bearer good", body, ct); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if len(repo.created) != 0 || len(store.files) != 0 {
		t.Error("something was created")
	}
}

func TestCreateReportRequestTooLarge(t *testing.T) {
	h, _, _ := newRouter(t)
	// Three photos at the limit plus a large unknown part go over the cap.
	big := make([]byte, MaxPhotoBytes)
	fields := append(validFields(), part{"junk", strings.Repeat("x", 2<<20)})
	body, ct := form(t, fields, big, big, big)

	if w := post(h, "Bearer good", body, ct); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
}

func TestCreateReportRateLimited(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.recent = RecentReports{Count: 5, Oldest: time.Now().Add(-9 * time.Minute)}
	body, ct := form(t, validFields(), photo(t))

	w := post(h, "Bearer good", body, ct)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if ra := w.Header().Get("Retry-After"); ra != "60" && ra != "61" {
		t.Errorf("Retry-After = %q, want about 60", ra)
	}
	if len(repo.created) != 0 {
		t.Error("a report was created")
	}
}

func TestCreateReportServerError(t *testing.T) {
	h, repo, store := newRouter(t)
	repo.createErr = errors.New("db down")
	body, ct := form(t, validFields(), photo(t))

	if w := post(h, "Bearer good", body, ct); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if len(store.files) != 0 {
		t.Error("photos were left behind")
	}
}

func getReport(h http.Handler, authorization, id string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/reports/"+id, nil)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestGetReportReturnsTheReport(t *testing.T) {
	h, repo, _ := newRouter(t)
	stored := storedReport()
	stored.AuthorID = uuid.MustParse(testUserID)
	repo.stored = map[uuid.UUID]Report{stored.ID: stored}

	w := getReport(h, "Bearer good", stored.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", w.Code, w.Body.String())
	}

	var got struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Details    *string `json:"details"`
		Reference  *string `json:"reference"`
		AuthorName *string `json:"authorName"`
		Status     string  `json:"status"`
		CreatedAt  string  `json:"createdAt"`
		Category   struct {
			Slug, Label string
		} `json:"category"`
		Location struct {
			Latitude, Longitude float64
		} `json:"location"`
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != stored.ID.String() || got.Title != stored.Title || got.Status != "publicado" {
		t.Errorf("report = %+v", got)
	}
	if got.Details == nil || *got.Details != "Hace una semana" || got.Reference != nil || got.AuthorName != nil {
		t.Errorf("details %v, reference %v, author %v", got.Details, got.Reference, got.AuthorName)
	}
	if got.Category.Slug != "limpieza" || got.Category.Label != "Limpieza" {
		t.Errorf("category = %+v", got.Category)
	}
	if got.Location.Latitude != -11.93 || got.Location.Longitude != -77.05 {
		t.Errorf("location = %+v", got.Location)
	}
	if got.CreatedAt != "2026-11-12T10:00:00Z" {
		t.Errorf("createdAt = %q", got.CreatedAt)
	}
	if len(got.Photos) != 2 || !strings.Contains(got.Photos[0].URL, "1.jpg") || !strings.Contains(got.Photos[1].URL, "2.jpg") {
		t.Errorf("photos = %+v", got.Photos)
	}

	var raw map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if _, ok := raw["authorName"]; !ok {
		t.Error("authorName must be present (null) so the page can show «Vecino anónimo»")
	}
	if strings.Contains(w.Body.String(), testUserID) {
		t.Errorf("response reveals the publisher's user id: %s", w.Body.String())
	}
}

func TestGetReportNotFound(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.stored = map[uuid.UUID]Report{}

	for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
		if w := getReport(h, "Bearer good", id); w.Code != http.StatusNotFound {
			t.Errorf("id %q: status = %d, want 404", id, w.Code)
		}
	}
}

func TestGetReportRequiresAValidToken(t *testing.T) {
	h, repo, _ := newRouter(t)
	stored := storedReport()
	repo.stored = map[uuid.UUID]Report{stored.ID: stored}

	for _, authorization := range []string{"", "Bearer bad"} {
		if w := getReport(h, authorization, stored.ID.String()); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
	}
}

func TestGetReportServerError(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.getErr = errors.New("db down")

	if w := getReport(h, "Bearer good", uuid.NewString()); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func listReports(h http.Handler, authorization, query string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/reports"+query, nil)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

type listBody struct {
	Items []map[string]any `json:"items"`
	// NextCursor is a pointer to tell null from a value.
	NextCursor *string `json:"nextCursor"`
}

func TestListReportsReturnsItemsInOrder(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.listRows = summaries(3)
	repo.listRows[0].Reference = "frente al mercado"

	w := listReports(h, "Bearer good", "")
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
		if item["id"] != repo.listRows[i].ID.String() {
			t.Errorf("item %d id = %v, want %v", i, item["id"], repo.listRows[i].ID)
		}
	}
	first := got.Items[0]
	if first["reference"] != "frente al mercado" || first["status"] != "publicado" {
		t.Errorf("first = %v", first)
	}
	if got.Items[1]["reference"] != nil {
		t.Errorf("reference = %v, want null", got.Items[1]["reference"])
	}
	for _, item := range got.Items {
		if v, ok := item["distanceMeters"]; !ok || v != nil {
			t.Errorf("distanceMeters = %v (present %v), want null", v, ok)
		}
	}
	cover := first["coverPhoto"].(map[string]any)["url"]
	if cover != "https://storage.test/reports/0/1.jpg?expires=3600" {
		t.Errorf("cover = %v", cover)
	}
	if cat := first["category"].(map[string]any); cat["slug"] != "limpieza" || cat["label"] != "Limpieza" {
		t.Errorf("category = %v", cat)
	}
}

func TestListReportsNextCursorAndLimit(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.listRows = summaries(3)

	w := listReports(h, "Bearer good", "?limit=2")
	var first listBody
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.NextCursor == nil || *first.NextCursor == "" {
		t.Fatalf("first page = %+v", first)
	}

	w = listReports(h, "Bearer good", "?limit=2&cursor="+*first.NextCursor)
	if w.Code != http.StatusOK {
		t.Fatalf("second page status = %d, body = %s", w.Code, w.Body)
	}
	if !repo.listParams.HasAfter || repo.listParams.AfterID != repo.listRows[1].ID {
		t.Errorf("params = %+v", repo.listParams)
	}
}

func TestListReportsDistance(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.listRows = summaries(1)
	d := 120
	repo.listRows[0].Distance = &d

	w := listReports(h, "Bearer good", "?latitude=-11.93&longitude=-77.05")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var got listBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Items[0]["distanceMeters"] != float64(120) {
		t.Errorf("distanceMeters = %v, want 120", got.Items[0]["distanceMeters"])
	}
}

func TestListReportsEmpty(t *testing.T) {
	h, _, _ := newRouter(t)
	w := listReports(h, "Bearer good", "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"items":[],"nextCursor":null}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestListReportsBadRequests(t *testing.T) {
	for name, query := range map[string]string{
		"bad cursor":         "?cursor=***",
		"limit zero":         "?limit=0",
		"limit too large":    "?limit=51",
		"limit not a number": "?limit=abc",
		"only latitude":      "?latitude=-11.9",
		"only longitude":     "?longitude=-77.0",
		"latitude range":     "?latitude=91&longitude=-77",
		"longitude range":    "?latitude=-11&longitude=181",
	} {
		t.Run(name, func(t *testing.T) {
			h, _, _ := newRouter(t)
			if w := listReports(h, "Bearer good", query); w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400, body = %s", w.Code, w.Body)
			}
		})
	}
}

func TestListReportsRequiresAValidToken(t *testing.T) {
	h, _, _ := newRouter(t)
	for _, authorization := range []string{"", "Bearer bad"} {
		if w := listReports(h, authorization, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
	}
}

func TestListReportsNeverExposesTheAuthorOrDetails(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.listRows = summaries(2)

	body := strings.ToLower(listReports(h, "Bearer good", "").Body.String())
	for _, forbidden := range []string{"authorname", "authorid", "author_id", "details", "userid", testUserID} {
		if strings.Contains(body, forbidden) {
			t.Errorf("response contains %q: %s", forbidden, body)
		}
	}
}

func TestListReportsServerError(t *testing.T) {
	h, repo, _ := newRouter(t)
	repo.listErr = errors.New("db down")
	if w := listReports(h, "Bearer good", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

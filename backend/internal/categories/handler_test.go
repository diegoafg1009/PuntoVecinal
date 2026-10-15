package categories

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"

	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

// fakeVerifier accepts only the token "good".
type fakeVerifier struct{}

func (fakeVerifier) VerifyRequest(r *http.Request) (platformauth.Claims, error) {
	if r.Header.Get("Authorization") != "Bearer good" {
		return platformauth.Claims{}, platformauth.ErrUnauthorized
	}
	return platformauth.Claims{UserID: "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f", IsAnonymous: true}, nil
}

type fakeRepo struct {
	rows []Category
	err  error
}

func (f fakeRepo) List(context.Context) ([]Category, error) {
	return f.rows, f.err
}

var fiveRows = []Category{
	{Slug: "limpieza", Label: "Limpieza"},
	{Slug: "alumbrado", Label: "Alumbrado"},
	{Slug: "pistas-y-veredas", Label: "Pistas y veredas"},
	{Slug: "agua-y-desague", Label: "Agua y desagüe"},
	{Slug: "otro", Label: "Otro"},
}

func newRouter(t *testing.T, repo Repository) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	if err := Register(r, fakeVerifier{}, repo); err != nil {
		t.Fatal(err)
	}
	return r
}

func get(h http.Handler, authorization string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/categories", nil)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

type item struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

func TestListCategoriesReturnsFiveInOrderOnEveryCall(t *testing.T) {
	h := newRouter(t, fakeRepo{rows: fiveRows})

	var first, second []item
	for _, got := range []*[]item{&first, &second} {
		w := get(h, "Bearer good")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if err := json.Unmarshal(w.Body.Bytes(), got); err != nil {
			t.Fatal(err)
		}
	}

	want := make([]item, len(fiveRows))
	for i, row := range fiveRows {
		want[i] = item{Slug: row.Slug, Label: row.Label}
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("items = %v, want %v", first, want)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("second call = %v, want same as first %v", second, first)
	}
}

func TestListCategoriesRequiresAValidToken(t *testing.T) {
	h := newRouter(t, fakeRepo{rows: fiveRows})
	for _, authorization := range []string{"", "Bearer bad"} {
		if w := get(h, authorization); w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", authorization, w.Code)
		}
	}
}

func TestListCategoriesFailsWithoutAPartialList(t *testing.T) {
	h := newRouter(t, fakeRepo{rows: fiveRows[:2], err: errors.New("db down")})

	w := get(h, "Bearer good")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if json.Valid(w.Body.Bytes()) {
		var items []item
		if json.Unmarshal(w.Body.Bytes(), &items) == nil && len(items) > 0 {
			t.Fatalf("body has a list: %s", w.Body.String())
		}
	}
}

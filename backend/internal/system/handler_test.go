package system

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

type rejectAll struct{}

func (rejectAll) VerifyRequest(*http.Request) (auth.Claims, error) {
	return auth.Claims{}, auth.ErrUnauthorized
}

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func healthz(t *testing.T, db Pinger) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	if err := Register(r, rejectAll{}, db); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil)) // no token
	return w
}

func TestHealthzIsPublicAndReady(t *testing.T) {
	if w := healthz(t, fakeDB{}); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHealthzReportsUnavailableDatabase(t *testing.T) {
	w := healthz(t, fakeDB{err: errors.New("down")})
	if w.Code != http.StatusServiceUnavailable || w.Body.String() != "database unavailable" {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

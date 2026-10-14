package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

type fakeDB struct{}

func (fakeDB) Ping(context.Context) error { return nil }

type rejectAll struct{}

func (rejectAll) VerifyRequest(*http.Request) (auth.Claims, error) {
	return auth.Claims{}, auth.ErrUnauthorized
}

func TestRouterAppliesEachContractsSecurity(t *testing.T) {
	s := &Server{db: fakeDB{}, verifier: rejectAll{}}
	h, err := s.RegisterRoutes()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want int
	}{
		{"/healthz", http.StatusOK},      // security: [] in the contract
		{"/me", http.StatusUnauthorized}, // protected by default
		{"/nope", http.StatusNotFound},   // not in any contract
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if w.Code != tt.want {
			t.Errorf("GET %s without a token: status = %d, want %d", tt.path, w.Code, tt.want)
		}
	}
}

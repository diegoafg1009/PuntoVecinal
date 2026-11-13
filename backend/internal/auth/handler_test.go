package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
)

const testUserID = "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f"

// fakeVerifier accepts only the token "good", as an anonymous or a registered
// user depending on anonymous.
type fakeVerifier struct{ anonymous bool }

func (f fakeVerifier) VerifyRequest(r *http.Request) (platformauth.Claims, error) {
	if r.Header.Get("Authorization") != "Bearer good" {
		return platformauth.Claims{}, platformauth.ErrUnauthorized
	}
	return platformauth.Claims{UserID: testUserID, IsAnonymous: f.anonymous}, nil
}

func newRouter(t *testing.T, anonymous bool) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	if err := Register(r, fakeVerifier{anonymous: anonymous}); err != nil {
		t.Fatal(err)
	}
	return r
}

func get(h http.Handler, path, authorization string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestGetMe(t *testing.T) {
	for _, anonymous := range []bool{true, false} {
		h := newRouter(t, anonymous)

		first := get(h, "/me", "Bearer good")
		second := get(h, "/me", "Bearer good")
		if first.Code != http.StatusOK || second.Code != http.StatusOK {
			t.Fatalf("status = %d, %d", first.Code, second.Code)
		}

		var body struct {
			ID          string `json:"id"`
			IsAnonymous bool   `json:"isAnonymous"`
		}
		if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.ID != testUserID || body.IsAnonymous != anonymous {
			t.Fatalf("anonymous=%v: got %+v", anonymous, body)
		}
		if first.Body.String() != second.Body.String() {
			t.Fatalf("same token must yield the same user: %q vs %q", first.Body, second.Body)
		}
	}
}

func TestGetMeRejectsBadTokens(t *testing.T) {
	h := newRouter(t, true)
	for name, header := range map[string]string{
		"missing":      "",
		"bad token":    "Bearer bad",
		"wrong scheme": "Basic good",
	} {
		t.Run(name, func(t *testing.T) {
			if w := get(h, "/me", header); w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
		})
	}
}

package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

const testSpec = `{
  "openapi": "3.0.3",
  "info": {"title": "test", "version": "1"},
  "security": [{"bearerAuth": []}],
  "paths": {
    "/open":   {"get": {"security": [], "responses": {"200": {"description": "ok"}}}},
    "/closed": {"get": {"responses": {"200": {"description": "ok"}}}},
    "/upload": {"post": {
      "requestBody": {"required": true, "content": {"multipart/form-data": {"schema": {"type": "object", "required": ["name"], "properties": {"name": {"type": "string"}}}}}},
      "responses": {"200": {"description": "ok"}}
    }}
  },
  "components": {"securitySchemes": {"bearerAuth": {"type": "http", "scheme": "bearer"}}}
}`

type middlewareSetup struct {
	handler http.Handler
	called  *bool
	claims  *Claims
}

func newMiddlewareSetup(t *testing.T) middlewareSetup {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromData([]byte(testSpec))
	if err != nil {
		t.Fatal(err)
	}
	key := newKey(t)
	v := newTestVerifier(t, key)

	var called bool
	var got Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got, _ = ClaimsFrom(r.Context())
	})
	setup := middlewareSetup{handler: Middleware(spec, v)(next), called: &called, claims: &got}
	setup.handler = withToken(setup.handler, sign(t, key, validClaims(true)))
	return setup
}

// withToken lets a test ask for the valid token with the header value "valid".
func withToken(h http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "valid" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		h.ServeHTTP(w, r)
	})
}

func serve(h http.Handler, path, authorization string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{name: "valid token reaches the handler", path: "/closed", header: "valid", wantStatus: 200, wantCalled: true},
		{name: "missing token", path: "/closed", wantStatus: 401},
		{name: "malformed header", path: "/closed", header: "Basic abc", wantStatus: 401},
		{name: "invalid token", path: "/closed", header: "Bearer not-a-jwt", wantStatus: 401},
		{name: "public operation needs no token", path: "/open", wantStatus: 200, wantCalled: true},
		{name: "unknown path", path: "/missing", wantStatus: 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newMiddlewareSetup(t)
			w := serve(s.handler, tt.path, tt.header)
			if w.Code != tt.wantStatus || *s.called != tt.wantCalled {
				t.Fatalf("status = %d (want %d), handler called = %v (want %v)", w.Code, tt.wantStatus, *s.called, tt.wantCalled)
			}
			if tt.wantStatus == 401 && w.Body.String() != "unauthorized\n" {
				t.Fatalf("body = %q", w.Body.String())
			}
		})
	}
}

func TestMiddlewareSetsClaims(t *testing.T) {
	s := newMiddlewareSetup(t)
	serve(s.handler, "/closed", "valid")
	if s.claims.UserID != "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f" || !s.claims.IsAnonymous {
		t.Fatalf("claims in context = %+v", *s.claims)
	}
}

func TestAuthFuncRejectsOtherSchemes(t *testing.T) {
	fn := NewAuthFunc(newTestVerifier(t, newKey(t)))
	err := fn(t.Context(), &openapi3filter.AuthenticationInput{SecuritySchemeName: "apiKey"})
	if err == nil {
		t.Fatal("expected an error for a scheme other than bearerAuth")
	}
}

// countingBody records how many bytes were read from it.
type countingBody struct {
	r    io.Reader
	read int
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.read += n
	return n, err
}
func (b *countingBody) Close() error { return nil }

func TestMiddlewareLeavesTheBodyToTheHandler(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromData([]byte(testSpec))
	if err != nil {
		t.Fatal(err)
	}
	key := newKey(t)
	v := newTestVerifier(t, key)
	token := sign(t, key, validClaims(true))

	var seen string
	h := Middleware(spec, v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
	}))

	// Not a multipart body that matches the contract: it must not be validated.
	r := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("raw-bytes"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if seen != "raw-bytes" {
		t.Errorf("handler read %q, want the untouched body", seen)
	}
}

func TestMiddlewareDoesNotReadTheBodyOfRejectedRequests(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromData([]byte(testSpec))
	if err != nil {
		t.Fatal(err)
	}
	key := newKey(t)
	h := Middleware(spec, newTestVerifier(t, key))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler reached without a token")
	}))

	body := &countingBody{r: strings.NewReader(strings.Repeat("x", 1<<16))}
	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if body.read != 0 {
		t.Errorf("%d bytes of the body were read before rejecting", body.read)
	}
}

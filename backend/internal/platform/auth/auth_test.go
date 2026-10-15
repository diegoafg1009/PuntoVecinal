package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer = "http://supabase.test/auth/v1"
	testKeyID  = "test-key"
)

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// newTestVerifier serves key's public half as a JWKS and returns a Verifier
// that trusts it.
func newTestVerifier(t *testing.T, key *ecdsa.PrivateKey) *Verifier {
	t.Helper()
	enc := base64.RawURLEncoding.EncodeToString
	// Uncompressed point: 0x04 || X (32 bytes) || Y (32 bytes).
	point, err := key.PublicKey.Bytes()
	if err != nil {
		t.Fatalf("encode public key: %v", err)
	}
	jwks := map[string]any{"keys": []map[string]string{{
		"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": testKeyID,
		"x": enc(point[1:33]),
		"y": enc(point[33:65]),
	}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	v, err := NewVerifier(ctx, "http://supabase.test", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func sign(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = testKeyID
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims(anonymous bool) jwt.MapClaims {
	return jwt.MapClaims{
		"sub":          "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f",
		"iss":          testIssuer,
		"aud":          "authenticated",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"is_anonymous": anonymous,
	}
}

func TestVerify(t *testing.T) {
	key := newKey(t)
	v := newTestVerifier(t, key)

	expired := validClaims(true)
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	otherIssuer := validClaims(true)
	otherIssuer["iss"] = "http://evil.test/auth/v1"
	otherAudience := validClaims(true)
	otherAudience["aud"] = "something-else"
	noSubject := validClaims(true)
	delete(noSubject, "sub")
	noExpiry := validClaims(true)
	delete(noExpiry, "exp")

	tests := []struct {
		name    string
		token   string
		want    Claims
		wantErr bool
	}{
		{name: "anonymous session", token: sign(t, key, validClaims(true)),
			want: Claims{UserID: "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f", IsAnonymous: true}},
		{name: "registered session", token: sign(t, key, validClaims(false)),
			want: Claims{UserID: "7b0d8f5e-1c2a-4d3b-9e8f-0a1b2c3d4e5f"}},
		{name: "expired", token: sign(t, key, expired), wantErr: true},
		{name: "no expiry", token: sign(t, key, noExpiry), wantErr: true},
		{name: "bad signature", token: sign(t, newKey(t), validClaims(true)), wantErr: true},
		{name: "wrong issuer", token: sign(t, key, otherIssuer), wantErr: true},
		{name: "wrong audience", token: sign(t, key, otherAudience), wantErr: true},
		{name: "no subject", token: sign(t, key, noSubject), wantErr: true},
		{name: "garbage", token: "not-a-jwt", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.Verify(tt.token)
			if tt.wantErr {
				if !errors.Is(err, ErrUnauthorized) {
					t.Fatalf("want ErrUnauthorized, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVerifyRequest(t *testing.T) {
	key := newKey(t)
	v := newTestVerifier(t, key)
	token := sign(t, key, validClaims(true))

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{name: "bearer", header: "Bearer " + token},
		{name: "lowercase scheme", header: "bearer " + token},
		{name: "missing header", header: "", wantErr: true},
		{name: "wrong scheme", header: "Basic " + token, wantErr: true},
		{name: "empty token", header: "Bearer ", wantErr: true},
		{name: "token without scheme", header: token, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}
			_, err := v.VerifyRequest(r)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaimsContext(t *testing.T) {
	if _, ok := ClaimsFrom(context.Background()); ok {
		t.Fatal("empty context must have no claims")
	}
	want := Claims{UserID: "u", IsAnonymous: true}
	got, ok := ClaimsFrom(WithClaims(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("got %+v, %v", got, ok)
	}
}

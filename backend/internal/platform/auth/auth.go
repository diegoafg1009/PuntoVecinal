// Package auth verifies Supabase access tokens (anonymous sessions included)
// and carries the verified caller through the request context.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// ErrUnauthorized is returned for any token that must be rejected: missing,
// malformed, expired, badly signed or issued by another project.
var ErrUnauthorized = errors.New("unauthorized")

// Claims is the verified caller behind a request.
type Claims struct {
	UserID      string
	IsAnonymous bool
}

type claims struct {
	jwt.RegisteredClaims
	IsAnonymous bool `json:"is_anonymous"`
}

// Verifier checks Supabase JWTs against the project's public keys.
type Verifier struct {
	keyfunc jwt.Keyfunc
	parser  *jwt.Parser
}

// NewVerifier fetches and caches the keys at jwksURL (refreshing them in the
// background until ctx ends) and accepts tokens issued by the Supabase
// project at supabaseURL.
func NewVerifier(ctx context.Context, supabaseURL, jwksURL string) (*Verifier, error) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("load jwks: %w", err)
	}
	return newVerifier(k.Keyfunc, strings.TrimRight(supabaseURL, "/")+"/auth/v1"), nil
}

func newVerifier(kf jwt.Keyfunc, issuer string) *Verifier {
	return &Verifier{
		keyfunc: kf,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"ES256", "RS256"}),
			jwt.WithIssuer(issuer),
			jwt.WithAudience("authenticated"),
			jwt.WithExpirationRequired(),
		),
	}
}

// Verify validates a raw token and returns the caller it belongs to.
func (v *Verifier) Verify(token string) (Claims, error) {
	var c claims
	if _, err := v.parser.ParseWithClaims(token, &c, v.keyfunc); err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}
	if c.Subject == "" {
		return Claims{}, fmt.Errorf("%w: token has no subject", ErrUnauthorized)
	}
	return Claims{UserID: c.Subject, IsAnonymous: c.IsAnonymous}, nil
}

// VerifyRequest reads the bearer token of r and verifies it.
func (v *Verifier) VerifyRequest(r *http.Request) (Claims, error) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return Claims{}, fmt.Errorf("%w: missing bearer token", ErrUnauthorized)
	}
	return v.Verify(strings.TrimSpace(token))
}

type claimsKey struct{}

// WithClaims stores the verified caller in ctx.
func WithClaims(ctx context.Context, p Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, p)
}

// ClaimsFrom returns the verified caller stored by WithClaims.
func ClaimsFrom(ctx context.Context) (Claims, bool) {
	p, ok := ctx.Value(claimsKey{}).(Claims)
	return p, ok
}

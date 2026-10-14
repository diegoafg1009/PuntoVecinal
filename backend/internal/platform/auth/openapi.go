package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// bearerScheme is the name of the bearer token security scheme in the contract.
const bearerScheme = "bearerAuth"

// RequestVerifier is satisfied by *Verifier.
type RequestVerifier interface {
	VerifyRequest(r *http.Request) (Claims, error)
}

// NewAuthFunc returns the function that openapi3filter calls for operations
// that require a token. It accepts only the bearer scheme, verifies the token
// and stores the Claims in the context of the request that the next handler
// will receive.
func NewAuthFunc(v RequestVerifier) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		if input.SecuritySchemeName != bearerScheme {
			return errors.New("unsupported security scheme")
		}

		r := input.RequestValidationInput.Request
		claims, err := v.VerifyRequest(r)
		if err != nil {
			slog.DebugContext(ctx, "request rejected", "error", err)
			return err
		}

		// The validator does not return a new context, so the request it
		// forwards is updated in place.
		*r = *r.WithContext(WithClaims(r.Context(), claims))
		return nil
	}
}

// Middleware applies the security requirements declared in spec: operations
// with `security: []` pass without a token, every other operation needs a
// valid bearer token, and paths that are not in spec get 404.
func Middleware(spec *openapi3.T, v RequestVerifier) func(http.Handler) http.Handler {
	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options:               openapi3filter.Options{AuthenticationFunc: NewAuthFunc(v)},
		ErrorHandlerWithOpts:  handleValidationError,
		SilenceServersWarning: true,
	})
}

func handleValidationError(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
	switch opts.StatusCode {
	case http.StatusUnauthorized:
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case http.StatusNotFound:
		http.Error(w, "not found", http.StatusNotFound)
	case http.StatusBadRequest, http.StatusMethodNotAllowed:
		http.Error(w, err.Error(), opts.StatusCode)
	default:
		slog.ErrorContext(ctx, "request validation failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

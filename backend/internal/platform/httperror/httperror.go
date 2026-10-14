// Package httperror holds the error responses shared by the API modules.
package httperror

import (
	"log/slog"
	"net/http"
)

// Handle answers 500 for an error returned by a handler, logging the cause
// instead of exposing it to the caller. It is meant to be used as the
// ResponseErrorHandlerFunc of the generated strict handlers.
func Handle(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "handler error", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

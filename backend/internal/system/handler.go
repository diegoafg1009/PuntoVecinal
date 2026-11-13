// Package system is the API module for operational endpoints.
package system

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/httperror"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/system/api"
)

// Pinger is satisfied by the database pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

func NewHandler(db Pinger) *Handler { return &Handler{db: db} }

// GetHealthz implements the readiness check from the OpenAPI contract.
func (h *Handler) GetHealthz(ctx context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		return api.GetHealthz503TextResponse("database unavailable"), nil
	}
	return api.GetHealthz200Response{}, nil
}

// Register mounts the module routes on r, protected as the module's contract
// declares.
func Register(r chi.Router, verifier auth.RequestVerifier, db Pinger) error {
	spec, err := api.GetSpec()
	if err != nil {
		return err
	}
	strict := api.NewStrictHandlerWithOptions(NewHandler(db), nil, api.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: httperror.Handle,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{auth.Middleware(spec, verifier)},
	})
	return nil
}

var _ api.StrictServerInterface = (*Handler)(nil)

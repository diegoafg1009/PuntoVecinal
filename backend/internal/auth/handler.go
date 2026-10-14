package auth

import (
	"context"
	"errors"

	"github.com/go-chi/chi/v5"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/auth/api"
	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/httperror"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// GetMe implements the getMe operation of the contract.
func (h *Handler) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	caller, err := h.service.Me(ctx)
	if errors.Is(err, ErrNoCaller) {
		return api.GetMe401TextResponse("unauthorized"), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetMe200JSONResponse{Id: caller.ID, IsAnonymous: caller.IsAnonymous}, nil
}

// Register mounts the module routes on r, protected as the module's contract
// declares.
func Register(r chi.Router, verifier platformauth.RequestVerifier) error {
	spec, err := api.GetSpec()
	if err != nil {
		return err
	}
	strict := api.NewStrictHandlerWithOptions(NewHandler(NewService()), nil, api.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: httperror.Handle,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{platformauth.Middleware(spec, verifier)},
	})
	return nil
}

var _ api.StrictServerInterface = (*Handler)(nil)

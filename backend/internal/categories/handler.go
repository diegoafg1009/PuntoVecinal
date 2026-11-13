package categories

import (
	"context"

	"github.com/go-chi/chi/v5"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/categories/api"
	platformauth "github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/httperror"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListCategories implements the listCategories operation of the contract.
func (h *Handler) ListCategories(ctx context.Context, _ api.ListCategoriesRequestObject) (api.ListCategoriesResponseObject, error) {
	items, err := h.service.List(ctx)
	if err != nil {
		return nil, err
	}
	body := make(api.ListCategories200JSONResponse, len(items))
	for i, item := range items {
		body[i] = api.Category{Slug: item.Slug, Label: item.Label}
	}
	return body, nil
}

// Register mounts the module routes on r, protected as the module's contract
// declares.
func Register(r chi.Router, verifier platformauth.RequestVerifier, repo Repository) error {
	spec, err := api.GetSpec()
	if err != nil {
		return err
	}
	strict := api.NewStrictHandlerWithOptions(NewHandler(NewService(repo)), nil, api.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: httperror.Handle,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:  r,
		Middlewares: []api.MiddlewareFunc{platformauth.Middleware(spec, verifier)},
	})
	return nil
}

var _ api.StrictServerInterface = (*Handler)(nil)

package httpserver

import (
	"context"
	"time"

	api "github.com/diegoafg1009/PuntoVecinal/backend/internal/server/gen"
)

type pinger interface {
	Ping(ctx context.Context) error
}

// GetHealthz implements the readiness check from the OpenAPI contract.
func (s *Server) GetHealthz(ctx context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return readiness(ctx, s.db), nil
}

func readiness(ctx context.Context, p pinger) api.GetHealthzResponseObject {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		return api.GetHealthz503TextResponse("database unavailable")
	}
	return api.GetHealthz200Response{}
}

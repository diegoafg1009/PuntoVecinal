package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	authmodule "github.com/diegoafg1009/PuntoVecinal/backend/internal/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/system"
)

// RegisterRoutes builds the router: shared middleware plus one mount per API
// module. Which operations need a token is decided by each module's contract.
func (s *Server) RegisterRoutes() (http.Handler, error) {
	r := chi.NewRouter()
	r.Use(requestLogger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	if err := system.Register(r, s.verifier, s.db); err != nil {
		return nil, err
	}
	if err := authmodule.Register(r, s.verifier); err != nil {
		return nil, err
	}

	return r, nil
}

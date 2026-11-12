package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/categories"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/categories/repository"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/config"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/storage"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/reports"
	reportsrepo "github.com/diegoafg1009/PuntoVecinal/backend/internal/reports/repository"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/system"
)

type Server struct {
	port int

	db         system.Pinger
	categories categories.Repository
	reports    reports.Repository
	photos     reports.PhotoStore
	embedder   reports.Embedder
	verifier   auth.RequestVerifier
}

// NewServer builds the HTTP server. The embedder turns report texts into
// vectors for the similar-reports check; it is built by the caller because the
// backfill job needs the same one.
func NewServer(cfg config.Config, db *pgxpool.Pool, verifier auth.RequestVerifier, embedder reports.Embedder) (*http.Server, error) {
	s := &Server{
		port:       cfg.App.Port,
		db:         db,
		categories: repository.NewPgRepository(db),
		reports:    reportsrepo.NewPgRepository(db),
		photos:     storage.NewS3(cfg.S3),
		embedder:   embedder,
		verifier:   verifier,
	}

	handler, err := s.RegisterRoutes()
	if err != nil {
		return nil, fmt.Errorf("register routes: %w", err)
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server, nil
}

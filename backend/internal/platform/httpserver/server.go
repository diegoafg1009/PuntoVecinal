package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/config"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/system"
)

type Server struct {
	port int

	db       system.Pinger
	verifier auth.RequestVerifier
}

func NewServer(cfg config.Config, db *pgxpool.Pool, verifier auth.RequestVerifier) (*http.Server, error) {
	s := &Server{
		port:     cfg.App.Port,
		db:       db,
		verifier: verifier,
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

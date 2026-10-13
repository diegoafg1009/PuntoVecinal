package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/config"
)

type Server struct {
	port int

	db *pgxpool.Pool
}

func NewServer(cfg config.Config, db *pgxpool.Pool) *http.Server {
	NewServer := &Server{
		port: cfg.App.Port,
		db:   db,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

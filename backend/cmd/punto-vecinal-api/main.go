package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/auth"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/config"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/database"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/embedding"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/httpserver"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/logging"
	"github.com/diegoafg1009/PuntoVecinal/backend/internal/reports"
	reportsrepo "github.com/diegoafg1009/PuntoVecinal/backend/internal/reports/repository"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	slog.Info("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	logging.Setup(cfg.App.Env)

	pool, err := database.NewPool(context.Background(), cfg.Database)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	verifier, err := auth.NewVerifier(context.Background(), cfg.Supabase.URL, cfg.Supabase.JWKSURL)
	if err != nil {
		slog.Error("create token verifier", "error", err)
		os.Exit(1)
	}

	// Similar reports need a Voyage key. Without one the check answers 503 and
	// publishing works as before.
	embedder := embedding.NewVoyage(cfg.Voyage.APIKey)
	if !embedder.Configured() {
		slog.Warn("VOYAGE_APIKEY is empty: similar reports are disabled")
	}

	server, err := httpserver.NewServer(cfg, pool, verifier, embedder)
	if err != nil {
		slog.Error("create http server", "error", err)
		os.Exit(1)
	}

	// Embed the reports published while Voyage was unreachable. It stops with
	// the server context.
	backfillCtx, stopBackfill := context.WithCancel(context.Background())
	defer stopBackfill()
	if embedder.Configured() {
		go reports.RunEmbeddingBackfill(backfillCtx, reportsrepo.NewPgRepository(pool), embedder, slog.Default())
	}

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(server, done)

	slog.Info("starting server", "addr", server.Addr, "env", cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server error", "error", err)
		os.Exit(1)
	}

	// Wait for the graceful shutdown to complete
	<-done
	slog.Info("graceful shutdown complete")
}

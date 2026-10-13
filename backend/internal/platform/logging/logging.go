// Package logging configures the process-wide slog logger.
package logging

import (
	"log/slog"
	"os"
)

// Setup installs the default logger: human-readable text in development and
// JSON everywhere else.
func Setup(env string) {
	var handler slog.Handler
	if env == "development" {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(handler))
}

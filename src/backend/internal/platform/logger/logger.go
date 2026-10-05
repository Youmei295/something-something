// Package logger provides a small structured-logging helper built on log/slog.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON logger in production and a human-readable logger elsewhere.
func New(env string) *slog.Logger {
	level := slog.LevelInfo
	if strings.EqualFold(env, "development") {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if env == "production" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	// The service name is attached by each entrypoint (see internal/services),
	// so the base logger stays context-free.
	return slog.New(h)
}

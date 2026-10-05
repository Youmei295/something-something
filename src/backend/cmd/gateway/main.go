package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/platform/httpserver"
	"github.com/youmei295/something-something/src/backend/internal/platform/logger"
	"github.com/youmei295/something-something/src/backend/internal/transport/gateway"
)

// cmd/gateway is the single entry point that routes to the standalone services.
func main() {
	if err := run(); err != nil {
		slog.Error("gateway exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Env).With(slog.String("service", "gateway"))

	gw, err := gateway.New(cfg.Gateway, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return httpserver.Run(ctx, "gateway", cfg.Gateway.Addr, gw.Handler(), log, cfg.ShutdownTimeout)
}

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/youmei295/something-something/src/backend/internal/bootstrap"
	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/platform/logger"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Env).With(slog.String("service", "worker"))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	core, err := bootstrap.NewCore(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer core.Close()

	worker, err := core.Worker()
	if err != nil {
		return err
	}
	return worker.Run(ctx)
}

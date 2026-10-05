// Package services provides the shared entrypoint used by every standalone
// service binary. A service is defined purely by what it enables, which keeps
// the individual main functions to a single line and guarantees consistent
// startup, seeding, and shutdown behavior.
package services

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/youmei295/something-something/src/backend/internal/bootstrap"
	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/platform/httpserver"
	"github.com/youmei295/something-something/src/backend/internal/platform/logger"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// Run starts an HTTP service with the given contexts enabled and blocks until
// the process receives SIGINT/SIGTERM.
func Run(name string, enable transporthttp.Enable) {
	if err := run(name, enable); err != nil {
		slog.Error("service exited", slog.String("service", name), slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(name string, enable transporthttp.Enable) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Env).With(slog.String("service", name))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	core, err := bootstrap.NewCore(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer core.Close()

	// The identity service owns the host account; seeding here makes a fresh
	// microservice deployment usable even if the migrate job did not run.
	if enable.Identity {
		if created, err := core.AuthService().SeedHost(ctx, cfg.Host.Email, cfg.Host.Name, cfg.Host.Password); err != nil {
			log.Warn("host seed skipped", slog.String("error", err.Error()))
		} else if created {
			log.Info("host account created", slog.String("email", cfg.Host.Email))
		}
	}

	api, err := core.API(ctx, enable)
	if err != nil {
		return err
	}
	return httpserver.Run(ctx, name, cfg.HTTP.Addr, api.Router(), log, cfg.ShutdownTimeout)
}

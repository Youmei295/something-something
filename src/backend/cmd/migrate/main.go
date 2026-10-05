package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/youmei295/something-something/src/backend/internal/adapters/auth"
	"github.com/youmei295/something-something/src/backend/internal/adapters/postgres"
	"github.com/youmei295/something-something/src/backend/internal/app"
	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/platform/clock"
	"github.com/youmei295/something-something/src/backend/internal/platform/id"
	"github.com/youmei295/something-something/src/backend/internal/platform/logger"
	"github.com/youmei295/something-something/src/backend/migrations"
)

func main() {
	seed := flag.Bool("seed", false, "seed the host account after migrating")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log := logger.New(cfg.Env).With(slog.String("service", "migrate"))
	ctx := context.Background()

	if err := migrations.Up(ctx, cfg.DatabaseURL); err != nil {
		log.Error("migrate failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("migrations applied")

	if !*seed {
		return
	}

	// Seeding only needs the database, not storage or mail, so build a minimal
	// dependency graph and avoid coupling migration to those services.
	store, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("seed: database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer store.Close()

	ids := id.NewGenerator()
	authSvc := app.NewAuthService(store, auth.NewArgon2Hasher(), clock.New(), ids, cfg.Auth.SessionTTL)

	created, err := authSvc.SeedHost(ctx, cfg.Host.Email, cfg.Host.Name, cfg.Host.Password)
	if err != nil {
		log.Error("seed failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if created {
		log.Info("host account created", slog.String("email", cfg.Host.Email))
	} else {
		log.Info("host account already exists; nothing to do")
	}
}

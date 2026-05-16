package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	"sso/internal/logger"
	"sso/internal/storage/postgres"
	"sync"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	cfg := config.Load()

	log := setupLogger(cfg.Env)

	log.Info("starting application",
		slog.String("env", cfg.Env),
	)

	pool, err := postgres.NewPool(context.Background(), cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		log.Error("failed to connect postgres", "error", err)
	}
	defer pool.Close()
	repo := postgres.NewRepository(pool)
	application := app.New(log, cfg.GRPC.Port, repo, cfg.TokenTTL, cfg.RefreshTokenTTL, cfg.TimeoutDuration)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("starting gRPC server", slog.Int("port", cfg.GRPC.Port))
		application.GRPCServer.MustRun()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	log.Info("shutting down application...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.GRPCServer.Stop(shutdownCtx); err != nil {
		log.Error("grpc shutdown failed", "error", err)
	}

	wg.Wait()

	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = logger.New(slog.LevelDebug)
	case envProd:
		log = logger.New(slog.LevelInfo)
	}
	return log
}

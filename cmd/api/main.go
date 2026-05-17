package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	"sso/internal/logger"
	"sso/internal/storage/postgres"
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
		os.Exit(1)
	}

	repo := postgres.NewRepository(pool)
	application := app.New(log, cfg.GRPC.Port, repo, cfg.TokenTTL, cfg.RefreshTokenTTL, cfg.TimeoutDuration)

	go func() {
		log.Info("starting gRPC server", slog.Int("port", cfg.GRPC.Port))
		application.GRPCServer.MustRun()
	}()

	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		http.ListenAndServe(":8080", nil)
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

	pool.Close()
	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	switch env {
	case envLocal:
		return logger.New(slog.LevelDebug)
	case envProd:
		return logger.New(slog.LevelInfo)
	default:
		return logger.New(slog.LevelInfo)
	}
}

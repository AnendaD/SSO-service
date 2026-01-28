package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	"sso/internal/lib/logger/handlers/slogpretty"
	"sso/internal/proxy"
	"sync"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting application",
		slog.String("env", cfg.Env),
	)

	application := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL, cfg.RefreshTokenTTL)

	var wg sync.WaitGroup

	// Запускаем gRPC сервер в горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("starting gRPC server", slog.Int("port", cfg.GRPC.Port))
		application.GRPCServer.MustRun()
	}()

	// Запускаем HTTP прокси сервер в горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		server := proxy.NewProxyServer("localhost:44044")
		if err := server.Start("8080", log); err != nil {
			log.Error("failed to start proxy server", slog.String("error", err.Error()))
			panic("failed to start proxy server")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	log.Info("shutting down application...")

	// Останавливаем gRPC сервер
	application.GRPCServer.Stop()

	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}

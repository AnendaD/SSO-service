package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/config"
	"sso/internal/logger"
	"sso/internal/proxy"
	"syscall"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	cfg := config.Load()
	log := setupLogger(cfg.Env)

	proxyServer, err := proxy.NewProxyServer(fmt.Sprintf("localhost:%d", cfg.GRPC.Port), log)

	if err != nil {
		log.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := proxyServer.Start(fmt.Sprintf("%d", cfg.HTTP.Port), log); err != nil {
			log.Error("proxy failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	log.Info("shutting down proxy...")
	proxyServer.Close()

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

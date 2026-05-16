package app

import (
	"log/slog"
	"time"

	grpcapp "sso/internal/app/grpc"
	"sso/internal/services/auth"
	"sso/internal/storage/postgres"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	repo *postgres.Repository,
	tokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	timeoutDuration time.Duration,
) *App {
	authService := auth.New(log, repo, repo, repo, repo, repo, tokenTTL, refreshTokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort, timeoutDuration)

	return &App{
		GRPCServer: grpcApp,
	}
}

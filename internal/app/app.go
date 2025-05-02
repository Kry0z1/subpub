package app

import (
	"log/slog"
	"time"

	grpcsubpub "github.com/Kry0z1/subpub/internal/app/grpc"
	"github.com/Kry0z1/subpub/internal/service"
)

type App struct {
	GRPCServer *grpcsubpub.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	timeout time.Duration,
) *App {
	srvc := service.New(log)

	grpcApp := grpcsubpub.New(&srvc, log, grpcPort, timeout)

	return &App{
		GRPCServer: grpcApp,
	}
}

package suite

import (
	"log/slog"

	"github.com/Kry0z1/subpub/internal/app"
	"github.com/Kry0z1/subpub/internal/config"
	"github.com/Kry0z1/subpub/internal/logger/slogdiscard"
)

func StartServer(cfg *config.Config) func() {
	logger := slog.New(slogdiscard.NewDiscardHandler())

	application := app.New(logger, cfg.GRPC.Port, cfg.GRPC.Timeout)

	go func() {
		application.GRPCServer.MustRun()
	}()

	return func() {
		application.GRPCServer.Stop(cfg.StopTimeout)
	}
}

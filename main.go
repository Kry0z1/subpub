package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kry0z1/subpub/internal/app"
	"github.com/Kry0z1/subpub/internal/config"
	"github.com/Kry0z1/subpub/internal/logger/slogpretty"
)

var (
	localStr = "local"
	prodStr  = "prod"
)

func main() {
	cfg := config.MustLoad()
	fmt.Println(cfg)

	logger := setupLogger(cfg.Env)

	application := app.New(logger, cfg.GRPC.Port, cfg.GRPC.Timeout)

	go func() {
		application.GRPCServer.MustRun()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	application.GRPCServer.Stop(cfg.StopTimeout)
}

func setupLogger(level string) *slog.Logger {
	switch level {
	case localStr:
		return slog.New(slogpretty.NewPrettyHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case prodStr:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	default:
		return slog.Default()
	}
}

package grpcsubpub

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	subpubServer "github.com/Kry0z1/subpub/internal/grpc"
	"github.com/Kry0z1/subpub/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int

	// stored to kill gracefully
	subpub *service.SubPubService

	// end all streams
	cancel func()
}

func New(
	subpubService *service.SubPubService,
	log *slog.Logger,
	port int,
	timeout time.Duration,
) *App {
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(
			logging.PayloadReceived, logging.PayloadSent,
		),
	}

	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			log.Error("Recovered from panic", slog.Any("panic", p))
			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(recoveryOpts...),
		logging.UnaryServerInterceptor(InterceptorLogger(log), loggingOpts...),
		InterceptorTimeout(timeout),
	))

	cctx, cancel := context.WithCancel(context.Background())
	subpubServer.Register(gRPCServer, subpubService, cctx)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
		subpub:     subpubService,
		cancel:     cancel,
	}
}

func (a *App) Run() error {
	const op = "app.grpc.Run"

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("grpc server started", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Stop(timeout time.Duration) {
	const op = "app.grpc.Stop"

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log := a.log.With(slog.String("op", op))

	log.Info("stopping gRPC server", slog.Int("port", a.port))
	stopped := make(chan struct{})
	go func() {
		a.cancel()
		a.gRPCServer.GracefulStop()
		stopped <- struct{}{}
	}()

	select {
	case <-stopped:
		log.Info("server stopped")
	case <-time.After(timeout):
		log.Info("couldn't gracefully stop server")
		log.Info("killing server forcibly")
		a.gRPCServer.Stop()
		log.Info("server stopped")
	}

	log.Info("stopping subpub system")
	err := a.subpub.Stop(ctx)

	if err != nil {
		log.Info("couldn't gracefully stop subpub system")
		log.Info("killing subpub system forcibly")
	}

	log.Info("subpub system stopped")
}

func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func InterceptorTimeout(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return handler(tctx, req)
	}
}

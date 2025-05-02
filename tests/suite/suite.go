package suite

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/Kry0z1/subpub/internal/config"
	pubsubv1 "github.com/Kry0z1/subpub/protos/gen/go/pubsub"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Suite struct {
	*testing.T
	PubSub     pubsubv1.PubSubClient
	Cfg        *config.Config
	serverStop func()
}

func (s *Suite) Subscribe(ctx context.Context, key string) (grpc.ServerStreamingClient[pubsubv1.Event], error) {
	return s.PubSub.Subscribe(ctx, &pubsubv1.SubscribeRequest{Key: key})
}

func (s *Suite) Publish(ctx context.Context, key string, data string) error {
	_, err := s.PubSub.Publish(ctx, &pubsubv1.PublishRequest{
		Key:  key,
		Data: data,
	})

	return err
}

func (s *Suite) Close() {
	s.serverStop()
}

func New(t *testing.T) (context.Context, Suite) {
	t.Helper()

	cfg := config.MustLoadPath(configPath())

	serverStop := StartServer(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.Timeout)
	cc, err := grpc.DialContext(ctx, grpcAddress(cfg), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to grpc server: %v", err)
	}

	t.Cleanup(func() {
		t.Helper()
		cancel()
	})

	return ctx, Suite{
		T:          t,
		PubSub:     pubsubv1.NewPubSubClient(cc),
		Cfg:        cfg,
		serverStop: serverStop,
	}
}

func configPath() string {
	var res string
	if res = os.Getenv("CONFIG_PATH"); res == "" {
		res = "../config/test.yaml"
	}

	return res
}

func grpcAddress(cfg *config.Config) string {
	return net.JoinHostPort("localhost", strconv.Itoa(cfg.GRPC.Port))
}

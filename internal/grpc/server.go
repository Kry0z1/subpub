package grpc

import (
	"context"
	pubsubv1 "github.com/Kry0z1/subpub/protos/gen/go/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SubPub interface {
	Subscribe(key string) (chan string, error)
	Publish(ctx context.Context, key string, data string) error
}

type SubPubServer struct {
	pubsubv1.UnimplementedPubSubServer
	subpub    SubPub
	cancelCtx context.Context
}

func (s SubPubServer) Subscribe(request *pubsubv1.SubscribeRequest, g grpc.ServerStreamingServer[pubsubv1.Event]) error {
	pipe, err := s.subpub.Subscribe(request.GetKey())
	if err != nil {
		return status.Error(codes.Internal, "couldn't subscribe")
	}

	for {
		select {
		case msg := <-pipe:
			if pipe == nil {
				return nil
			}
			if err := g.Send(&pubsubv1.Event{Data: msg}); err != nil {
				return status.Error(codes.Aborted, "stream has broken")
			}
		case <-s.cancelCtx.Done():
			return status.Error(codes.Aborted, "server died")
		}
	}
}

func (s SubPubServer) Publish(ctx context.Context, request *pubsubv1.PublishRequest) (*emptypb.Empty, error) {
	err := s.subpub.Publish(ctx, request.Key, request.Data)
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.Internal, "publish failed")
	}

	return &emptypb.Empty{}, nil
}

func New(subpub SubPub, ctx context.Context) pubsubv1.PubSubServer {
	return &SubPubServer{subpub: subpub, cancelCtx: ctx}
}

func Register(server *grpc.Server, subpub SubPub, ctx context.Context) {
	pubsubv1.RegisterPubSubServer(server, New(subpub, ctx))
}

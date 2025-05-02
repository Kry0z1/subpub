package service

import (
	"context"
	"fmt"
	"github.com/Kry0z1/subpub/pkg/subpub"
	"log/slog"
)

type SubPub interface {
	Subscribe(key string) (chan string, error)
	Publish(ctx context.Context, key string, data string) error
}

type SubPubService struct {
	subpubSystem subpub.SubPub
	log          *slog.Logger
}

func New(log *slog.Logger) SubPubService {
	return SubPubService{
		log:          log,
		subpubSystem: subpub.NewSubPub(),
	}
}

func (s *SubPubService) Subscribe(key string) (chan string, error) {
	const op = "service.Subscribe"

	log := s.log.With(
		slog.String("op", op),
		slog.String("key", key),
	)

	returnChan := make(chan string)
	cb := func(msg interface{}) {
		returnChan <- msg.(string)
	}

	log.Info("started subscription")
	_, err := s.subpubSystem.Subscribe(key, cb)
	if err != nil {
		log.Error("subscription failed: ", err.Error())
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("subscription successful")
	return returnChan, nil
}

// on ctx cancel returns but eventually message will be sent
// if context is canceled error from subpub will be omitted
func (s *SubPubService) Publish(ctx context.Context, key string, data string) error {
	const op = "service.Publish"

	log := s.log.With(
		slog.String("op", op),
		slog.String("key", key),
		slog.String("data", data),
	)

	published := make(chan error)

	log.Info("started publish", key, data)

	go func() {
		published <- s.subpubSystem.Publish(key, data)
	}()

	select {
	case err := <-published:
		log.Info("successfully published")
		if err == nil {
			return nil
		}
		log.Error("publish failed: ", err.Error())
		return fmt.Errorf("%s: %w", op, err)
	case <-ctx.Done():
		log.Info("timed out")
		return fmt.Errorf("%s: %w", op, ctx.Err())
	}
}

func (s *SubPubService) Stop(ctx context.Context) error {
	return s.subpubSystem.Close(ctx)
}

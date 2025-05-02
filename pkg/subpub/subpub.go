package subpub

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrClosed      = errors.New("subpub system is closed")
	ErrTopicClosed = errors.New("this topic is closed")
)

type subpub struct {
	// used sync.Map because I expect not a lot of topics
	// but lots of publishing
	broadcasters sync.Map // map[string]*broadcaster
	closed       bool
}

// Subscribe searches for broadcaster on given subject,
// creates it if there is none and initializes new subscriber.
func (s *subpub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if s.closed {
		return nil, ErrClosed
	}

	newBroadcast := newBroadcaster()
	bAny, _ := s.broadcasters.LoadOrStore(subject, &newBroadcast)
	b := bAny.(*broadcaster)

	id := b.GetNextId()

	sub := newSubscription(id, cb, b)

	b.RegisterSub(sub, id)

	sub.Start()

	return sub, nil
}

// Publish passes subject to broadcaster on subject if there is any.
//
// On empty subject returns nil.
//
// If broadcaster was closed during Close call, but it wasn't successful
// (context got canceled) returns ErrTopicClosed.
func (s *subpub) Publish(subject string, msg interface{}) error {
	if s.closed {
		return ErrClosed
	}

	bAny, ok := s.broadcasters.Load(subject)
	if !ok {
		return nil
	}
	if err := bAny.(*broadcaster).Publish(msg); err != nil {
		return ErrTopicClosed
	}
	return nil
}

// Close initiates closing process on all the broadcasters.
//
// If context is done before Close call returns ctx.Err().
//
// If context is done during closing of one of broadcasters,
// no more broadcasters will be closed.
// Moreover, broadcaster will try to close after Close returns
// in its own goroutine.
//
// For guarantees on broadcaster closing refer to
// the appropriate broadcaster.
func (s *subpub) Close(ctx context.Context) error {
	closed := make(chan struct{})

	if ctx.Err() != nil {
		return ctx.Err()
	}

	go func() {
		s.broadcasters.Range(func(key, value any) bool {
			err := value.(*broadcaster).Close(ctx)
			return err == nil
		})
		if ctx.Err() == nil {
			s.closed = true
			closed <- struct{}{}
		}
	}()

	select {
	case <-closed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newSubPub() *subpub {
	return &subpub{broadcasters: sync.Map{}}
}

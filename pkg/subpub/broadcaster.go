package subpub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var ErrBroadcasterClosed = errors.New("broadcaster is closed")

type broadcaster struct {
	// May be sync.Map is better here, depends on use cases
	mut           *sync.RWMutex
	subscriptions map[int64]*subscription

	maxSubscriptionID *atomic.Int64
	closed            bool
}

// Closes broadcaster and  unsubscribes all of its subscribers.
//
// On success returns nil.
//
// If ctx is closed before close, broadcaster will not be marked as closed.
// Otherwise, broadcaster will be marked close and no subsequent messages
// on the topic will be delivered.
//
// After context closing at most 1 subscriber will be stopped.
//
// On closed context returns ctx.Err().
func (b *broadcaster) close(ctx context.Context) error {
	b.mut.Lock()
	defer b.mut.Unlock()

	// if context is done already don't close broadcaster
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if b.closed {
		return nil
	}
	b.closed = true
	for _, sub := range b.subscriptions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sub.unsubscribe()
	}

	return nil
}

func (b *broadcaster) publish(message interface{}) error {
	if b.closed {
		return ErrBroadcasterClosed
	}

	b.mut.RLock()
	defer b.mut.RUnlock()
	for _, sub := range b.subscriptions {
		sub.receiver <- message
	}

	return nil
}

func newBroadcaster() broadcaster {
	return broadcaster{
		mut:               &sync.RWMutex{},
		subscriptions:     make(map[int64]*subscription),
		maxSubscriptionID: &atomic.Int64{},
	}
}

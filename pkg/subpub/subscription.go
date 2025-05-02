package subpub

import (
	"sync"
	"sync/atomic"
)

type subscription struct {
	id       int64
	receiver chan interface{}
	cb       MessageHandler
	b        *broadcaster

	active *atomic.Bool

	mut          *sync.Mutex
	cond         *sync.Cond
	queueLen     *atomic.Int64
	messageQueue []interface{}

	receiverClosed  chan struct{}
	processorClosed chan struct{}
}

// messageReceiver receives messages and
// moves them to the internal queue.
//
// Stops on closing of s.receiver.
//
// Blocking call, should be used in goroutine.
func (s *subscription) messageReceiver() {
	for message := range s.receiver {
		s.mut.Lock()
		s.messageQueue = append(s.messageQueue, message)
		s.queueLen.Add(1)
		s.mut.Unlock()
		s.cond.Signal()
	}
	s.receiverClosed <- struct{}{}
}

// queueProcessor processes queue.
//
// Stops on two conditions met at the same time:
//  1. messageQueue must be empty;
//  2. s.active == false.
//
// After those criteria met call to s.cond.L.Signal() will stop processor.
//
// Blocking call, should be used in goroutine.
func (s *subscription) queueProcessor() {
	for {
		if !s.active.Load() {
			if s.queueLen.Load() == 0 {
				break
			}
		} else {
			s.cond.L.Lock()
			for s.queueLen.Load() == 0 && s.active.Load() {
				s.cond.Wait()
			}
			s.cond.L.Unlock()
		}

		s.mut.Lock()
		copiedQueue := s.messageQueue
		s.messageQueue = make([]interface{}, 0)
		s.queueLen.Store(0)
		s.mut.Unlock()

		for _, message := range copiedQueue {
			s.cb(message)
		}
	}
	s.processorClosed <- struct{}{}
}

func (s *subscription) Start() {
	go s.messageReceiver()
	go s.queueProcessor()
}

// UnsubscribeNoLock does the same as Unsubscribe, but
// doesn't take lock for broadcaster.subscriptions.
//
// Used primarily on broadcaster closing.
func (s *subscription) UnsubscribeNoLock() {
	s.b.UnregisterSubNoLock(s.id)

	// stop receiver
	close(s.receiver)

	// wait
	<-s.receiverClosed

	// stop handler
	s.active.Store(false)
	s.cond.Signal()

	// wait
	<-s.processorClosed
}

// Unsubscribe deletes subscription from broadcaster map
// and stops receiver and processor.
//
// Waits for receiver and processor to be stopped.
func (s *subscription) Unsubscribe() {
	s.b.UnregisterSub(s.id)

	// stop receiver
	close(s.receiver)

	// wait
	<-s.receiverClosed

	// stop handler
	s.active.Store(false)
	s.cond.Signal()

	// wait
	<-s.processorClosed
}

func newSubscription(id int64, cb MessageHandler, b *broadcaster) *subscription {
	sub := &subscription{
		id:       id,
		receiver: make(chan interface{}, 1),
		cb:       cb,
		b:        b,

		active: &atomic.Bool{},

		mut:          &sync.Mutex{},
		cond:         sync.NewCond(&sync.Mutex{}),
		queueLen:     &atomic.Int64{},
		messageQueue: make([]interface{}, 0),

		receiverClosed:  make(chan struct{}),
		processorClosed: make(chan struct{}),
	}
	sub.active.Store(true)
	return sub
}

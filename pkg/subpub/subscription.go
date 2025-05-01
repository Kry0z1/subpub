package subpub

import "sync"

type subscription struct {
	id       int64
	receiver chan interface{}
	cb       MessageHandler
	b        *broadcaster

	active bool

	mut          *sync.Mutex
	cond         *sync.Cond
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
		if !s.active {
			if len(s.messageQueue) == 0 {
				break
			}
		} else {
			s.cond.L.Lock()
			for len(s.messageQueue) == 0 && s.active {
				s.cond.Wait()
			}
			s.cond.L.Unlock()
		}

		s.mut.Lock()
		copiedQueue := s.messageQueue
		s.messageQueue = make([]interface{}, 0)
		s.mut.Unlock()

		for _, message := range copiedQueue {
			s.cb(message)
		}
	}
	s.processorClosed <- struct{}{}
}

func (s *subscription) start() {
	go s.messageReceiver()
	go s.queueProcessor()
}

// unsubscribe does the same as Unsubscribe, but
// doesn't take lock for broadcaster.subscriptions.
//
// Used primarily on broadcaster closing.
func (s *subscription) unsubscribe() {
	delete(s.b.subscriptions, s.id)

	// stop receiver
	close(s.receiver)

	// wait
	<-s.receiverClosed

	// stop handler
	s.active = false
	s.cond.Signal()

	// wait
	<-s.processorClosed
}

// Unsubscribe deletes subscription from broadcaster map
// and stops receiver and processor.
//
// Waits for receiver and processor to be stopped.
func (s *subscription) Unsubscribe() {
	s.b.mut.Lock()
	delete(s.b.subscriptions, s.id)
	s.b.mut.Unlock()

	// stop receiver
	close(s.receiver)

	// wait
	<-s.receiverClosed

	// stop handler
	s.active = false
	s.cond.Signal()

	// wait
	<-s.processorClosed
}

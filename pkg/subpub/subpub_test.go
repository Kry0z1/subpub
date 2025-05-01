package subpub_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Kry0z1/subpub/pkg/subpub"
	"github.com/stretchr/testify/assert"
)

// Базовый тест подписки и публикации
func TestBasicOneSubscriber(t *testing.T) {
	sp := subpub.NewSubPub()
	var received bool
	sub, err := sp.Subscribe("test", func(msg interface{}) {
		received = msg == "hello"
	})
	assert.NoError(t, err)

	defer sub.Unsubscribe()

	err = sp.Publish("test", "hello")
	assert.NoError(t, err)

	assert.Eventually(t, func() bool { return received }, 500*time.Millisecond, 10*time.Millisecond, "Message not received")
}

// Тест FIFO порядка сообщений
func TestMessageOrder(t *testing.T) {
	sp := subpub.NewSubPub()
	var messages []int
	var mu sync.Mutex

	sub, _ := sp.Subscribe("order", func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		messages = append(messages, msg.(int))
	})
	defer sub.Unsubscribe()

	for i := 0; i < 100; i++ {
		err := sp.Publish("order", i)
		assert.NoError(t, err)
	}

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(messages) == 100
	}, time.Second, 50*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < 100; i++ {
		assert.Equal(t, i, messages[i], "Wrong message order")
	}
}

// Тест множественных подписчиков
func TestBasicMultipleSubscribers(t *testing.T) {
	sp := subpub.NewSubPub()
	var count int
	var mu sync.Mutex

	const subsCount = 10

	for i := 0; i < subsCount; i++ {
		sub, _ := sp.Subscribe("multi", func(msg interface{}) {
			mu.Lock()
			count++
			mu.Unlock()
		})
		defer sub.Unsubscribe()
	}

	err := sp.Publish("multi", "event")
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return count == subsCount
	}, 500*time.Millisecond, 10*time.Millisecond)
}

// Тест отписки
func TestUnsubscribe(t *testing.T) {
	sp := subpub.NewSubPub()
	var calls int

	sub, _ := sp.Subscribe("unsub", func(msg interface{}) {
		calls++
	})
	sub.Unsubscribe()

	err := sp.Publish("unsub", "data")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Zero(t, calls, "Handler should not be called after unsubscribe")
}

// Тест медленного подписчика
func TestSlowSubscriber(t *testing.T) {
	sp := subpub.NewSubPub()
	var fastDone, slowDone atomic.Int64

	subSlow, _ := sp.Subscribe("slow", func(msg interface{}) {
		time.Sleep(200 * time.Millisecond)
		slowDone.Add(1)
	})
	defer subSlow.Unsubscribe()

	subFast, _ := sp.Subscribe("slow", func(msg interface{}) {
		fastDone.Add(1)
	})
	defer subFast.Unsubscribe()

	for range 3 {
		err := sp.Publish("slow", nil)
		assert.NoError(t, err)
	}

	assert.Eventually(t, func() bool { return fastDone.Load() == 3 }, 100*time.Millisecond, 10*time.Millisecond, "Fast subscriber blocked")
	assert.Eventually(t, func() bool { return slowDone.Load() == 3 }, 1000*time.Millisecond, 10*time.Millisecond, "Slow subscriber didn't finish")
}

// Тест корректного закрытия
func TestGracefulClose(t *testing.T) {
	sp := subpub.NewSubPub()
	var wg sync.WaitGroup

	wg.Add(1)
	_, err := sp.Subscribe("close", func(msg interface{}) {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
	})
	assert.NoError(t, err)

	err = sp.Publish("close", nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = sp.Close(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "Should respect context timeout")

	wg.Wait()
}

// Тест утечек горутин
func TestGoroutineLeak(t *testing.T) {
	initial := runtime.NumGoroutine()

	sp := subpub.NewSubPub()
	sub, _ := sp.Subscribe("leak", func(msg interface{}) {})

	_ = sp.Publish("leak", nil)

	sub.Unsubscribe()

	err := sp.Close(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, runtime.NumGoroutine(), initial, "Goroutine leaked after successful close")
}

// Тест ошибок при закрытой системе
func TestClosedSystemErrors(t *testing.T) {
	sp := subpub.NewSubPub()
	err := sp.Close(context.Background())

	assert.NoError(t, err)

	_, err = sp.Subscribe("closed", func(msg interface{}) {})
	assert.Error(t, err, "Should reject subscribe after successful close")

	err = sp.Publish("closed", nil)
	assert.Error(t, err, "Should reject publish after successful close")
}

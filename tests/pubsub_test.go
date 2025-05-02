package tests

import (
	"context"
	"fmt"
	pubsubv1 "github.com/Kry0z1/subpub/protos/gen/go/pubsub"
	"github.com/Kry0z1/subpub/tests/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"strconv"
	"sync"
	"testing"
	"time"
)

const receiveTimeout = 2 * time.Second

type receiveData struct {
	data string
	err  error
}

func msgReceive(stream grpc.ServerStreamingClient[pubsubv1.Event]) chan receiveData {
	ch := make(chan receiveData)
	go func() {
		data, err := stream.Recv()
		ch <- receiveData{
			data: data.GetData(),
			err:  err,
		}
	}()
	return ch
}

func TestBasic(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	stream, err := st.Subscribe(ctx, "basic")
	assert.NoError(t, err)

	err = st.Publish(ctx, "basic", "test")
	require.NoError(t, err)

	select {
	case msg := <-msgReceive(stream):
		assert.Equal(t, "test", msg.data, "Received wrong message")
	case <-time.After(receiveTimeout):
		t.Error("Message has not been received")
	}
}

func TestDifferentKeys(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	stream1, err := st.Subscribe(ctx, "key1")
	assert.NoError(t, err)

	stream2, err := st.Subscribe(ctx, "key2")
	assert.NoError(t, err)

	err = st.Publish(ctx, "key1", "test")
	require.NoError(t, err)

	select {
	case msg := <-msgReceive(stream1):
		assert.Equal(t, "test", msg.data, "Received wrong message")
	case <-time.After(receiveTimeout):
		t.Error("Message has not been received")
	}

	select {
	case <-msgReceive(stream2):
		assert.Fail(t, "Received but shouldn't")
	case <-time.After(receiveTimeout):
	}
}

func TestMultipleSubscribers(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	stream1, err := st.Subscribe(ctx, "multiple")
	assert.NoError(t, err)

	stream2, err := st.Subscribe(ctx, "multiple")
	assert.NoError(t, err)

	err = st.Publish(ctx, "multiple", "test")
	require.NoError(t, err)

	select {
	case msg := <-msgReceive(stream1):
		assert.Equal(t, "test", msg.data, "Received wrong message")
	case <-time.After(receiveTimeout):
		t.Error("Message has not been received")
	}

	select {
	case msg := <-msgReceive(stream2):
		assert.Equal(t, "test", msg.data, "Received wrong message")
	case <-time.After(receiveTimeout):
		t.Error("Message has not been received")
	}
}

func TestGracefulStop(t *testing.T) {
	ctx, st := suite.New(t)

	stream, err := st.Subscribe(ctx, "shutdown")
	require.NoError(t, err)

	st.Close()

	_, err = stream.Recv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server died")
}

func TestClientDisconnect(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	cctx, cancel := context.WithCancel(ctx)

	stream, err := st.Subscribe(cctx, "disconnect")
	assert.NoError(t, err)

	cancel()

	_, err = stream.Recv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestHighLoad(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	const (
		messagesCount    = 10
		topicsCount      = 10
		subscribersCount = 50
	)

	var wgRun, wgSubscribed sync.WaitGroup
	wgRun.Add(subscribersCount * topicsCount)
	wgSubscribed.Add(subscribersCount * topicsCount)

	for i := range topicsCount {
		for range subscribersCount {
			go func() {
				defer wgRun.Done()
				stream, err := st.Subscribe(ctx, strconv.Itoa(i))
				require.NoError(t, err)
				wgSubscribed.Done()

				for j := range messagesCount {
					data := <-msgReceive(stream)
					require.NoError(t, data.err)
					require.Equal(t, fmt.Sprintf("message-%d", j), data.data)
				}
			}()
		}
	}

	wgSubscribed.Wait()

	for i := range topicsCount {
		go func() {
			for j := range messagesCount {
				err := st.Publish(ctx, strconv.Itoa(i), fmt.Sprintf("message-%d", j))
				require.NoError(t, err)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wgRun.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * receiveTimeout):
		t.Fatal("High load test timeout")
	}
}

func TestEmptyData(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	stream, err := st.Subscribe(ctx, "empty")
	require.NoError(t, err)

	err = st.Publish(ctx, "empty", "")
	require.NoError(t, err)

	select {
	case data := <-msgReceive(stream):
		assert.NoError(t, data.err)
		assert.Empty(t, data.data)
	case <-time.After(receiveTimeout):
		t.Error("Message has not been received")
	}
}

func TestConcurrentPublish(t *testing.T) {
	ctx, st := suite.New(t)
	defer st.Close()

	stream, err := st.Subscribe(ctx, "concurrent")
	require.NoError(t, err)

	const workers = 1000
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func() {
			defer wg.Done()
			err := st.Publish(ctx, "concurrent", fmt.Sprintf("msg-%d", i))
			require.NoError(t, err)
		}()
	}

	received := make(map[string]bool)
	var mu sync.Mutex

	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				return
			}
			mu.Lock()
			received[event.Data] = true
			mu.Unlock()
		}
	}()

	wg.Wait()
	time.Sleep(receiveTimeout)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, workers, len(received))
}

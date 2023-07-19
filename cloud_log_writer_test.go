package cloudlog

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testPublisher struct {
	Events []LogEvent

	mu sync.Mutex
}

func (t *testPublisher) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Events = nil
}

func (t *testPublisher) PublishLogEvents(_ context.Context, e []LogEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Events = append(t.Events, e...)

	return nil
}

func TestCloudLogWriter(t *testing.T) {
	pub := new(testPublisher)
	w := CloudLogWriter{
		Publisher:    pub,
		sendTimeout:  100 * time.Millisecond,
		maxBatchSize: 10,
	}

	t.Run("publish on timeout", func(t *testing.T) {
		defer pub.reset()

		w.Write([]byte("hello world"))

		assert.Empty(t, pub.Events)

		<-time.After(200 * time.Millisecond)

		assert.Len(t, pub.Events, 1)
	})

	t.Run("publish on max batch size", func(t *testing.T) {
		defer pub.reset()

		for i := 0; i < w.maxBatchSize; i++ {
			w.Write([]byte(fmt.Sprintf("message %d", i)))
		}

		w.wg.Wait()

		assert.Len(t, pub.Events, 10)
	})
}

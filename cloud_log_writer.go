// Package cloudlog implements a simple writer used as a target for a logger. It
// batches messages before sending them to AWS CloudWatch Logs
package cloudlog

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
	"unicode/utf8"
)

// LogEvent contains a single log message
type LogEvent struct {
	Message   string
	Timestamp int64 // Time in milliseconds since epoch
}

// CloudLogPublisher defines the interface used to public a batch of log events
type CloudLogPublisher interface {
	PublishLogEvents(context.Context, []LogEvent) error
}

// CloudLogWriter provides a standard io.Writer interface for writing log
// messages. It batches the messages before sending them to the publisher either
// on a timeout, max size, or closing the writer.
type CloudLogWriter struct {
	Publisher CloudLogPublisher

	maxBatchSize      int
	sendTimeout       time.Duration
	sendTimeoutCancel context.CancelFunc

	mu     sync.Mutex
	wg     sync.WaitGroup
	buffer []LogEvent
}

// Create a new CloudLogWriter instance with the provided publisher
func NewWriter(pub CloudLogPublisher) *CloudLogWriter {
	if pub == nil {
		panic("must provide a CloudLogPublisher")
	}
	return &CloudLogWriter{
		Publisher:    pub,
		maxBatchSize: 25,
		sendTimeout:  10 * time.Second,
	}
}

var (
	// ErrInvalidUTF8 is returned when a log message contains invalid UTF-8
	ErrInvalidUTF8 = errors.New("invalid UTF-8")
)

func (w *CloudLogWriter) Write(p []byte) (int, error) {
	if !utf8.Valid(p) {
		return 0, ErrInvalidUTF8
	}

	err := w.writeEvent(context.Background(), LogEvent{
		Message:   string(p),
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	})

	return len(p), err
}

func (w *CloudLogWriter) writeEvent(ctx context.Context, e LogEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, e)

	w.unsafe_scheduleSendTimeout()

	if len(w.buffer) >= w.maxBatchSize {
		return w.unsafe_sendBuffer(ctx, sendBufferSourceMaxBatchSize)
	}

	return nil
}

func (w *CloudLogWriter) unsafe_cancelSendTimeout() {
	if w.sendTimeoutCancel != nil {
		w.sendTimeoutCancel()
		w.sendTimeoutCancel = nil
	}
}

func (w *CloudLogWriter) unsafe_scheduleSendTimeout() {
	ctx, cancel := context.WithCancel(context.Background())
	w.unsafe_cancelSendTimeout()
	w.sendTimeoutCancel = cancel

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.sendTimeout):
			w.mu.Lock()
			defer w.mu.Unlock()

			w.unsafe_sendBuffer(ctx, sendBufferSourceTimeout)
		}
	}()
}

const (
	sendBufferSourceTimeout = iota
	sendBufferSourceMaxBatchSize
	sendBufferSourceSend
)

func (w *CloudLogWriter) unsafe_sendBuffer(ctx context.Context, src int) error {
	if len(w.buffer) == 0 {
		return nil
	}

	w.unsafe_cancelSendTimeout()

	events := make([]LogEvent, len(w.buffer))
	copy(events, w.buffer)
	w.buffer = make([]LogEvent, 0, w.maxBatchSize)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// TODO: Handle error
		w.Publisher.PublishLogEvents(ctx, events)
	}()

	return nil
}

func (w *CloudLogWriter) CloseContext(ctx context.Context) error {
	if err := w.Send(ctx); err != nil {
		return err
	}

	w.wg.Wait()

	return nil
}

func (w *CloudLogWriter) Close() error {
	return w.CloseContext(context.Background())
}

func (w *CloudLogWriter) Send(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.unsafe_sendBuffer(ctx, sendBufferSourceSend)
}

var _ io.WriteCloser = (*CloudLogWriter)(nil)

func NewNullCloudLogPublisher() CloudLogPublisher {
	return new(nullCloudLogPublisher)
}

type nullCloudLogPublisher struct{}

func (*nullCloudLogPublisher) PublishLogEvents(context.Context, []LogEvent) error {
	return nil
}

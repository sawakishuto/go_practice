package eventlog

import (
	"context"
	"sync"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type RecordingPublisher struct {
	mu sync.Mutex
	events []book.ShelfEvent
}

func (r *RecordingPublisher) Publish(ctx context.Context, event book.ShelfEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)

	return nil
}

func NewRecordingPublisher() *RecordingPublisher {
	return &RecordingPublisher{}
}

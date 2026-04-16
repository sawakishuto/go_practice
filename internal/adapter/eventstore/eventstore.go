package eventstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type EventStore struct {
	store   map[string][]book.ShelfEvent
	versions map[string]int
	mu      sync.Mutex
}

func NewEventStore() *EventStore {
	return &EventStore{
		store:    make(map[string][]book.ShelfEvent),
		versions: make(map[string]int),
	}
}

func (es *EventStore) Append(ctx context.Context, streamID string, expectedVersion int, ev ...book.ShelfEvent) error {
	currentver := es.store[streamID]
	exver := expectedVersion
	es.mu.Lock()

	if len(currentver) != exver {
		return fmt.Errorf("不一致")
	}

	currentver = append(currentver,ev... )
	defer es.mu.Unlock()
	return nil

}

func (es *EventStore)Load(ctx context.Context, streamID string) ([]book.ShelfEvent, int, error) {
	es.mu.Lock()
	events := es.store[streamID]
	versions := es.versions[streamID]
	eventslice := make([]book.ShelfEvent, len(events))
	_ = copy(eventslice, events)
	return eventslice, versions, nil
}

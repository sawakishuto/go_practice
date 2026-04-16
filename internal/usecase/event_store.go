package usecase

import (
	"context"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type EventStore interface {
	Append(ctx context.Context, streamID string, expectedVersion int, ev ...book.ShelfEvent) error
	Load(ctx context.Context, streamID string) ([]book.ShelfEvent, int, error)
}

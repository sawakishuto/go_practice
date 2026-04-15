package usecase

import (
	"context"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type EventPublisher interface {
	Publish(ctx context.Context, event book.ShelfEvent) error
}

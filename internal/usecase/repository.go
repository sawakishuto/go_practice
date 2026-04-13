package usecase

import (
	"context"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type Repository interface {
	Save(ctx context.Context, b *book.Book) error
	FindByID(ctx context.Context, id string) (*book.Book, error)
}

package book

import "context"

type Repository interface {

	Save(ctx context.Context, b *Book) error
	FindByID(ctx context.Context, id string) (*Book, error)

}
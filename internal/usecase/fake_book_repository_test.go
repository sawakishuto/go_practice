package usecase

import (
	"context"
	"sync"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type FakeBookRepository struct {
	books map[string]*book.Book
	mu    sync.RWMutex
}

func NewFakeBookRepository() *FakeBookRepository {
	return &FakeBookRepository{
		books: make(map[string]*book.Book),
		mu:    sync.RWMutex{},
	}
}

func (r *FakeBookRepository) Save(ctx context.Context, b *book.Book) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	bk := *b
	r.books[bk.ID()] = &bk
	return nil
}

func (r *FakeBookRepository) FindByID(ctx context.Context, id string) (*book.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, exists := r.books[id]
	if !exists {
		return nil, book.BookNotFound
	}
	return found, nil
}
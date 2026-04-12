package memory

import (
	"context"
	"sync"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type BookRepository struct {

	mu sync.RWMutex
	books map[string]*book.Book

}

func NewBookRepository() *BookRepository {
	return &BookRepository{
		books: make(map[string]*book.Book),
	}
}

func (r *BookRepository) Save(ctx context.Context, b *book.Book) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	bk := *b
	r.books[bk.ID()] = &bk
	return nil
}

func (r *BookRepository) FindByID(ctx context.Context, id string) (*book.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, exists := r.books[id]
	if !exists {
		return nil, book.BookNotFound
	}
	return found, nil
}
package channelrepo

import (
	"context"
	"sync"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

type ChannelRepo struct {
	mu sync.RWMutex
	books map[string]*book.Book
	ch chan string
}

func NewChannelRepo() *ChannelRepo {
	reqChan := make(chan string)
	go func(i int) {

		for {
			select {
			case req := <-reqChan:
					if req == "save" {

					}

			}
		}
	}()
}
func (r *ChannelRepo) Save(ctx context.Context, b *book.Book) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	bk := *b
	r.books[bk.ID()] = &bk
	return nil
}

func (r *ChannelRepo) FindByID(ctx context.Context, id string) (*book.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, exists := r.books[id]
	if !exists {
		return nil, book.BookNotFound
	}
	return found, nil
}

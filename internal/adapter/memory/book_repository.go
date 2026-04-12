package memory

type BookRepository {

	mu sync.RWMutex
	books map[string]*book.Book

}

func NewBookRepository() *BookRepository {
	return &BookRepository{}
}

func (r *BookRepository) Save(ctx context.Context, b *book.Book) error {
	r,mu.Lock()
	defer mu.Unlock()
	book := *b
	r.books[book.ID()] = &book
	return nil
}

func (r *BookRepository) FindByID(ctx context.Context, id string) (*book.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	book, exists := r.books[id]
	if !exists {
		return nil, book.BookNotFound
	}
	return book, nil
}
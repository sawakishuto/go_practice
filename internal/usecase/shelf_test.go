package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

func TestShelfService_Register_Borrow_Return_flow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewFakeBookRepository()
	svc := NewShelfService(repo)

	id, err := svc.RegisterBook(ctx, "The Great Gatsby", "F. Scott Fitzgerald")
	if err != nil {
		t.Fatalf("RegisterBook: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	b, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID after register: %v", err)
	}
	if !b.IsAvailable() {
		t.Fatal("new book should be available")
	}

	if err := svc.BorrowBook(ctx, id); err != nil {
		t.Fatalf("BorrowBook: %v", err)
	}
	b, err = repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID after borrow: %v", err)
	}
	if b.IsAvailable() {
		t.Fatal("after borrow, book should not be available (repository returns a copy — refetch after Save)")
	}

	if err := svc.BorrowBook(ctx, id); !errors.Is(err, book.AlreadyBorrowed) {
		t.Fatalf("second BorrowBook: got %v, want AlreadyBorrowed", err)
	}

	if err := svc.ReturnBook(ctx, id); err != nil {
		t.Fatalf("ReturnBook: %v", err)
	}
	b, err = repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID after return: %v", err)
	}
	if !b.IsAvailable() {
		t.Fatal("after return, book should be available again")
	}

	if err := svc.ReturnBook(ctx, id); !errors.Is(err, book.NotBorrowed) {
		t.Fatalf("second ReturnBook: got %v, want NotBorrowed", err)
	}
}

func TestShelfService_BorrowBook_unknown_id(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := NewShelfService(NewFakeBookRepository())

	if err := svc.BorrowBook(ctx, "no-such-id"); !errors.Is(err, book.BookNotFound) {
		t.Fatalf("got %v, want BookNotFound", err)
	}
}

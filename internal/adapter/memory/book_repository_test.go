package memory

import (
	"context"
	"testing"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

func TestBookRepository_Save_and_FindByID(t *testing.T) {

	var title, err = book.NewTitle("スコットランド")
	if err != nil {
		t.Fatalf("book title is invalid")
	}

	book := book.NewBook("1", title, "F. Scott Fitzgerald")
	if book == nil {
		t.Errorf("Expected non-nil book, got nil")
	}
	repo := NewBookRepository()
	if repo == nil {
		t.Errorf("Expected non-nil repository, got nil")
	}
	err = repo.Save(context.Background(), book)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	foundBook, err := repo.FindByID(context.Background(), book.ID())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if foundBook == nil {
		t.Errorf("Expected non-nil book, got nil")
	}
	if foundBook.ID() != book.ID() {
		t.Errorf("Expected book ID to be %s, got %s", book.ID(), foundBook.ID())
	}
	if foundBook.Title() != book.Title() {
		t.Errorf("Expected book title to be %s, got %s", book.Title(), foundBook.Title())
	}
	if foundBook.Author() != book.Author() {
		t.Errorf("Expected book author to be %s, got %s", book.Author(), foundBook.Author())
	}

}

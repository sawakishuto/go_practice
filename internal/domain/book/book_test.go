package book

import (
	"errors"
	"testing"
)

// Step 2 の仕様を 1 本のテストにまとめた例（必要なら t.Run で分割してもよい）。
func TestBook_Borrow_and_Return_lifecycle(t *testing.T) {
	t.Parallel()

	var title, err = NewTitle("The Great Gatsby")
	if err != nil {
		t.Fatalf("thin book title is invalid")
	}

	b := NewBook("1", title, "F. Scott Fitzgerald")

	// 1. 新しい本は貸出可能であること。
	if !b.IsAvailable() {
		t.Fatal("new book should be available")
	}

	// 2. Borrow のあと貸出中になること。
	if err := b.Borrow(); err != nil {
		t.Fatalf("Borrow: %v", err)
	}
	if b.IsAvailable() {
		t.Fatal("after Borrow, book should not be available")
	}

	// 3. もう一度 Borrow すると AlreadyBorrowed になること（errors.Is で検証）。
	err = b.Borrow()
	if !errors.Is(err, AlreadyBorrowed) {
		t.Fatalf("second Borrow: got %v, want errors.Is(..., AlreadyBorrowed)", err)
	}

	// 4. Return で貸出可能に戻ること。
	if err := b.Return(); err != nil {
		t.Fatalf("Return: %v", err)
	}
	if !b.IsAvailable() {
		t.Fatal("after Return, book should be available again")
	}

	// 5. 貸出可能な状態で Return すると NotBorrowed になること。
	err = b.Return()
	if !errors.Is(err, NotBorrowed) {
		t.Fatalf("Return when available: got %v, want errors.Is(..., NotBorrowed)", err)
	}
}

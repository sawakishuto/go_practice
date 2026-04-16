package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

// ShelfService は蔵書まわりのユースケース（アプリケーションサービス）。
type ShelfService struct {
	repo Repository
	evpub EventPublisher
	evstore EventStore
}

// NewShelfService は ShelfService を構築する。
func NewShelfService(repo Repository, evpub EventPublisher, evstore EventStore) *ShelfService {
	return &ShelfService{repo: repo, evpub: evpub, evstore: evstore}
}

// RegisterBook は新しい本を登録し、採番した ID を返す。
func (s *ShelfService) RegisterBook(ctx context.Context, title, author string) (string, error) {
	id, err := newBookID()
	if err != nil {
		return "", fmt.Errorf("usecase: id: %w", err)
	}
	t, err := book.NewTitle(title)
	if err != nil {
		return "", fmt.Errorf("usecase: id: %w", err)

	}

	b := book.NewBook(id, t, author)
	if err := s.repo.Save(ctx, b); err != nil {
		return "", err
	}
	event := &book.BookRegistered{
		ID:         b.ID(),
		Title:      b.Title(),
		Author:     b.Author(),
		OccurredAt: time.Now(),
	}
	s.evpub.Publish(ctx, event)
	return b.ID(), nil
}

// BorrowBook は指定 ID の本を貸し出す。
func (s *ShelfService) BorrowBook(ctx context.Context, bookID string, evpub EventPublisher) error {

	b, err := s.repo.FindByID(ctx, bookID)
	if err != nil {
		return err
	}

	_ ,version, err := s.evstore.Load(ctx, bookID)
	if err != nil {
		return err
	}

	if err := b.Borrow(); err != nil {
		return err
	}

	err = s.repo.Save(ctx, b)
	if err != nil {
		return err
	}
	event := &book.BookBorrowed{
		ID:         b.ID(),
		OccurredAt: time.Now(),
	}

	err = s.evstore.Append(ctx, bookID, version, event)
	if err != nil {
		return err
	}
	err = s.evpub.Publish(ctx, event)
	if err != nil {
		return err
	}

	return nil
}

// ReturnBook は指定 ID の本を返却する。
func (s *ShelfService) ReturnBook(ctx context.Context, bookID string) error {
	b, err := s.repo.FindByID(ctx, bookID)
	if err != nil {
		return err
	}
	if err := b.Return(); err != nil {
		return err
	}
	return s.repo.Save(ctx, b)
}

func newBookID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

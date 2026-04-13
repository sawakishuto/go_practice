package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/sawakishuto/go_practice/internal/domain/book"
)

// ShelfService は蔵書まわりのユースケース（アプリケーションサービス）。
type ShelfService struct {
	repo book.Repository
}

// NewShelfService は ShelfService を構築する。
func NewShelfService(repo book.Repository) *ShelfService {
	return &ShelfService{repo: repo}
}

// RegisterBook は新しい本を登録し、採番した ID を返す。
func (s *ShelfService) RegisterBook(ctx context.Context, title, author string) (string, error) {
	id, err := newBookID()
	if err != nil {
		return "", fmt.Errorf("usecase: id: %w", err)
	}
	b := book.NewBook(id, title, author)
	if err := s.repo.Save(ctx, b); err != nil {
		return "", err
	}
	return b.ID(), nil
}

// BorrowBook は指定 ID の本を貸し出す。
func (s *ShelfService) BorrowBook(ctx context.Context, bookID string) error {
	b, err := s.repo.FindByID(ctx, bookID)
	if err != nil {
		return err
	}
	if err := b.Borrow(); err != nil {
		return err
	}
	return s.repo.Save(ctx, b)
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

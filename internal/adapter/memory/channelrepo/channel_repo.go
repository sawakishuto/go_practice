package channelrepo

import (
	"context"

	"github.com/sawakishuto/go_practice/internal/domain/book"
	"github.com/sawakishuto/go_practice/internal/usecase"
)

type opKind int

const (
	opSave opKind = iota
	opFindByID
)

// request は係員 goroutine へ渡す 1 件の仕事。
// リクエストの構造体
type request struct {
	kind   opKind
	book   *book.Book
	id     string
	errCh  chan error
	findCh chan findResult
}

//　結果の構造体
type findResult struct {
	b   *book.Book
	err error
}

// ChannelRepo は map を 1 本の goroutine（係員）だけが触るインメモリ実装。
// Mutex は使わない（直列化は channel による）。
// リクエストの構造体をチャネルに入れるための構造体
// channelは入れられたリクエストを受け取り側に送信する
type ChannelRepo struct {
	ops chan request
}

// NewChannelRepo は仕事用 channel を用意し係員を起動して返す。
// チャネルリポジトリのコンストラクタ
// リクエストを受け取るためのチャネルを作成してワーカーに渡す
func NewChannelRepo() *ChannelRepo {
	ops := make(chan request)
	// チャネルリポジトリの作成
	r := &ChannelRepo{ops: ops}
	// チャネルリポジトリのワーカーを起動する
	go r.worker(ops)
	return r
}
// ワーカーを発火させるとfor文で受け取りを開始する
func (r *ChannelRepo) worker(ops <-chan request) {
	books := make(map[string]*book.Book)
	for {
		req := <-ops
		switch req.kind {
		case opSave:
			bk := *req.book
			books[bk.ID()] = &bk
			req.errCh <- nil
		case opFindByID:
			b, ok := books[req.id]
			if !ok {
				req.findCh <- findResult{nil, book.BookNotFound}
				continue
			}
			req.findCh <- findResult{b, nil}
		}
	}
}

// Save は依頼を係員に送り、完了まで待つ。
func (r *ChannelRepo) Save(ctx context.Context, b *book.Book) error {
	errCh := make(chan error, 1)
	req := request{kind: opSave, book: b, errCh: errCh}
	select {
	case r.ops <- req:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// FindByID は依頼を係員に送り、結果まで待つ。
func (r *ChannelRepo) FindByID(ctx context.Context, id string) (*book.Book, error) {
	findCh := make(chan findResult, 1)
	req := request{kind: opFindByID, id: id, findCh: findCh}
	select {
	case r.ops <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case fr := <-findCh:
		return fr.b, fr.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var _ usecase.Repository = (*ChannelRepo)(nil)

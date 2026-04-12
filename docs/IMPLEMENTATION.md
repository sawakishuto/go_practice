# 実装方法（ファイル単位の具体手順）

[TRAINING.md](./TRAINING.md) の Step を **ファイルとコードの粒度** に落としたメモです。コピペ用の完成コードではなく、**何を書くか・よくあるバグ**を具体化します。

---

## 1. 実装の順番（Phase 1）

この順で進めると、途中で `go test` しやすいです。

| 順番 | ファイル | 理由 |
|------|----------|------|
| 1 | `internal/domain/book/errors.go` | 他パッケージから参照する定数が先に欲しい |
| 2 | `internal/domain/book/book_test.go`（失敗から） | ルールの仕様を固定する |
| 3 | `internal/domain/book/book.go` | テストを緑にする |
| 4 | `internal/domain/book/repository.go` | `Save` / `FindByID` の契約 |
| 5 | `internal/adapter/memory/book_repository.go` | 契約の実装 |
| 6 | `internal/adapter/memory/book_repository_test.go` | メモリ実装の検証 |
| 7 | `internal/usecase/shelf.go` | ユースケース |
| 8 | `internal/usecase/shelf_test.go` | 結合的な流れ |
| 9 | `cmd/shelf/main.go` | 手動確認 |

---

## 2. `errors.go` — 具体

- パッケージ名は `book`。
- **`var ErrName = errors.New("book: ...")`** のように、**メッセージにパッケージ接頭辞**を付けるとログで追いやすい。
- このプロジェクトの例: `AlreadyBorrowed`, `NotBorrowed`, `BookNotFound`。

テスト側では **`errors.Is(err, book.AlreadyBorrowed)`** のようにパッケージ名付きで参照する。

---

## 3. `book.go` — 具体

- **`type Book struct`** のフィールドは **非公開**（`id`, `isBorrowed` など）にし、外からは **`ID()`, `Borrow()`, `Return()`** などメソッドで操作する。
- **`NewBook(id, title, author string) *Book`** で **必ず貸出可能**の初期状態にする。
- **`Borrow` / `Return` は `*Book` レシーバ**（中身が変わるため）。値レシーバにするとコピーに対して変更してしまい、呼び出し元が変わらない。

```go
func (b *Book) Borrow() error {
    if b.isBorrowed {
        return AlreadyBorrowed
    }
    b.isBorrowed = true
    return nil
}
```

---

## 4. `repository.go` — 具体

```go
type Repository interface {
    Save(ctx context.Context, b *Book) error
    FindByID(ctx context.Context, id string) (*Book, error)
}
```

- 第一引数は **`ctx context.Context`**。インメモリでは `_ = ctx` でも、**シグネチャを揃える**と DB 実装に替えやすい。

---

## 5. `memory/book_repository.go` — 具体とハマりどころ

### 5.1 構造体とコンストラクタ

```go
type BookRepository struct {
    mu    sync.RWMutex
    books map[string]*book.Book
}

func NewBookRepository() *BookRepository {
    return &BookRepository{
        books: make(map[string]*book.Book),
    }
}
```

- **`make` を忘れると** `Save` 時に **`panic: assignment to entry in nil map`**。

### 5.2 ロック

- **読むだけ** `FindByID`: `RLock` / `RUnlock`。
- **書く** `Save`: `Lock` / `Unlock`。
- どちらも **`defer` で解放**すると `return` が多くても安全。

### 5.3 map の lookup

```go
found, ok := r.books[id]
if !ok {
    return nil, book.BookNotFound
}
return found, nil
```

- **`ok` が false** のときだけ NotFound。`found == nil` だけでは「キー無し」と区別できない場合がある。

### 5.4 変数名とパッケージ名

- ローカル変数を **`book` としない**（パッケージ `book` と衝突し、`book.BookNotFound` が壊れる）。**`bk`, `found`** などにする。

### 5.5 コピーして保存するパターン（現状のコード）

```go
bk := *b
r.books[bk.ID()] = &bk
```

- map の中身と呼び出し元の `*Book` を分離したいときの一例。テストでは **Save 後に状態を見るなら FindByID し直す**（[TESTING.md](./TESTING.md)）。

---

## 6. `usecase/shelf.go` — 具体

### 6.1 構造体で `repo` を持つ

```go
type ShelfService struct {
    repo book.Repository
}

func NewShelfService(repo book.Repository) *ShelfService {
    return &ShelfService{repo: repo}
}
```

- 各操作は **`func (s *ShelfService) RegisterBook(...)`** のようにメソッドにする。毎回 `repo` を引数で渡す **パッケージ関数**でも動くが、**依存が増えたときに拡張しづらい**。

### 6.2 `RegisterBook` の流れ（具体ステップ）

1. `id, err := newBookID()`（失敗時は `fmt.Errorf("usecase: id: %w", err)` などで文脈を足すとよい）。
2. `b := book.NewBook(id, title, author)` — ローカル変数名は **`b`** にしてパッケージ `book` と被らせない。
3. `return b.ID(), s.repo.Save(ctx, b)` のようにまとめてもよいが、**エラー時に ID を返さない**よう注意。

### 6.3 ID 生成（具体）

**やらない方がよい例:**

```go
buf := make([]byte, 10)
rand.Read(buf)
id := string(buf) // バイナリをそのまま string に。表示・デバッグに不向き
```

**よい例（このリポジトリと同型）:**

```go
buf := make([]byte, 8)
if _, err := rand.Read(buf); err != nil {
    return "", err
}
return hex.EncodeToString(buf), nil
```

### 6.4 `BorrowBook` / `ReturnBook` の型

どちらも同じ型:

1. `b, err := s.repo.FindByID(ctx, bookID)` → `err != nil` なら return。
2. `err = b.Borrow()` または `Return()` → `err != nil` なら return。
3. `return s.repo.Save(ctx, b)`。

---

## 7. `cmd/shelf/main.go` — 具体

```go
func main() {
    ctx := context.Background()
    repo := memory.NewBookRepository()
    svc := usecase.NewShelfService(repo)
    id, err := svc.RegisterBook(ctx, "タイトル", "著者")
    // err チェック、BorrowBook、ReturnBook、fmt.Println
}
```

- **`main` は依存の組み立てだけ**に近づけると読みやすい。

---

## 8. 整形と静的解析

```bash
gofmt -w .
go vet ./...
```

- import は **標準ライブラリ → 空行 → 第三者**（`gofmt` が整える）。

---

## 関連ドキュメント

- なぜこの分割か: [DESIGN.md](./DESIGN.md)
- テストの書き方: [TESTING.md](./TESTING.md)
- カリキュラム全体: [TRAINING.md](./TRAINING.md)

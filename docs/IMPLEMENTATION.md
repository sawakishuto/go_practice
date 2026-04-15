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
    repo book.Repository // Phase 1 の例。Phase 2 演習後は usecase 側で定義した interface（例: BookRepository）に差し替えることがある
}

func NewShelfService(repo book.Repository) *ShelfService {
    return &ShelfService{repo: repo}
}
```

- 各操作は **`func (s *ShelfService) RegisterBook(...)`** のようにメソッドにする。毎回 `repo` を引数で渡す **パッケージ関数**でも動くが、**依存が増えたときに拡張しづらい**。
- **ポートの型**を `book.Repository` にするか `usecase.BookRepository` にするかは [DESIGN.md](./DESIGN.md) §10 のトレードオフ参照。

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

## 7. 短変数宣言 `:=` と代入 `=`（エラー処理で毎回出る）

Go の **`:=`** は **新しい変数を宣言しつつ代入**する構文です。**左辺のうち少なくとも 1 つ**は、**そのスコープではまだ無い名前**でなければなりません。

### 7.1 典型的なパターン（ユースケース／`main`）

```go
id, err := svc.RegisterBook(ctx, title, author)
if err != nil {
    return "", err
}
err = svc.BorrowBook(ctx, id)
```

- **1 行目:** `id` と `err` の **両方を新しく宣言**しているので `:=` が使える。
- **2 行目以降:** `err` はすでに存在する。`BorrowBook` はたいてい **`error` だけ**返すので、**`err = ...`** と **代入だけ**する。

### 7.2 「`err` が `nil` だから `=`」は誤解

**いいえ。** 次に `=` を使う理由は **「`err` が nil かどうか」ではなく**、**同じブロックで `err` がすでに宣言済みだから**です。`err` にまだエラー値が入っていても、**新しい代入で上書き**するだけです。

### 7.3 `:=` だけ続けられない理由

同じブロックで `err` がある状態で **`err := svc.BorrowBook(...)`** と書くと、**新しい変数が 1 つも増えない短変数宣言**になり、コンパイルエラーになります（`=` にするか、**別名**を付けるか、**`_ , err := ...`** のように左辺に新しい名前を増やす）。

---

## 8. `cmd/shelf/main.go` — `package main` とエントリポイント

### 8.1 実行可能ファイルの条件

`go run ./cmd/shelf` や `go build` で **実行ファイル**にするには、そのディレクトリの Go ファイルが:

1. **`package main`** であること  
2. **`func main()`** が 1 つ定義されていること  

を満たす必要があります。`package shelf` のように **別名のパッケージ**のまま `func main()` があっても、ビルド対象は **「ライブラリ用パッケージ」** とみなされ、

```text
package github.com/.../cmd/shelf is not a main package
```

のようなエラーになります。**ディレクトリ名が `cmd/shelf` でも、パッケージ名は `main`** が正解です。

### 8.2 中身の責務

```go
func main() {
    ctx := context.Background()
    repo := memory.NewBookRepository()
    svc := usecase.NewShelfService(repo)
    id, err := svc.RegisterBook(ctx, "タイトル", "著者")
    // err チェック、BorrowBook、ReturnBook、fmt.Println / log
}
```

- **`main` は依存の組み立てと入出力**に近づけると読みやすい。長いドメインロジックは **`ShelfService` や `Book` に寄せる**。
- エラーは **`log.Fatal` / `fmt` + `os.Exit`** など、プロセスの方針に合わせて処理する（§7 の `:=` / `=` と組み合わせる）。

---

## 9. channel ベースの `BookRepository` 実装ガイド（係員モデル）

[DESIGN.md](./DESIGN.md) §8 の設計を、**ファイルに落とすときのチェックリスト**です。パッケージ名は例として `channelrepo`（**パッケージ名 `channel` は避ける** — 言語キーワード・組み込みと紛らわしい）。

### 9.1 全体像（責務の分割）

| 部品 | 責務 |
|------|------|
| **`ChannelRepo` struct** | 仕事用 **`chan request`**（など）を **フィールド**で持つ。外から見えるメソッドは **`Save` / `FindByID` だけ**（`usecase.Repository` 相当のシグネチャ）。 |
| **`New...()`** | `ops := make(chan request)`、**係員を `go r.worker(ops)` のように 1 回起動**、ポインタを **`return`**。**ここでは `Save` を呼ばない**。**`select` の無限ループは書かない**（係員側の仕事）。 |
| **係員 `worker`** | 引数で **`<-chan request`** を受け取り、**ローカル `map` またはクロージャで共有 map** を **この goroutine だけが触る**。`for { select { case req := <-ops: ... } }` または `for req := range ops` でループ。 |
| **`Save` / `FindByID`** | **引数に `chan` を増やさない**。メソッド内で **返信用 `chan` を `make(..., 1)`**、依頼 struct を組み立て、**`r.ops <- req`**。その後 **`select { case ... := <-reply: case <-ctx.Done(): }`** で返信またはキャンセル。 |

### 9.2 `request` struct に含めるとよいもの

- **操作種別**（`iota` や文字列で「保存」「取得」）。
- **ペイロード**（保存なら `*book.Book`、取得なら `id string`）。
- **返信先**（例: `errReply chan error`、`findReply chan findResult`）。係員は処理後に **一度だけ送る**。

### 9.3 コンストラクタでやってはいけないこと

- **`go func` の中だけで `reqChan := make(chan ...)`** して、struct のフィールドに載せない → **`Save` が同じ chan を参照できず**、係員と通信できない。
- **コンストラクタの末尾で `<-ch` してブロック** → **`return` できず**、呼び出し側がリポジトリを受け取れない。
- **コンストラクタ内で `Save` を実行** → 初期化と業務が混ざる。初期化は **空の map と channel と goroutine だけ**に留めるのが一般的。

### 9.4 係員が `req` を受け取ったあと

1. `switch req.kind` などで分岐。  
2. **map を読み書き**（この goroutine だけが触る）。  
3. **`req.errReply <- nil` や `req.findReply <- findResult{...}`** で返す。  
4. ループの先頭に戻り、**次の依頼を待つ**。

`case req := <-ch:` の **中身が空**のままだと、依頼は消化されず **呼び出し側が `<-reply` で永遠に待つ**などの不具合になります。

### 9.5 `context` と二段の `select`

- **第一段:** `r.ops <- req` と **`ctx.Done()`** — キューに載せるまでがキャンセル可能。
- **第二段:** 返信の受信と **`ctx.Done()`** — 係員が遅い／詰まったときに **呼び出し側が待ち続けない**ようにする。

### 9.6 Mutex を併用しない

map へのアクセスを **係員に一本化**するなら、**`Save` 内で `sync.Mutex` を取らない**のが筋です（[DESIGN.md](./DESIGN.md) §8.4）。Mutex 版は **`memory.BookRepository`** に任せ、channel 版は **メッセージだけ**、と役割を分ける。

### 9.7 コンパイル時チェック

```go
var _ usecase.Repository = (*ChannelRepo)(nil)
```

interface を満たしていなければ **ビルド時にエラー**になる。リネームやシグネ変更の安全網になる。

### 9.8 終了処理（テスト・長寿命プロセス）

係員が **`for {}` で永遠に回る**実装だと、テスト終了後も goroutine が残ることがあります。**`Close()`** で shutdown 用 channel を閉じる、`ops` を閉じて `range` を抜ける、**`sync.WaitGroup`** で終了待ち、など [TRAINING.md](./TRAINING.md) Phase 2 Step 5 に沿って足す。

---

## 10. 整形と静的解析

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

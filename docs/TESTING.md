# テスト（具体的に何をどこで書くか）

このリポジトリでは **レイヤーごとに検証するものが違う**。同じシナリオを何度も書かず、**一番内側のルールはドメインだけで完結**させる。

---

## 1. レイヤー別: 何をテストするか

| パッケージ | 主な目的 | 典型例（このプロジェクト） |
|------------|----------|----------------------------|
| `domain/book` | **ビジネスルール**だけ。DB もリポジトリも不要。 | 新規は貸出可能 / 二重 Borrow / 未貸出 Return |
| `adapter/memory` | **Repository 契約**を満たすか。`map`・ロックの前提。 | Save した ID が FindByID で取れる / 無い ID は BookNotFound |
| `usecase` | **手順のつながり**。ドメインとリポジトリを組み合わせた結果。 | 登録→借りる→二重借り→返す→二重返し / 存在しない ID で Borrow |

`cmd/shelf` は **手動 or 統合テスト**でよい。最初は `go run` で十分なことが多い。

---

## 2. ドメインのテスト（`book_test.go`）— 具体例

**検証すること:**

- `NewBook` 直後は `IsAvailable() == true`。
- `Borrow()` 後は貸出中。
- もう一度 `Borrow()` → `errors.Is(err, book.AlreadyBorrowed)`。
- `Return()` で再び貸出可能。
- 貸出可能なまま `Return()` → `errors.Is(err, book.NotBorrowed)`。

**ポイント:**

- ドメインは **`context` もリポジトリもいらない**。テストが短く保てる。
- エラーは **`errors.Is`** で種類を見る（ラップされても通しやすい）。

```go
err := b.Borrow()
if !errors.Is(err, AlreadyBorrowed) {
    t.Fatalf("got %v", err)
}
```

---

## 3. メモリリポジトリのテスト（`book_repository_test.go`）— 具体例

**最低限やること:**

1. `NewBookRepository()` を使う（**手で `&BookRepository{}` しない** → `map` が nil で panic しやすい）。
2. `Save` → `FindByID` で **同じ ID** が取れる。
3. 存在しない ID の `FindByID` が **`book.BookNotFound`**（`errors.Is` で検証するとよい）。

```go
ctx := context.Background()
repo := memory.NewBookRepository()
b := book.NewBook("id-1", "t", "a")
if err := repo.Save(ctx, b); err != nil {
    t.Fatal(err)
}
got, err := repo.FindByID(ctx, "id-1")
if err != nil {
    t.Fatal(err)
}
if got.ID() != "id-1" {
    t.Fatalf("got %q", got.ID())
}
_, err = repo.FindByID(ctx, "missing")
if !errors.Is(err, book.BookNotFound) {
    t.Fatalf("got %v", err)
}
```

---

## 4. ユースケースのテスト（`shelf_test.go`）— 具体例と落とし穴

### 4.1 おすすめの 1 本のシナリオ

1. `RegisterBook` → `id` が空でない。
2. `repo.FindByID(ctx, id)` で **貸出可能**であることを確認。
3. `BorrowBook(ctx, id)`。
4. **もう一度 `repo.FindByID`** して **貸出中**であることを確認。  
   ← ここを省略すると、`Save` がコピーを保持している実装では **古いポインタのまま** `IsAvailable()` が true のままになり、誤った期待になる。
5. もう一度 `BorrowBook` → `errors.Is(..., book.AlreadyBorrowed)`。
6. `ReturnBook` → 再 `FindByID` で貸出可能。
7. もう一度 `ReturnBook` → `errors.Is(..., book.NotBorrowed)`。

### 4.2 ネガティブパス（別テストでよい）

```go
svc := NewShelfService(memory.NewBookRepository())
err := svc.BorrowBook(context.Background(), "存在しないID")
if !errors.Is(err, book.BookNotFound) {
    t.Fatalf("got %v", err)
}
```

### 4.3 アサーションとメッセージ

- `if b.IsAvailable() { t.Fatal("...") }` の **メッセージは「なぜ失敗したか」** と一致させる。  
  例: 借りた直後に「まだ貸出可能だった」と書くと、読み手が迷わない。

---

## 5. `t.Parallel()` を付けるとき

- **テストごとに独立した `repo` / `svc`** を `New` するなら、並列でも壊れにくい。
- **グローバル変数や共有 map** を触るテストでは付けない。

```go
func TestSomething(t *testing.T) {
    t.Parallel()
    repo := memory.NewBookRepository()
    // ...
}
```

---

## 6. 実行コマンド（具体）

```bash
# 全部
go test ./...

# 競合検出（ロック漏れを疑うとき）
go test ./... -race

# カバレッジ
go test ./... -cover
```

---

## 7. チェックリスト（レビュー時）

- [ ] ドメインの失敗は **`errors.Is`** で見ているか。
- [ ] `Save` のあと状態を見るテストで **`FindByID` を再度呼んでいるか**。
- [ ] `BookRepository` を **`NewBookRepository` 経由**で作っているか。
- [ ] テストの失敗メッセージが **条件と矛盾していないか**。

---

## 関連ドキュメント

- レイヤー分担: [DESIGN.md](./DESIGN.md)
- ファイルの書き方: [IMPLEMENTATION.md](./IMPLEMENTATION.md)
- 手順: [TRAINING.md](./TRAINING.md)

---

## 今日の学び（2026-04-13）

### `sync.WaitGroup`

- **`wg.Add(1)`** は `go` する**直前（親側）**で足す。子 goroutine の中だけ `Add` するとタイミングがずれやすい。
- **`defer wg.Done()`** を各 `go func` の先頭付近で書き、**`wg.Wait()`** で「全員終わってから」検証する。`Wait()` のあとの処理は **子が `Done` したあと**にだけ走る（`err` が `nil` かどうかは無関係）。
- **ループ変数**は `go func(i int) { ... }(i)` のように **引数で渡す**（クロージャが同じ `i` を共有しないように）。

### テスト関数の名前

- `go test` が実行するのは **`Test` で始まる**関数だけ。`multiAccessFromUser` のような名前は **未使用扱い**になる。→ **`TestMultiAccessFromUser`** のようにする。

### `t.Fatal` / `t.Fatalf` と goroutine

- **子 goroutine から `t.Fatalf` を直接呼ばない**方が安全。エラーは **Mutex でスライスに集める**、**バッファ付き err chan** などに溜め、**`Wait()` 後**にメインのテスト goroutine で `t.Fatal` する。

### `go test ./... -race` と標準出力

- **`-race` は `Println` を禁止しない。** 成功テストの出力を見やすくするなら **`-v`** を付ける。テスト向けには **`t.Log` / `t.Logf`** も使いやすい。

### `t.Parallel()` との違い

- **`t.Parallel()`** は **別のテスト関数同士**の並列化。
- **1 つのテストの中で `go` を複数起動する**並行は **`WaitGroup`** 側の話。混同しない。

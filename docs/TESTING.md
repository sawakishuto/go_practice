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

**注意:** `t.Parallel()` は **「別の `Test...` 関数同士」** を同時に走らせるためのものです。**1 つのテストの中で複数クライアントをシミュレーションする**話とは別物です（§6・§7）。

---

## 6. 1 本のテストの中で並行させる — `go` と `sync.WaitGroup`

本番に近い「同時に複数リクエストが来る」状況は、**1 つの `Test...` の中で `go` を複数起動**して再現します。ここでは **`sync.WaitGroup`** で「全部終わってから検証する」までを揃えます。

### 6.1 `WaitGroup` の役割

- **`Add(n)`** … まだ終わっていない仕事を **n 件**増やした、と数える。
- **`Done()`** … 1 件片付いた（内部カウンタを 1 減らす）。通常 **`defer wg.Done()`** を各 `go func` の先頭で書く。
- **`Wait()`** … カウンタが **0 になるまでブロック**する。

`Wait()` を抜けたあとに書いたアサーションは、**`Add` した分だけ `Done` が呼ばれたあと**にだけ実行されます。つまり **「並行で投げた仕事が一通り return したあと」** に状態を検証できます。  
**よくある誤解:** 「`err` が `nil` だから次は `=` でいい」と同種の誤解で、「`err` が `nil` だから `Wait` が要る／要らないが決まる」わけではありません。`Wait` は **goroutine の終了同期**用です。

### 6.2 `Add` をどこで呼ぶか

**原則:** **`go` する直前**に親（テストのメインの流れ）が `Add` する。

子 goroutine の中だけ `Add(1)` すると、親がいち早く `Wait()` に入った瞬間にまだ `Add` されていない、という **レース**が起きやすいです。`go test -race` が怒るパターンのひとつです。

### 6.3 ループとクロージャ

`for i := 0; i < N; i++` の中で `go func() { ... i ... }()` と書くと、**全 goroutine が同じ `i` を参照し、ループ終了後の値になる**ことがあります。次のどちらかにします。

- **`go func(i int) { ... }(i)`** のように **引数でコピー**する。
- ループ内で **`ii := i`** して `go func() { ... ii ... }()` とする。

### 6.4 `WaitGroup` を値で渡さない

`WaitGroup` は **コピーしてはいけない**型です（ドキュメント明記）。`go func(wg sync.WaitGroup)` のように **値で渡さない**。外側の `var wg sync.WaitGroup` をクロージャが捕まえる形が普通です。

### 6.5 結果やエラーを集める

複数 `go` から **エラーを `t.Fatal` で落とす**場合、**子から直接 `t.Fatalf` しない**方が安全です（§8）。代わりに:

- **`sync.Mutex` で守った `[]error` に append** し、`Wait()` のあとにループして `t.Fatal`。
- **バッファ付き `chan error`** に送り、`Wait()` 後に `close` して drain する。

成功時だけの並行負荷なら、**件数や最終状態**だけ `Wait()` 後に検証してもよいです。

### 6.6 channel 版リポジトリと「依頼 struct / `errCh`」（テストで意識すること）

`internal/adapter/memory/channelrepo` のように **係員 goroutine + `ops chan request`** で map を守る実装では、`Save` の外側から見ると **普通の同期メソッド**です。内部では **依頼 struct** に **`errCh` などの返信用 channel を埋め込み**、`ops` に送ったあと **`<-errCh` で係員からの返事までブロック**します（「`errCh` を別ループが監視している」のではなく、**同じ呼び出し goroutine が受信で待つ**」点は [DESIGN.md](./DESIGN.md) §8.2 の補足、[IMPLEMENTATION.md](./IMPLEMENTATION.md) §9.2.1）。

並行テストでは **`WaitGroup` は「複数のクライアント `go` が一通り終わったか」**の同期に使い、**各 `Save` 内の `errCh` は呼び出しごとに作り捨て**であることは別レイヤです。`-race` は **係員と複数クライアント**のあいだのデータ競合がないかも含めて炙ります。

---

## 7. `t.Parallel()` と「テスト内の `go`」の使い分け

| | `t.Parallel()` | テスト内で複数 `go` + `WaitGroup` |
|---|----------------|-----------------------------------|
| **単位** | **別ファイル／別関数**の `TestXxx` 同士 | **1 つの `TestXxx` の内部** |
| **目的** | テストスイート全体の実行時間短縮 | 同一コンポーネントへの **同時アクセス**の再現 |
| **典型** | 各テストが独立した `repo` を `New` | `RegisterBook` を N 本の `go` から叩く |
| **同期** | フレームワークがテスト境界で制御 | 自分で `WaitGroup`（や channel） |

**両方を同じテストで使うことは可能**ですが、共有状態がないか余計に注意します。

---

## 8. テスト関数の命名と `testing.T` を goroutine から使うとき

### 8.1 `go test` が拾う名前

- **`Test` で始まり**、シグネチャが **`func TestXxx(t *testing.T)`** の関数だけが、引数なしで `go test` されたときに実行されます。
- `multiAccessFromUser` のような名前は **通常の関数**扱いになり、**未使用**と静的解析に怒られたり、**そもそもテストとして走りません**。→ **`TestMultiAccessFromUser`** にリネームします。

（`Benchmark...`、`Example...` は別ルールです。）

### 8.2 子 goroutine と `t.Fatal` / `t.Fatalf`

`FailNow` 系は **テストを走らせているメインの goroutine** から呼ぶのが安全、という説明が公式にあります。子 `go` から `t.Fatalf` すると、**パニックや不正な終了**に見えることがあります。

**推奨パターン:** 子は **エラーを返すチャネルや共有スライスに書くだけ**にし、**`wg.Wait()` 後**に親が `t.Fatalf` する。

---

## 9. 標準出力、`t.Log`、`-race` フラグ

### 9.1 `-race` は出力を殺さない

**`go test -race`** はデータ競合検出用のビルドを足すだけで、**`fmt.Println` を禁止しません**。「`-race` を付けたらログが出ない」と感じる場合は、別の理由です。

### 9.2 成功テストの出力が見えにくい理由

`go test` はデフォルトでは **成功したテストからの標準出力を省略**しがちです。ログを確実に見たいときは **`-v`（verbose）** を付けます。

```bash
go test ./... -race -v
```

### 9.3 `t.Log` / `t.Logf`

テストの文脈に紐づくログは **`t.Log`** の方が、`go test` の出力形式と相性がよいです。`-v` と組み合わせて読みます。

### 9.4 `-race` をいつ付けるか

**複数 goroutine から同じ `Repository` や `ShelfService` を触るテスト**を書いたら、**必ず** `go test ./... -race` を通す習慣にすると、Mutex 漏れや共有ポインタの問題に早く気づけます。

---

## 10. 実行コマンド（具体）

```bash
# 全部
go test ./...

# 競合検出（ロック漏れ・共有状態を疑うとき）
go test ./... -race

# 詳細ログ（t.Log や一部の stdout を見やすく）
go test ./... -v
go test ./... -race -v

# カバレッジ
go test ./... -cover
```

---

## 11. チェックリスト（レビュー時）

- [ ] ドメインの失敗は **`errors.Is`** で見ているか。
- [ ] `Save` のあと状態を見るテストで **`FindByID` を再度呼んでいるか**。
- [ ] `BookRepository` を **`NewBookRepository` 経由**で作っているか。
- [ ] テストの失敗メッセージが **条件と矛盾していないか**。
- [ ] **並行テスト**を書いたら **`go test -race`** を CI または手元で通しているか。
- [ ] 複数 `go` を使うテストで **`WaitGroup`（または同等の待ち合わせ）** があるか。
- [ ] 子 goroutine から **`t.Fatalf` していないか**（親で集約しているか）。

---

## 関連ドキュメント

- レイヤー分担: [DESIGN.md](./DESIGN.md)
- ファイルの書き方: [IMPLEMENTATION.md](./IMPLEMENTATION.md)
- 手順: [TRAINING.md](./TRAINING.md)

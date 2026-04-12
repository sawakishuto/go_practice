# Go × DDD トレーニングカリキュラム（自分で 1 から作る）

このリポジトリには **完成コードは置いていません**。`go.mod` とこのドキュメントだけを土台に、**あなたがファイルを追加**して進めます。

---

## 使い方

1. **Phase 1 のステップを上から順に** 実行する（飛ばさない）。
2. 各ステップの終わりで **`go test ./...` を実行**し、緑なら次へ。
3. 詰まったら、**どのステップか・エラーメッセージ全文** をメモして質問する。

---

## フェーズ一覧（先の見通し）

| Phase | テーマ |
|-------|--------|
| **1** | ドメインモデル・ユースケース・インメモリ永続化・`main`（このドキュメントで手順どおりに自作） |
| **2** | リポジトリとテストの切り離し、テーブル駆動テストの強化 |
| **3** | 値オブジェクト（`Title` など）、アダプタでの入出力変換 |
| **4** | ドメインイベント（任意） |
| **5** | HTTP API（`net/http` または `chi`） |
| **6** | DB 永続化（`database/sql` など） |
| **7** | 境界づけられたコンテキストでのパッケージ分割 |

Phase 2 以降の詳細は、Phase 1 が **`go test ./...` 通過**してから読み進めてください（後半に同梱）。

---

# Phase 1 — ゼロから「本棚」を作る

**ビジネスルール（仕様）**

- 本には **ID・タイトル・著者** がある。
- 本は **貸出可能** か **貸出中** のどちらか。
- **貸出可能** なときだけ「借りる」ことができる。すでに貸出中なら **エラー**。
- **貸出中** のときだけ「返す」ことができる。貸出可能なのに返すと **エラー**。
- アプリケーションは **登録・借りる・返す** ができる。永続化はまず **メモリ上** でよい。

**レイヤー（作るディレクトリ）**

- `internal/domain/book` … 用語とルール（エンティティ・ドメインエラー・リポジトリの **interface**）
- `internal/usecase` … アプリケーションサービス（例: `ShelfService`）
- `internal/adapter/memory` … 上記 interface のインメモリ実装
- `cmd/shelf` … `main` で動作確認

**ルール:** `internal/domain/book` からは **`net/http`・`database/sql`・外部ライブラリ** を import しない（ドメインを純粋に保つ）。

---

## Step 0 — モジュール

リポジトリに `go.mod` があることを確認する。なければ次を実行する（モジュールパスは自分の GitHub に合わせてよい）。

```bash
go mod init github.com/sawakishuto/go_practice
```

**完了条件:** `go.mod` が存在する。

---

## Step 1 — ドメインエラー

ファイル: `internal/domain/book/errors.go`  
パッケージ名: `book`

次の **意味** を表す、パッケージレベルの `var Err... = errors.New("...")` を定義する。

- すでに貸出中なのに借りようとした
- 貸出中でないのに返そとした
- リポジトリが指定 ID の本を見つけられない

**学ぶこと:** ドメインの「失敗の種類」を **型で区別**しやすくする（後で `errors.Is` する）。

**完了条件:** `go build ./internal/domain/book/...` が通る。

---

## Step 2 — エンティティ（テストファースト推奨）

ファイル: `internal/domain/book/book_test.go`  
先に **失敗するテスト** を書く。

1. 新しい本は **貸出可能** であること。
2. `Borrow` のあと **貸出中** になること。
3. もう一度 `Borrow` すると **Step 1 の「すでに貸出中」エラー** になること（`errors.Is` で検証）。
4. `Return` で **貸出可能** に戻ること。
5. 貸出可能な状態で `Return` すると **「貸出中でない」エラー** になること。

ファイル: `internal/domain/book/book.go`  
テストが通る最小実装を書く。

- コンストラクタ: `NewBook(id, title, author string) *Book` のような形でよい。
- 状態は **非公開フィールド** で持ち、必要ならゲッターだけ公開する。
- `Borrow` / `Return` は **ポインタレシーバ**（状態が変わるため）。

**ヒント:** 状態を `bool` でも `iota` の列挙でもよい。読みやすい方を選ぶ。

**完了条件:** `go test ./internal/domain/book/...` が緑。

---

## Step 3 — リポジトリ契約（interface）

ファイル: `internal/domain/book/repository.go`

`context.Context` を第一引数に取る `Repository` interface を定義する。最低限次を満たす。

- 本を **保存** する（新規・更新どちらも同じメソッドでよい）
- **ID** で本を **取得** する（なければ Step 1 の `ErrNotFound` を返す想定でよい）

**学ぶこと:** 永続化の「契約」はドメイン側に置くと、ユースケースが DB に依存しにくい。

**完了条件:** `go build ./internal/domain/book/...` が通る（実装クラスはまだなくてよい）。

---

## Step 4 — インメモリ実装

ファイル: `internal/adapter/memory/book_repository.go`  
パッケージ名: `memory`

`book.Repository` を満たす型を実装する。

- `map[string]*book.Book` のような構造で保持してよい。
- **複数 goroutine** から触る可能性を考え、`sync.RWMutex` で保護するとよい（`go test -race` の土台になる）。
- 保存・取得するとき、**呼び出し側が勝手に内部 map を書き換えない**よう、必要ならコピーを返す（設計の選択として比較してみる）。

**完了条件:** このパッケージ用に **短いテスト** を書き、`Save` → `FindByID` で同じ内容が取れること、存在しない ID で `ErrNotFound` になることを確認する。`go test ./internal/adapter/memory/...` が緑。

---

## Step 5 — ユースケース

ファイル: `internal/usecase/shelf.go`

- 構造体 `ShelfService` が `book.Repository` を **フィールドで受け取る**（コンストラクタ `NewShelfService(repo book.Repository) *ShelfService` など）。
- メソッド（すべて `ctx context.Context` を第一引数に）:
  - **登録**: タイトルと著者を受け取り、**新しい ID を採番**して本を保存し、**ID を返す**。採番は `crypto/rand` など **標準ライブラリ** でよい。
  - **借りる**: ID で取得 → ドメインの `Borrow` → 保存。
  - **返す**: 同様に `Return` → 保存。

ファイル: `internal/usecase/shelf_test.go`

- インメモリ実装を使い、**登録 → 借りる → 借りる（失敗）→ 返す → 返す（失敗）** の流れを 1 テストで検証する。
- 存在しない ID で借りようとすると `ErrNotFound` になるテストを書く。

**学ぶこと:** ユースケースは **フロー** を組み立て、ルールの中身はエンティティに任せる。

**完了条件:** `go test ./internal/usecase/...` が緑。

---

## Step 6 — エントリポイント

ファイル: `cmd/shelf/main.go`  
パッケージ `main`

- `memory.New...` と `usecase.NewShelfService` を組み立てる。
- 登録・借りる・返すを **数行の fmt.Println** で確認できるようにする（引数不要でよい）。

**完了条件:** `go run ./cmd/shelf` がパニックせず、期待どおりの文言が出る。

---

## Step 7 — 全体確認

```bash
go test ./... -race
go run ./cmd/shelf
```

**完了条件:** テストがすべて緑。ここまでで Phase 1 完了。

---

## Phase 1 修了後の伸ばししろ（任意）

- `ShelfService` に **一覧** を追加し、`Repository` に必要なメソッドを増やす。
- ドメインのエラーを **`fmt.Errorf("...: %w", err)`** でラップする箇所と、ラップしない箇所を比較する。

---

## Phase 2 以降（概要）

**Phase 2:** `Repository` interface を `usecase` 側に移してみるなど、**インターフェースの置き場所** を変えて同じテストが通るか試す。フェイク実装を別ファイルに切り出す。

**Phase 3:** `Title` を `NewTitle(string) (Title, error)` で検証する値オブジェクトにする。

**Phase 5 以降:** HTTP 層を `internal/adapter/http` に追加し、DTO からユースケースへ変換する。

（詳細は Phase 1 完了後に、このファイルへ追記するか、別ドキュメントに分けてもよい。）

---

## よく使うコマンド

```bash
go test ./...
go test ./... -race
go test ./... -cover
go run ./cmd/shelf
```

# Go × DDD トレーニングカリキュラム（自分で 1 から作る）

このリポジトリには **完成コードは置いていません**。`go.mod` とこのドキュメントだけを土台に、**あなたがファイルを追加**して進めます。

---

## 使い方

1. **Phase 1 のステップを上から順に** 実行する（飛ばさない）。
2. 各ステップの終わりで **`go test ./...` を実行**し、緑なら次へ。
3. 詰まったら、**どのステップか・エラーメッセージ全文** をメモして質問する。

**深掘りの置き場所:** このファイルは **ゴールと順番**が中心です。設計の比較（Mutex と channel、ポートの置き場）、並行テスト（`WaitGroup`、`t.Parallel` の違い、`-race`）、Go の文法（`package main`、`:=` と `=`）、channelrepo の具体分担は、次のドキュメントの **通常の章**に統合してあります。

| 読みたい内容 | 参照 |
|--------------|------|
| Mutex 版と channel 版、actor／コンストラクタと係員、ポートをドメイン／ユースケースのどちらに置くか | [DESIGN.md](./DESIGN.md) §8〜§10 |
| テスト内の `go` と `WaitGroup`、`t.Parallel` との違い、`-v` と `-race`、子 goroutine と `t.Fatal` | [TESTING.md](./TESTING.md) §5〜§9 |
| `cmd/shelf` の `package main`、`:=` / `=`、channel ベースリポジトリの実装チェックリスト | [IMPLEMENTATION.md](./IMPLEMENTATION.md) §7〜§9 |

---

## フェーズ一覧（先の見通し）

| Phase | テーマ |
|-------|--------|
| **1** | ドメインモデル・ユースケース・インメモリ永続化・`main`（このドキュメントで手順どおりに自作） |
| **2** | ポートの切り出し、テーブル駆動テスト、**goroutine / channel** による並行アクセスの実践 |
| **3** | 値オブジェクト（`Title` など）、アダプタでの入出力変換 |
| **4** | ドメインイベント（任意） |
| **5** | HTTP API（`net/http` または `chi`） |
| **6** | DB 永続化（`database/sql` など） |
| **7** | 境界づけられたコンテキストでのパッケージ分割 |

Phase 2 の詳細は **このファイルの「Phase 2」節**。Phase 3 以降の概要はその直後にあります。Phase 1 が **`go test ./...` 通過**してから Phase 2 に進んでください。

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
パッケージ **`main`**（ディレクトリ名が `shelf` でも、**パッケージ名は `main`**。`package shelf` のままだと `go run` で **not a main package** になる。詳細は [IMPLEMENTATION.md](./IMPLEMENTATION.md) §8）。

- `memory.New...` と `usecase.NewShelfService` を組み立てる。
- 登録・借りる・返すを **数行の fmt.Println** で確認できるようにする（引数不要でよい）。
- 最初の操作は **`id, err := ...`**、続く操作は **`err = ...`** とすることが多い（理由は [IMPLEMENTATION.md](./IMPLEMENTATION.md) §7）。

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

# Phase 2 — リポジトリ契約の切り離し、テーブル駆動テスト、並行処理

この Phase は **6 つの Step** です。難しい言葉が出てきたら、下の「用語メモ」を先に読んでも大丈夫です。

**用語メモ（読み飛ばして OK、詰まったら戻る）**

| 言葉 | ざっくり意味 |
|------|----------------|
| **interface（インターフェース）** | 「こういうメソッドがあれば使える」という**約束書**。中身の実装は別。 |
| **ポート** | このカリキュラムでは、ユースケース側の interface のこと（永続化への窓口）。 |
| **フェイク** | 本物の DB や `memory` ではなく、**テスト用に手軽に作った実装**。 |
| **テーブル駆動テスト** | テストケースを **スライスの表** に並べて、`for` で回してまとめて試す書き方。 |
| **goroutine** | `go` で起動する **別スレッドのような処理**。複数が同時に動き得る。 |
| **`go test -race`** | 「同時に触って壊れていないか」を調べる **データ競合チェック**。付けて実行する。 |
| **channel（chan）** | goroutine 同士が **データを渡すための管**。片方が送り、片方が受け取る。 |
| **Mutex** | 「今は一人だけ」札。**同じデータを同時にいじらない**ための鍵。 |

**この Phase のゴール（できるようになること）**

1. ユースケースのテストが、**いつも `memory` パッケージに頼らなくても**書ける（フェイクで十分になる）。
2. テストを **表形式** にまとめて、ケースを足しやすくする。
3. **複数の goroutine** が同じリポジトリを触るテストを書き、**`-race` で問題が出ない**ようにする。
4. **channel** を使って「**データは一人の goroutine だけが触る**」リポジトリをもう一つ作り、Phase 1 の Mutex 版と **見比べる**。

**前提:** Phase 1 の `go test ./... -race` が通っていること。

---

## Step 1 — ユースケース側へ「ポート」を移す

**なぜ？**  
いま `Repository` は `internal/domain/book` にあります。ここでは一度、**「本を保存・取得する約束」** を **`internal/usecase` に移す**練習をします（名前は `BookRepository` などでよい）。

**やること（順番どおり）**

1. `internal/usecase` に **interface** を書く。中身は Phase 1 の `Repository` と同じでよい（`Save` / `FindByID`、`context` と `*book.Book` を使う）。
2. `ShelfService` が持つフィールドの型を、その **interface** に変える。
3. `internal/domain/book` の **`repository.go` は削除**する（ドメインには「本」と「エラー」だけ残すイメージ）。
4. `internal/adapter/memory` の `BookRepository` が、まだ同じ約束を満たしているか確認する。  
   わかりやすい確認方法: ファイルのどこかに次の **1 行** を書くと、満たしていなければ **ビルド時にエラー**になる。

   ```go
   var _ usecase.BookRepository = (*BookRepository)(nil)
   ```

**ひとことで:** 「約束書（interface）を **使う側の近く** に置く」Go の書き方に慣れる。

**完了条件:** `go build ./...` が通る。

---

## Step 2 — テスト用フェイクを別ファイルへ

**なぜ？**  
`ShelfService` のテストが **`memory` を import している**と、「ユースケースのテスト」と「本番用のインメモリ実装」がくっつきます。テストだけで使う **軽い実装（フェイク）** に分けます。

**やること**

- `fake_book_repository_test.go` のような **別ファイル** に、map で本を覚えておくだけの実装を書く（`_test.go` なら本番ビルドに乗らない）。
- `shelf_test.go` から **`internal/adapter/memory` の import をなくす**。

フェイクの中に Mutex を入れるかは、**そのテストが一人で動くだけ**なら省略でもよい。

**完了条件:** `go test ./internal/usecase/...` が緑。

---

## Step 3 — テーブル駆動テストへ寄せる

**なぜ？**  
「成功」「失敗」「別の理由で失敗」…とテストが増えると、`if` のコピペが増えがちです。**表にして `for` で回す**と、あとからケースを 1 行足すだけで済みます。

**やること（どちらか、または両方）**

- **`book_test.go`:** `Borrow` / `Return` のパターンを `tests := []struct { name string; ... }{ ... }` にまとめ、`for` + `t.Run(tt.name, ...)` で実行する。
- **`book_repository_test.go`（memory）:** 同じように、ID や結果が違うケースを表に並べる。

**完了条件:** `go test ./... -race` が緑。

---

## Step 4 — 並行アクセスをテストする（goroutine）

**なぜ？**  
実アプリでは **同時に複数リクエスト** が来ます。一人ずつしかテストしていないと、**同時に触ると壊れるバグ**を見逃します。

**やること**

1. テストの中で **`go` を何度か使い**、`Save` / `FindByID` や `RegisterBook` を **同時に** 呼ぶ。
2. **`sync.WaitGroup`** で「全部の `go` が終わるまで待つ」。待たずにテストが終わると、裏でまだ動いているのに **検証してしまう**ので注意。
3. **`go test ./... -race`** を必ず通す。エラーが出たら、map や `*Book` を **鍵なしで複数 goroutine が触っていないか** を疑う。

**ちがいのメモ:** `t.Parallel()` は **別のテスト関数同士** を並べるもの。いまやっているのは **1 つのテストの中で `go` をたくさん使う**話。`WaitGroup` の詳細・`t.Fatal` の扱い・`-v` は [TESTING.md](./TESTING.md) §6〜§9。

**完了条件:** `go test ./... -race` が緑。

---

## Step 5 — channel ベースのリポジトリ実装

**なぜ？**  
Phase 1 の `memory` は **Mutex（鍵）** で「同時に map を触らない」ようにしました。Step 5 では **鍵ではなく channel** で同じ目的を達する練習です。

**たとえ話でいうと**

- **map（本の棚）の前には常に同じ係員が一人。** 係員以外は棚に手を伸ばさない。
- ほかの人（`Save` を呼んだコード）は、**窓口のトレー（channel）に「保存して」「取って」と書いた紙を載せる**だけ。
- 係員が map を触り、**結果をまた channel で返す**。

**channel とは（超短く）**  
「値を送る」と「受け取る」が **決まった順番** でつながる **管** です。複数 goroutine が同じ map を直接いじらない代わりに、**管を通して依頼と返事**をやり取りします。

**外から見た形**  
`Save` や `FindByID` の **引数と戻り値は Phase 1 と同じ** のままにします。中身だけ「係員 goroutine に channel で頼む」に変わります。

**流れ（`FindByID` が呼ばれたとき）**

```text
呼び出し側                    係員（専用 goroutine 1 本）
    |                                  |
    |  「ID 〇〇を取って」と chan に送る  |
    | ----------------------------->   |  map を読む
    |                                  |
    |  結果を chan で受け取る            |
    | <-----------------------------   |
    |  return
```

`Save` も「依頼 → 係員が map を更新 → 結果を返す」です。

---

**やること（初心者向け・順番固定）**

1. **新しいフォルダとパッケージ**  
   例: `internal/adapter/channelrepo`。パッケージ名だけは **`channel` にしない**（Go の言葉と同じで混乱するため）。

2. **係員 goroutine を 1 本起動**  
   `NewBookRepository()` のような関数の中で `go func() { ... }()` を使う。  
   `for { select { case req := <-仕事用のchan: ... } }` のように、**ずっと仕事を待つループ**にする（`select` は「いくつかの待ちのうち、**先に来た方**」を処理するための書き方）。

3. **`Save` / `FindByID` の中身**  
   - 依頼内容を struct にまとめる（種類: 保存 or 取得、ID、本のデータなど）。  
   - **おすすめの形:** その struct の中に **`reply chan 結果の型`** を入れておく。係員が処理したあと **`reply <- 結果`**、呼び出し側は **`<-reply` で待つ**。  
   - `context` を使っている場合、返事を待つときに **`select`** で `ctx.Done()` も見ると、「時間切れ・キャンセル」で待ち続けずに抜けられる。

4. **終了処理（忘れがち）**  
   テストが終わっても係員が **ずっとループ**しているとまずいので、`Close()` などで **「もう閉店」** を伝え、係員の `for` を抜ける。完全に終わるまで待つなら `WaitGroup` を使ってもよい。

5. **約束を守れているか**  
   Step 1 と同様、`var _ usecase.BookRepository = (*あなたの型)(nil)` で **コンパイルに確認**させる。

6. **動作確認**  
   普通のテストに加え、Step 4 と同じく **複数 `go` から同時に** 叩く。`go test ./... -race` が緑ならよし。

**つまずきやすいところ**

- **`<-reply` で永遠に待つ** → 係員側が `reply` に一度も送っていない、チャネルの向きが逆、など。
- **テスト終了後も goroutine が残る** → `Close` を書く／テストで `defer repo.Close()` する。
- **コンストラクタで `select` や `<-ch` だけして return しない** → 係員と `chan` の接続・実装手順は [IMPLEMENTATION.md](./IMPLEMENTATION.md) §9、[DESIGN.md](./DESIGN.md) §8〜§9。

**この Step で掴みたいこと**

- map を **直接奪い合わず**、channel で「依頼と返事」にする考え方。
- 外からは **普通の関数**、中は **1 本の goroutine** という分け方。
- Mutex 版と **どちらが読みやすいか** は、人と規模による（どちらが正解、ではない）。

**完了条件:** `go test ./... -race` が緑。`memory`（Mutex）と channel 版の **両方** が同じ `BookRepository` を満たしていること。

---

## Step 6 — （任意）`t.Run` とエラー検証の整理

テスト名（`t.Run("...", ...)`）を読めば **何を確認しているか** わかるようにする。`errors.Is` でエラーの種類を見ているなら、**期待するエラーと実際のエラー**がずれていないか、コメントで一言足してもよい。

---

## Phase 2 修了条件

```bash
go test ./... -race
```

すべて緑で Phase 2 完了。次は **Phase 3（値オブジェクト）** に進む（概要は下記）。

---

## Phase 3 以降（概要）

**Phase 3:** `Title` を `NewTitle(string) (Title, error)` で検証する値オブジェクトにする。アダプタでの入出力変換を足す。

**Phase 4:** ドメインイベント（任意）。

**Phase 5 以降:** HTTP 層を `internal/adapter/http` に追加し、DTO からユースケースへ変換する。

**Phase 6–7:** DB 永続化、境界づけられたコンテキストでのパッケージ分割（フェーズ一覧表を参照）。

---

## よく使うコマンド

```bash
go test ./...
go test ./... -race
go test ./... -cover
go run ./cmd/shelf
```

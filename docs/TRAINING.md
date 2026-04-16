# Go × DDD トレーニングカリキュラム（自分で 1 から作る）

このリポジトリには **完成コードは置いていません**。`go.mod` とこのドキュメントだけを土台に、**あなたがファイルを追加**して進めます。

---

## 使い方

1. **Phase 1 のステップを上から順に** 実行する（飛ばさない）。
2. 各ステップの終わりで `**go test ./...` を実行**し、緑なら次へ。
3. 詰まったら、**どのステップか・エラーメッセージ全文** をメモして質問する。

**深掘りの置き場所:** このファイルは **ゴールと順番**が中心です。設計の比較（Mutex と channel、ポートの置き場）、並行テスト（`WaitGroup`、`t.Parallel` の違い、`-race`）、Go の文法（`package main`、`:=` と `=`）、channelrepo の具体分担は、次のドキュメントの **通常の章**に統合してあります。


| 読みたい内容                                                                            | 参照                                             |
| --------------------------------------------------------------------------------- | ---------------------------------------------- |
| Mutex 版と channel 版、actor／コンストラクタと係員、ポートをドメイン／ユースケースのどちらに置くか                       | [DESIGN.md](./DESIGN.md) §8〜§10                |
| テスト内の `go` と `WaitGroup`、`t.Parallel` との違い、`-v` と `-race`、子 goroutine と `t.Fatal` | [TESTING.md](./TESTING.md) §5〜§9               |
| `cmd/shelf` の `package main`、`:=` / `=`、channel ベースリポジトリの実装チェックリスト                | [IMPLEMENTATION.md](./IMPLEMENTATION.md) §7〜§9 |
| ドメインイベント・イベントソーシング・MQ の用語とトレードオフ                                                  | [EVENTS.md](./EVENTS.md)                       |


---

## フェーズ一覧（先の見通し）


| Phase | テーマ                                                                                      |
| ----- | ---------------------------------------------------------------------------------------- |
| **1** | ドメインモデル・ユースケース・インメモリ永続化・`main`（このドキュメントで手順どおりに自作）                                        |
| **2** | ポートの切り出し、テーブル駆動テスト、**goroutine / channel** による並行アクセスの実践                                  |
| **3** | 値オブジェクト（`Title` など）、アダプタでの入出力変換                                                          |
| **4** | **ドメインイベント**、**イベントストア（追記・リプレイ）**、**メッセージ流し（インメモリ MQ 代用）**（[EVENTS.md](./EVENTS.md) と併読） |
| **5** | HTTP API（`net/http` または `chi`）                                                           |
| **6** | DB 永続化（`database/sql` など）                                                                |
| **7** | 境界づけられたコンテキストでのパッケージ分割                                                                   |


Phase 2 の詳細は **「Phase 2」節**、Phase 3 は **「Phase 3」節**、Phase 4 は **「Phase 4」節**。概念の背景は **[EVENTS.md](./EVENTS.md)**。Phase 1 が `**go test ./...` 通過**してから Phase 2 → Phase 3 → Phase 4 の順がおすすめです。

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

**ルール:** `internal/domain/book` からは `**net/http`・`database/sql`・外部ライブラリ** を import しない（ドメインを純粋に保つ）。

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
パッケージ `**main`**（ディレクトリ名が `shelf` でも、**パッケージ名は `main`**。`package shelf` のままだと `go run` で **not a main package** になる。詳細は [IMPLEMENTATION.md](./IMPLEMENTATION.md) §8）。

- `memory.New...` と `usecase.NewShelfService` を組み立てる。
- 登録・借りる・返すを **数行の fmt.Println** で確認できるようにする（引数不要でよい）。
- 最初の操作は `**id, err := ...`**、続く操作は `**err = ...`** とすることが多い（理由は [IMPLEMENTATION.md](./IMPLEMENTATION.md) §7）。

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
- ドメインのエラーを `**fmt.Errorf("...: %w", err)`** でラップする箇所と、ラップしない箇所を比較する。

---

# Phase 2 — リポジトリ契約の切り離し、テーブル駆動テスト、並行処理

この Phase は **6 つの Step** です。難しい言葉が出てきたら、下の「用語メモ」を先に読んでも大丈夫です。

**用語メモ（読み飛ばして OK、詰まったら戻る）**


| 言葉                      | ざっくり意味                                       |
| ----------------------- | -------------------------------------------- |
| **interface（インターフェース）** | 「こういうメソッドがあれば使える」という**約束書**。中身の実装は別。         |
| **ポート**                 | このカリキュラムでは、ユースケース側の interface のこと（永続化への窓口）。  |
| **フェイク**                | 本物の DB や `memory` ではなく、**テスト用に手軽に作った実装**。    |
| **テーブル駆動テスト**           | テストケースを **スライスの表** に並べて、`for` で回してまとめて試す書き方。 |
| **goroutine**           | `go` で起動する **別スレッドのような処理**。複数が同時に動き得る。       |
| `**go test -race`**     | 「同時に触って壊れていないか」を調べる **データ競合チェック**。付けて実行する。   |
| **channel（chan）**       | goroutine 同士が **データを渡すための管**。片方が送り、片方が受け取る。  |
| **Mutex**               | 「今は一人だけ」札。**同じデータを同時にいじらない**ための鍵。            |


**この Phase のゴール（できるようになること）**

1. ユースケースのテストが、**いつも `memory` パッケージに頼らなくても**書ける（フェイクで十分になる）。
2. テストを **表形式** にまとめて、ケースを足しやすくする。
3. **複数の goroutine** が同じリポジトリを触るテストを書き、`**-race` で問題が出ない**ようにする。
4. **channel** を使って「**データは一人の goroutine だけが触る**」リポジトリをもう一つ作り、Phase 1 の Mutex 版と **見比べる**。

**前提:** Phase 1 の `go test ./... -race` が通っていること。

---

## Step 1 — ユースケース側へ「ポート」を移す

**なぜ？**  
いま `Repository` は `internal/domain/book` にあります。ここでは一度、**「本を保存・取得する約束」** を `**internal/usecase` に移す**練習をします（名前は `BookRepository` などでよい）。

**やること（順番どおり）**

1. `internal/usecase` に **interface** を書く。中身は Phase 1 の `Repository` と同じでよい（`Save` / `FindByID`、`context` と `*book.Book` を使う）。
2. `ShelfService` が持つフィールドの型を、その **interface** に変える。
3. `internal/domain/book` の `**repository.go` は削除**する（ドメインには「本」と「エラー」だけ残すイメージ）。
4. `internal/adapter/memory` の `BookRepository` が、まだ同じ約束を満たしているか確認する。
  わかりやすい確認方法: ファイルのどこかに次の **1 行** を書くと、満たしていなければ **ビルド時にエラー**になる。

**ひとことで:** 「約束書（interface）を **使う側の近く** に置く」Go の書き方に慣れる。

**完了条件:** `go build ./...` が通る。

---

## Step 2 — テスト用フェイクを別ファイルへ

**なぜ？**  
`ShelfService` のテストが `**memory` を import している**と、「ユースケースのテスト」と「本番用のインメモリ実装」がくっつきます。テストだけで使う **軽い実装（フェイク）** に分けます。

**やること**

- `fake_book_repository_test.go` のような **別ファイル** に、map で本を覚えておくだけの実装を書く（`_test.go` なら本番ビルドに乗らない）。
- `shelf_test.go` から `**internal/adapter/memory` の import をなくす**。

フェイクの中に Mutex を入れるかは、**そのテストが一人で動くだけ**なら省略でもよい。

**完了条件:** `go test ./internal/usecase/...` が緑。

---

## Step 3 — テーブル駆動テストへ寄せる

**なぜ？**  
「成功」「失敗」「別の理由で失敗」…とテストが増えると、`if` のコピペが増えがちです。**表にして `for` で回す**と、あとからケースを 1 行足すだけで済みます。

**やること（どちらか、または両方）**

- `**book_test.go`:** `Borrow` / `Return` のパターンを `tests := []struct { name string; ... }{ ... }` にまとめ、`for` + `t.Run(tt.name, ...)` で実行する。
- `**book_repository_test.go`（memory）:** 同じように、ID や結果が違うケースを表に並べる。

**完了条件:** `go test ./... -race` が緑。

---

## Step 4 — 並行アクセスをテストする（goroutine）

**なぜ？**  
実アプリでは **同時に複数リクエスト** が来ます。一人ずつしかテストしていないと、**同時に触ると壊れるバグ**を見逃します。

**やること**

1. テストの中で `**go` を何度か使い**、`Save` / `FindByID` や `RegisterBook` を **同時に** 呼ぶ。
2. `**sync.WaitGroup`** で「全部の `go` が終わるまで待つ」。待たずにテストが終わると、裏でまだ動いているのに **検証してしまう**ので注意。
3. `**go test ./... -race`** を必ず通す。エラーが出たら、map や `*Book` を **鍵なしで複数 goroutine が触っていないか** を疑う。

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

**よくある疑問（チャットで確認した内容の要約）**

- **Q: リクエスト用の struct と、リクエストを送り合う channel があるってこと？**  
**A:** 大きく **2 段**です。（1）**依頼の中身**をまとめた `request` のような **struct**。（2）それを係員に届ける `**ops` のような仕事用 channel**。（3）さらに **「この呼び出し専用の返事」**を係員から受け取る `**errCh` や `findCh`** は、**struct のフィールドとして依頼に同梱**し、係員が処理のあと **そこへ一度だけ送る**、という形が多いです（実装例: `internal/adapter/memory/channelrepo`）。
- **Q: `errCh` は、依頼が送られたあと結果を返すときに依頼の中の `errCh` に入れるから、それを検知して次の処理が走る？**  
**A:** 「検知」というより **同じ goroutine が `<-errCh` で待っていて、係員が `req.errCh <- nil` したら受信が完了して次の行に進む**、という **同期の待ち合わせ**です。イベントループが別スレッドで `errCh` を見張っている、というモデルではありません（外から見た `Save` は **普通のブロッキング関数**のまま）。

---

**やること（初心者向け・順番固定）**

1. **新しいフォルダとパッケージ**
  例: `internal/adapter/channelrepo`。パッケージ名だけは `**channel` にしない**（Go の言葉と同じで混乱するため）。
2. **係員 goroutine を 1 本起動**
  `NewBookRepository()` のような関数の中で `go func() { ... }()` を使う。  
   `for { select { case req := <-仕事用のchan: ... } }` のように、**ずっと仕事を待つループ**にする（`select` は「いくつかの待ちのうち、**先に来た方**」を処理するための書き方）。
3. `**Save` / `FindByID` の中身**
  - 依頼内容を struct にまとめる（種類: 保存 or 取得、ID、本のデータなど）。  
  - **おすすめの形:** その struct の中に `**reply chan 結果の型`** を入れておく。係員が処理したあと `**reply <- 結果`**、呼び出し側は `**<-reply` で待つ**。  
  - `context` を使っている場合、返事を待つときに `**select`** で `ctx.Done()` も見ると、「時間切れ・キャンセル」で待ち続けずに抜けられる。
4. **終了処理（忘れがち）**
  テストが終わっても係員が **ずっとループ**しているとまずいので、`Close()` などで **「もう閉店」** を伝え、係員の `for` を抜ける。完全に終わるまで待つなら `WaitGroup` を使ってもよい。
5. **約束を守れているか**
  Step 1 と同様、`var _ usecase.BookRepository = (*あなたの型)(nil)` で **コンパイルに確認**させる。
6. **動作確認**
  普通のテストに加え、Step 4 と同じく **複数 `go` から同時に** 叩く。`go test ./... -race` が緑ならよし。

**つまずきやすいところ**

- `**<-reply` で永遠に待つ** → 係員側が `reply` に一度も送っていない、チャネルの向きが逆、など。
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

すべて緑で Phase 2 完了。次は **Phase 3（値オブジェクト）** に進む。

---

# Phase 3 — 値オブジェクト（`Title`）と境界での変換

**到達目標**

- 本の **タイトル**を、ただの `string` ではなく **検証済みの型 `Title`**（値オブジェクト）としてドメインに持ち込む。
- **不正なタイトル**（空文字、長すぎる、前後空白のみ、など。ルールは自分で決めてよい）を `**NewTitle` で弾く**。
- **HTTP や JSON はまだ書かない**。いまは `**RegisterBook` の入口**（ユースケース）で `string` → `Title` に変換し、**ドメインの内側は `Title` だけ**を見る形にする（Phase 5 で HTTP 層に同じ変換を寄せる練習の土台になる）。

**前提:** Phase 2 の修了条件（`go test ./... -race`）を満たしていること。

---

## Step 1 — `Title` 型と検証

ファイル: `internal/domain/book/title.go`（パッケージ `book`）

1. `**type Title struct`** の中身は **非公開**（例: `s string` のように小文字のみ）。外から文字列を勝手に差し替えられないようにする。
2. **コンストラクタ:** `func NewTitle(raw string) (Title, error)`
  - ルール例（**最低 1 つ、できれば 2〜3 つ**）: 前後の `**strings.TrimSpace`** 後が空ならエラー、**最大文字数**（例: 200）を超えたらエラー、など。
3. **観測用:** `func (t Title) String() string` で **表示・ログ用**に中身を取り出す（ドメイン外へ「検証済み文字列」として渡すときに使う）。

ファイル: `internal/domain/book/errors.go`  
**タイトル不正用**の `var Err... = errors.New("book: ...")` を 1 つ以上追加する。

ファイル: `internal/domain/book/title_test.go`  
**テーブル駆動**で、`NewTitle` の成功・失敗パターンを検証する（Phase 2 Step 3 の書き方を再利用）。

**完了条件:** `go test ./internal/domain/book/...` が緑。

---

## Step 2 — `Book` が `Title` を持つようにする

ファイル: `internal/domain/book/book.go`

- フィールドの `**title string` を `title Title` に変更**する（未エクスポートのままでよい）。
- `**NewBook(id string, title Title, author string) *Book`** のように、タイトルは `**Title` だけ**受け取る（`string` を直接受け取らない）。

ファイル: `internal/domain/book/book_test.go`

- 既存テストは `**NewTitle` で合法な `Title` を作ってから `NewBook`** に渡すよう直す。
- `**Title()` メソッド**の戻り値を `string` にするか、`**Title` 型のまま返すか**は設計の選択。まずは `**String()` と揃えて `string` で返す**と、既存のユースケースとの差分が小さくなる。

**完了条件:** `go test ./internal/domain/book/...` が緑。

---

## Step 3 — ユースケースで `string` → `Title` に変換する

ファイル: `internal/usecase/shelf.go`

- `**RegisterBook(ctx, title, author string)` のシグネチャはそのままでよい**（呼び出し元を壊さない）。
- 内部で `**book.NewTitle(title)`** を呼び、エラーなら **そのまま返す**（または `fmt.Errorf` でラップして文脈を足す。方針を決めて一貫させる）。
- 成功した `**Title` を `NewBook` に渡す**。

**学ぶこと:** **「生の入出力（string）」と「ドメインが信頼する値（Title）」の境界**をユースケース（または将来は HTTP アダプタ）に置く。

**完了条件:** `go test ./internal/usecase/...` が緑。

---

## Step 4 — フェイク・メモリ・channel 実装の追随

- `**Save` に渡る `*book.Book`** はもはや `**Title` を内包**している。リポジトリ実装側の **シグネチャは変えず**、コンパイルが通ればよい（コピー保存のパターンはそのまま使える）。
- `**shelf_test` / `goroutine_test` / `fake_book_repository`** など、`**NewBook` や `RegisterBook` を呼ぶ箇所**をすべて Phase 3 に合わせて直す。

**完了条件:** `go test ./... -race` が緑。

---

## Step 5 — `cmd/shelf` と任意の伸ばししろ

- `cmd/shelf/main.go` の `**RegisterBook` 呼び出し**がコンパイルし、実行して期待どおり動くこと。
- **（任意）** `Author` も同様に値オブジェクトにする。ルールはタイトルと別に決めてよい。

---

## Phase 3 修了条件

```bash
go test ./... -race
go run ./cmd/shelf
```

すべて緑・期待どおりなら Phase 3 完了。次は **Phase 4** に進む（開始前に **[EVENTS.md](./EVENTS.md)** を一通り読むことを推奨）。

---

# Phase 4 — ドメインイベント、イベントストア、メッセージ流し

**到達目標**

1. **ドメインイベント**を型で表し、**コマンド成功後**に発行できるようにする。
2. **追記専用のイベントストア**（インメモリ）に載せ、**リプレイ**で `Book` の状態を再構築できるようにする。
3. **メッセージキューの代用**として、**channel + goroutine** の購読者にイベントを流し、**少なくとも 1 回届く**前提の処理を書く（冪等性のコメント付き）。

**前提:** Phase 3 修了（`go test ./... -race` が緑）。  
**併読:** [EVENTS.md](./EVENTS.md)（用語・トレードオフ・本番でのアウトボックス）。

---

## Step 0 — 用語を自分の言葉にする

[EVENTS.md](./EVENTS.md) の §1〜§5 を読み、次を **メモに 1〜2 文ずつ**書く（リポジトリにコミットしなくてよい）。

- コマンドとイベントの違い
- イベント駆動とイベントソーシングの違い
- at-least-once と冪等性

**完了条件:** Phase 4 のコードに着手してよいと自分で判断できること。

---

## Step 1 — イベント型をドメインに定義する

ファイル: `internal/domain/book/shelf_event.go`（名前は任意だが `**event.go` だけだと紛らわしい**ので避けてもよい）

次のような **過去形の事実**を表す struct を **最低 2 種類**（推奨 3 種類）定義する。

- 例: `**BookRegistered`**（本 ID・タイトル文字列または `Title` の表現・著者・発生時刻）
- 例: `**BookBorrowed`**（本 ID・発生時刻）
- 例: `**BookReturned`**（本 ID・発生時刻）

**設計の選択（どちらか一貫）:**

- **A:** `time.Time` をフィールドに持つ（`time` をドメインに import してよいかチーム方針で決める）。
- **B:** 発生時刻は **ユースケースが付与**し、イベントは **事実のペイロードだけ**（Phase 4 では B でもよい）。

**完了条件:** `go build ./internal/domain/book/...` が通る。

---

## Step 2 — イベントの和型（判別しやすい形）

**目的:** `Publish(any)` のように **型が消える**のを避け、テストと `switch` で扱いやすくする。

次のいずれかを採用する（ドキュメントに例を書いた sealed interface でも、`**Kind` 列挙 + `Payload` 用フィールド**でもよい）。

1. **インターフェース + 小文字のマーカーメソッド**（export されないので **パッケージ外では実装できない**）
  例: `type ShelfEvent interface { shelfEvent() }` と各イベントが `func (*BookBorrowed) shelfEvent() {}`。
2. `**type EventKind int` + `struct { Kind EventKind; Registered *BookRegistered; Borrowed *BookBorrowed; ... }`** の **判別共用体風**（1 つだけ非 nil）。

**完了条件:** `go test ./internal/domain/book/...` が緑（イベント型だけのテストを足してもよい）。

---

## Step 3 — ユースケース成功後にイベントを組み立てる

ファイル: `internal/usecase/shelf.go`（および必要なら **新ファイル** `internal/usecase/shelf_events.go`）

- `**RegisterBook` / `BorrowBook` / `ReturnBook`** の **成功パス**の最後（`Save` が成功したあと）で、対応する **ドメインイベント struct** を組み立てる。
- まだ **どこにも送らなくてよい**。`// TODO: publish` でも、次 Step の **フィールド**に渡すでもよい。

**学ぶこと:** **コマンドの結果**として「何が起きたか」を **明示的なデータ**にする。

**完了条件:** 既存の `go test ./internal/usecase/...` が緑。

---

## Step 4 — `EventPublisher` ポートと記録用実装

### この Step のゴール（何ができたら終わりか）

`**ShelfService` が「イベントを外に出すための一本の口」として `EventPublisher` を持ち、成功したコマンドのあとに `Publish` が呼ばれる。テストでは `RecordingPublisher` に溜まった中身を読み取り、`BorrowBook` 成功で `BookBorrowed` が 1 件**あることを確認できる。

---

### 用語整理（struct か func か interface か）


| 名前                         | Go では何か                                                                   | 置き場所の例                                                    | 役割                                                                                                      |
| -------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `**EventPublisher`**       | `**interface` 型**（メソッド集合の契約）                                              | `internal/usecase`（例: `event_publisher.go`）               | ユースケース側の **ポート**。「イベントを渡せばよい」だけを知り、**誰が届けるか（MQ・ログ・メモリ）は知らない**。                                          |
| `**RecordingPublisher`**   | `**struct` + メソッド**（`Publish` を実装）                                        | `internal/adapter/eventlog` または `internal/adapter/memory` | **インメモリの受け皿**。届いた `book.ShelfEvent` を **スライスに `append`**。複数 goroutine から叩くなら `**sync.Mutex**` でスライスを守る。 |
| `**Publish` だけの `func` 型** | `type X func(ctx context.Context, ev book.ShelfEvent) error` のような **関数型** | ポートとしては **非推奨（最初から避けてよい）**                                | クロージャでテストは書きやすいが、**状態（溜めたイベント）＋ Mutex** を同じパターンで表しづらい。学習では **interface + struct** に寄せる。                 |


メソッドシグネチャの例（`ShelfEvent` の名前は **Step 2 で決めた和型**に合わせる）:

```text
Publish(ctx context.Context, ev book.ShelfEvent) error
```

Go には `implements` キーワードがない。**具体型が上記メソッドを持てば**、その型の値は `**EventPublisher` として渡せる**（ダックタイピング）。

---

### やること（順番付きチェックリスト）

1. **ポートをファイルに書く**
  - 例: `internal/usecase/event_publisher.go` に `type EventPublisher interface { Publish(ctx context.Context, ev book.ShelfEvent) error }` を定義する。
2. **記録用アダプタを struct で実装する**
  - パッケージ例: `internal/adapter/eventlog`（名前は任意。`memory` に置いてもよい）。  
  - `**RecordingPublisher` は `struct`** とし、フィールドに `**sync.Mutex`** と `**[]book.ShelfEvent`（または `[]*...`、方針を一つに）** を持つ。  
  - `**Publish` メソッド**: Lock → `append` → Unlock。返り値の `error` はこの段階では `**nil` 固定**でよい。  
  - テストで中身を検証するなら、**スライスをそのまま返さず** `copy` した `**Events() []book.ShelfEvent` など**を用意すると安全（呼び出し側が溜め場を書き換えないため）。
3. `**ShelfService` に配線する**
  - `ShelfService` に `**EventPublisher` 型のフィールド**を足す。  
  - `**NewShelfService(repo, publisher EventPublisher)`** のようにコンストラクタで **必ず受け取る**形が学習では分かりやすい（**推奨**）。  
  - **オプション案:** `publisher` が `nil` のときだけ `Publish` を呼ばない。**デメリット**は「忘れて `nil` が流れる」とイベントが静かに消えること。**必須案**でイベントを捨てたいときは、**何もしない `Publish` を持つ struct**（例: `NoOpEventPublisher`）を明示的に渡す。
4. **Step 3 で組み立てたイベントを、実際に `Publish` する**
  - `RegisterBook` / `BorrowBook` / `ReturnBook` それぞれで、`**repo.Save` が成功した直後**に `s.publisher.Publish(ctx, その操作に対応するイベント)` を呼ぶ。  
  - Step 3 で `// TODO: publish` のままなら、この Step で **TODO を消して**呼び出しに置き換える。
5. `**cmd/shelf` の組み立てを直す**
  - `NewShelfService` の引数が増えるので、`**main` で `NoOp` か `Recording` のどちらを渡すか**を決める（本番でログに出さないなら `NoOp` が無難）。
6. **テストを書く（この Step の受け入れ条件）**
  - `RecordingPublisher` を注入した `ShelfService` で `**RegisterBook` → `BorrowBook` が成功**したあと、記録されたイベント列を走査し、`***book.BookBorrowed` がちょうど 1 件**であることを検証する（型アサーションまたは `switch`）。  
  - フロー全体のテストでは `**NoOp`** を渡して既存の振る舞いを保ってもよい。

**補足（パッケージの向き）:** `adapter` が `usecase` を import すると、`**usecase` のテストから `adapter` を import したときに import サイクル**になりやすい。`RecordingPublisher` 本体は `**usecase` に依存しない**（`book.ShelfEvent` と `context` だけ）に寄せると安全。インターフェース実装の検証がしたければ、`**package eventlog_test` の `*_test.go`** で `var _ usecase.EventPublisher = (*eventlog.RecordingPublisher)(nil)` のように書く手もある。

**完了条件:** `go test ./... -race` が緑。

---

## Step 5 — イベントストア（追記・読出し）

### この Step のゴール（何ができたら終わりか）

**「イベントの列」をストリーム ID ごとにインメモリで保持し、追記（append-only）と読出しができる。**さらに `**Append` に「今のバージョンはこれはず」と渡す楽観ロック**を入れ、**バージョンがずれていたら追記を拒否する**ことで、**同時に二つのコマンドが同じ前提で書き込もうとした**ときの検出の練習ができる。

ユースケース側では、**コマンド 1 回につき**「いまストアに何列あるか把握 → ドメイン操作 → ストアへ新イベントを期待バージョン付きで追記 →（既存なら）`Publish`」の流れを **明示的な手順**として持てる状態にする（概念の背景は [EVENTS.md](./EVENTS.md) §4）。

---

### 用語整理（この Step で固定しておく言葉）


| 用語                       | 意味（この Step での使い方）                                                                                                                                         |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ストリーム ID（`streamID`）** | **1 本のイベント列**の名前。例: 1 冊の本なら `book:` + 本の ID のように **集約インスタンスごとに 1 本**にすると説明しやすい。                                                                           |
| **現在バージョン**              | そのストリームに **すでに何件 append 済みか**を表す整数。**初回 `Load` では「イベント 0 件」に対応する値**をルールで決める（後述）。                                                                          |
| `**Load`**               | 指定 `streamID` の **イベント列のコピー（または読み取り専用のスナップショット）**と、**現在バージョン**を返す。                                                                                        |
| `**Append`**             | 指定 `streamID` に **1 件以上のイベントを末尾に追加**する。`**expectedVersion` がストアの現在バージョンと一致するときだけ成功**させるのがこの Step の楽観ロック。一致しなければ **専用のエラー**（`errors.New` や sentinel）で拒否する。 |
| **楽観ロック**                | 「読んだときのバージョン」を呼び出し側が覚えておき、書くときに **「まだそのバージョンのままなら書いてよい」**と渡す方式。別コマンドが先に append していたらバージョンが進んでおり、**この `Append` は失敗**する。                                    |
| **追記専用（append-only）**    | 既存イベントを **上書き・削除しない**。訂正は別イベントで表す、など本格 ES の話は [EVENTS.md](./EVENTS.md) に任せ、この Step では **スライス末尾に足すだけ**でよい。                                                 |


---

### ポート（interface）— メソッドごとの「何をするか」

置き場所の例: `**internal/usecase`** に `EventStore` などの名前で interface を切る（名前は任意。既存の `Repository` / `EventPublisher` と並べてよい）。


| メソッド（例）                                                                          | 呼び出し側が期待すること                                                     | 実装側がすること（インメモリ）                                                                                                                                                       |
| -------------------------------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Load(ctx, streamID string) ([]book.ShelfEvent, int, error)`                     | **そのストリームの全イベント**（古い順）と、**いまのバージョン番号**が欲しい。                      | `map` からスライスを取り出し、**コピーして返す**と安全。バージョンは **「最後に append したあとの値」**と定義する。ストリームが **未登録**なら **空スライス**と、**初回用のバージョン**（例: `-1` または `0` のどちらか）を **ドキュメントかコメントで一つに決める**。        |
| `Append(ctx, streamID string, expectedVersion int, ev ...book.ShelfEvent) error` | **「バージョンが `expectedVersion` のときだけ」**渡した `ev` を **順に末尾に追加**してほしい。 | ストアの現在バージョンを読み、`expectedVersion` と違えば **すぐエラー**（追記しない）。一致なら **可変長 `ev` を順に append**し、**バージョンを「追加した件数」だけ進める**（例: 2 件まとめて append なら +2 か、常に +1 かは **ルールを決めてテストで固定**する）。 |


`**expectedVersion` のテストでやりたいこと:** 「正しい期待値では append できる」「意図的に **古い期待値**を渡すと **拒否される**」を **テスト関数 2 本以上**で表すと、引数の意味が体に染みる。

---

### インメモリ実装 — 何を `map` に載せるか

**最小構成の例（どちらか一つに決める）:**

1. `**map[string][]book.ShelfEvent` + `map[string]int`（現在バージョン）**
  - キーは `streamID`。  
  - バージョンは **スライス長と常に一致**させるなら `int` の map は省略できるが、**「バージョン ≠ 長さ」**の練習にするなら両方持つ。
2. `**map[string][]book.ShelfEvent` のみ**
  - **現在バージョン = `len(events)-1` または `len(events)`** のように **長さから導出**する。  
  - この場合も `**Load` が返すバージョン**と `**Append` が要求する `expectedVersion`** の対応を **一文で固定**する（オフバイワンでバグりやすい）。

**並行:** 複数 goroutine から叩くなら `**sync.Mutex`**（またはストア専用の **1 本の goroutine** に直列化）で `map` とスライスを守る。`go test ./... -race` を通す前提で決める。

---

### やること（順番付きチェックリスト）

1. **初回 `Load` の契約を決める**
  - 存在しない `streamID` を `**Load` したとき**: `([]T, version, nil)` の `**version` を何にするか**（`-1` / `0` / 「未使用はエラー」）を決め、**テストの期待値と一致**させる。
2. **ポート interface をファイルに書く**
  - `Load` / `Append` のシグネチャを上表に合わせる。`ev` は `...book.ShelfEvent` で **複数件まとめて append 可能**にしても、**常に 1 件**にしてもよい（後者なら可変長引数は省略してもよい）。
3. **インメモリ型を struct で実装する**
  - 例: `internal/adapter/eventstore` や `internal/adapter/memory` 配下。  
  - `**Append` 内:** 現在バージョン取得 → 不一致なら return err → 一致ならスライスへ append → バージョン更新（**Mutex 下で**）。
4. `**ShelfService`（または専用のコマンドハンドラ）にストアを注入する**
  - フィールドに `EventStore` を足し、`NewShelfService` の引数を増やす、など。**既存の `Repository` と二重に状態を持つ**と整合性が難しくなるので、この Step の学習目的を **「ストアを通す」「リポジトリだけ」**のどちらに寄せるかを **先に決める**（両方持つなら **どちらが正**かコメントで書く）。
5. **1 コマンドの中の順序をコードに落とす**
  1. `Load(streamID)` で **イベント列と `ver`** を得る。
  2. **（方針 A）** イベント列を **リプレイして**現在の `*Book` を組み立て、その上で `Borrow` などを実行する。
    **（方針 B）** これまで通り `**FindByID` + `Save`** で状態を持ち、ストアは **「事実のログ」**だけにする。  
    学習では **B の方が変更が少ない**ことが多い。**A にすると Step 6 と重なる**ので、Step 5 では B でよい、と決めてよい。
6. `**Append(..., ver, newEvents)`**
  - `ver` は **手順 5 の `Load` で得た現在バージョン**をそのまま渡す（中で進んだら **再 `Load` してから次のコマンド`**、という二段にしてもよい）。
7. `**Publish`**
  - Step 4 まで通しているなら、**ストアに載せたイベントと同じ内容を `Publish` する**のが自然。`**Append` 成功後**に呼ぶ。  
  - `**Append` は成功したが `Publish` が失敗した**ときにユースケースをどうするか（ロールバックしない／エラーを返す、など）を **一行コメント**で決める（本番はアウトボックスの話につながる）。
8. **テストを書く**
  - **追記検証:** 空のストリームに対し `Append` → `Load` で **件数が増えた**こと。  
  - **楽観ロック検証:** わざと `**expectedVersion` をずらして** `Append` し、**エラーになる**こと。正しい値では成功すること。  
  - 必要なら **複数イベントを一度の `Append`** で足したときの **バージョンの進み方**もテストで固定する。

**完了条件:** 上記の **追記 → 再 `Load` で件数が増える**テストと、**バージョン不一致で `Append` が失敗する**テストが緑。`go test ./... -race` が緑。

---

## Step 6 — リプレイ（イベントから状態を再構築）

ファイル: `internal/domain/book` または `internal/usecase`（どちらに置くか **README かコメントで一行**理由を書く）

- **純関数に近い** `Apply(events []book.ShelfEvent) (*Book, error)` または `**Reduce`**  
  - `BookRegistered` 相当から `**NewBook` 相当の状態**を作る。
  - その後 `BookBorrowed` / `BookReturned` を順に適用して **貸出状態が再現される**ことを確認する。
- **不正な順序**（未登録の `BookBorrowed` など）なら **エラー**にしてもよい。

**テスト:** イベント列を **手で組み立て** → `Apply` → `IsAvailable()` などが期待どおり。

**完了条件:** `go test ./... -race` が緑。

---

## Step 7 — メッセージキュー「代用」（channel + コンシューマ）

**目的:** **publish と処理を別 goroutine** に分け、**配信が遅延・重複しうる**世界に近づける。

1. `**internal/adapter/bus` など**に、`EventPublisher` を実装する型を置く。
2. 内部で `**chan book.ShelfEvent`（バッファあり推奨）** と **コンシューマ goroutine**（`for ev := range ch`）を持つ。
3. `**Publish` は channel に送るだけ**で早く返す（**非同期**）。バッファが満杯なら **ブロック**する点に注意。
4. コンシューマ側で `**Handler func(context.Context, book.ShelfEvent)`** を 1 件ずつ呼ぶ（**冪等**: 同じイベントが 2 回来ても壊れないように **コメントまたは簡単な重複検知**を書く）。

**テスト:** `Publish` を複数回 → コンシューマが **受け取った件数**を `sync.WaitGroup` で待って検証。`-race` 付き。

**完了条件:** `go test ./... -race` が緑。

---

## Step 8 — （任意）本番 MQ・アウトボックス

- **RabbitMQ / NATS / Redis Streams / SQS** などは **このリポジトリでは必須にしない**。ローカルで試す手順を **自分用メモ**や **別 README** に書くだけで Phase 4 としては十分。
- **Transactional Outbox** は [EVENTS.md](./EVENTS.md) §5.5。DB 実装が入った **Phase 6** のあとに読み直すと実感がつきやすい。

---

## Phase 4 学びの記録（実装・ドキュメント・運用）

この節は、**Phase 4 Step 1〜4 を進める過程で出た論点**を、手順書（上の Step）とは別角度でまとめたものです。将来の自分やレビュー相手が「なぜそう書いたか」を追いやすくするための **メタ情報**です。[EVENTS.md](./EVENTS.md) が概念、[IMPLEMENTATION.md](./IMPLEMENTATION.md) §6.5 がコード粒度の補足と読み分けるとよいです。

### 1. ドキュメント側でやったこと（Step 4 の書き直し）

**課題:** 「ポートを定義」「インメモリ実装」などの箇条書きだけだと、**何をどのファイルに書き、どうつなぐか**が読み手の頭の中で補完されてしまう。

**対応:** [Step 4](#step-4--eventpublisher-ポートと記録用実装) を次の順に再構成した。

1. **ゴール** … 終了時に何ができていればよいかを一文〜数文で固定する。
2. **用語表** … `EventPublisher`（interface）、`RecordingPublisher`（struct）、「`func` 型だけのポート」を並べ、**Go のどの言語機能に相当するか**を明示する。
3. **順番付きチェックリスト** … ポート定義 → アダプタ → `ShelfService` 配線 → `Publish` 呼び出し → `main` → テスト、の **依存順**で番号を振った。
4. **import サイクル** … `adapter` が `usecase` を import すると、`usecase` のテストから `adapter` を引きたいときに詰まりやすい、と一文で理由を書いた。
5. **テストの受け入れ** … `BorrowBook` 成功後に `*book.BookBorrowed` が **ちょうど 1 件**、と観測対象を具体化した。

**学び:** 学習用ドキュメントは「正しさ」だけでなく **作業分解（WBS）と観測可能な完了条件**があると、実装とドキュメントのズレに気づきやすい。

### 2. 設計判断の整理（ポート・アダプタ・注入）


| 論点                         | 推奨（このリポジトリの学習方針）                                   | 理由の一言                                                |
| -------------------------- | -------------------------------------------------- | ---------------------------------------------------- |
| ポートの置き場所                   | `internal/usecase` の `EventPublisher`              | ユースケースが「外へ事実を出す」契約を持つ。ドメインはイベント **型**まで。             |
| 記録用実装の形                    | `RecordingPublisher` は **struct + `Publish` メソッド** | スライスと `Mutex` をフィールドに持てる。テストで「溜まった列」を読む。             |
| `func` だけのポート              | 最初は避けてよい                                           | 状態付きのテストダブルが書きにくく、**interface + 具体型**の練習目的に合わない。     |
| `publisher` を `nil` 許容にするか | 学習では **必須フィールド + `NoOpEventPublisher`** を推奨        | `nil` 分岐は「イベントが静かに消える」バグの温床。捨てるなら **明示的な NoOp** がよい。 |
| `Publish` の第一引数            | `context.Context` を付ける                             | リポジトリと同様、あとからタイムアウト・キャンセル伝播を足しやすい。                   |


### 3. 実装でつまずきやすい点（チェックリスト）

Step 4 をコードに落とすとき、次を **上から順に**確認すると戻りやすい。

1. `**EventPublisher` のメソッドシグネチャと、`RecordingPublisher.Publish` が一致しているか**（`ctx` の有無・順序まで含めて **一字一句**同型になる必要がある）。
2. **Step 3 で組み立てた `event` 変数を、Step 4 で本当に `Publish` に渡しているか**（組み立てただけで **未使用**のままだと、コンパイルは通ってもイベントは出ない）。
3. `**Save` の成功後**にだけ `Publish` するか（失敗パスで事実が流れないか）。
4. `**fmt.Errorf(...)` を `return` せずに捨てていないか**（コンパイラは怒らないが意味がない）。
5. `**NewShelfService` の呼び出し元**（`shelf_test`, `goroutine_test`, `cmd/shelf`）を、引数が増えたあと **すべて更新したか**。
6. **テストの二段構え** … 既存の **登録→借りる→…のフロー**は `NoOp` で軽く保ち、**イベント件数の検証**は `RecordingPublisher` を注入した **別テスト**に分けると読みやすい（[TESTING.md](./TESTING.md) の考え方と一致）。

### 4. テストと `-race`

- **受け入れ:** `go test ./... -race` が緑であること（Phase 4 修了条件と同じ）。  
- **RecordingPublisher** は共有スライスに append するため、`**Mutex` なし**だと `-race` で検知されうる。意図して Mutex を使う練習にもなる。

**import サイクル（本チャットで実際に出たパターン）:**  
`usecase` のテストが `internal/adapter/eventlog` を import する一方、`eventlog` パッケージ本体が `**var _ usecase.EventPublisher = (*RecordingPublisher)(nil)`** のために `usecase` を import すると、

```text
import cycle not allowed in test
```

になる。**対策の例:** `RecordingPublisher` のあるパッケージは `**book` と `context` だけ**に依存させ、`usecase` は import しない。インターフェースを満たすことの検証は `**package eventlog_test`** のファイルで `var _ usecase.EventPublisher = ...` と書く（[Step 4](#step-4--eventpublisher-ポートと記録用実装) 補足と同じ）。

**受け入れ条件の再確認（要件メモ）:** インメモリの `**RecordingPublisher`**（`Mutex` ＋ スライス `append`）、`ShelfService` への `**EventPublisher` 注入**（学習では **必須 + `NoOp`** が分かりやすい）、`**BorrowBook` 成功後に `BookBorrowed` が記録上 1 件**、`go test ./... -race` が緑。

### 7. 読み返し用リンク


| 内容              | ドキュメント                                              |
| --------------- | --------------------------------------------------- |
| Step 4 の手順（改訂後） | このファイルの [Step 4](#step-4--eventpublisher-ポートと記録用実装) |
| 用語・ES・MQ        | [EVENTS.md](./EVENTS.md)                            |
| ファイル粒度・よくあるバグ   | [IMPLEMENTATION.md](./IMPLEMENTATION.md) §6.5       |
| テスト方針・`-race`   | [TESTING.md](./TESTING.md)                          |


---

## Phase 4 修了条件

```bash
go test ./... -race
```

次は **Phase 5（HTTP）**（DTO と値オブジェクトの境界）へ。イベントを **Webhooks で外向きに出す**などは Phase 5 以降の伸ばししろとする。

---

## Phase 5 以降（概要）

**Phase 5:** HTTP 層を `internal/adapter/http` に追加し、**DTO（JSON）の `string` → `NewTitle` など**をアダプタで行い、ユースケースにはすでに検証済みの型を渡す。必要なら **重要なドメインイベントを Webhook / 外向きキューに publish** する設計を検討する。

**Phase 6–7:** DB 永続化、境界づけられたコンテキストでのパッケージ分割（フェーズ一覧表を参照）。**イベントストアと DB を同一トランザクションに載せる**などはここで現実味が出る。

---

## よく使うコマンド

```bash
go test ./...
go test ./... -race
go test ./... -cover
go run ./cmd/shelf
```


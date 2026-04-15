# go_practice

Go と **ドメイン駆動設計（DDD）** を段階的に学ぶリポジトリです。

## 重要: 学び方

[TRAINING.md](docs/TRAINING.md) は **自分の手で 1 から追う前提**の手順です。  
すでにコードがある場合も、設計・テスト・実装の整理は [DESIGN.md](docs/DESIGN.md) などを参照してください。

**知見の置き場所（読み分け）:** 用語の定義・手順は TRAINING にありますが、「なぜそうするか」「他のやり方との比較」「ハマりどころの深掘り」は **トピックごと**に次のドキュメントへ統合してあります。

| 知りたいこと | 主に読むファイル |
|--------------|------------------|
| レイヤー責務、依存の向き、Mutex 版と channel 版の対比、ポートをドメインに置くかユースケースに置くか | [DESIGN.md](docs/DESIGN.md) |
| どのパッケージで何をテストするか、並行テスト、`WaitGroup`、`t.Parallel` の違い、`-race`、子 goroutine と `t.Fatal` | [TESTING.md](docs/TESTING.md) |
| ファイル順、`package main`、`:=` と `=`、メモリ／channel リポジトリの具体的な書き分け | [IMPLEMENTATION.md](docs/IMPLEMENTATION.md) |
| ドメインイベント・イベントソーシング・MQ・アウトボックスの用語 | [EVENTS.md](docs/EVENTS.md) |
| Phase 4 での **学びの整理**（Step 4 改訂の意図、ポート／アダプタ、Git 運用、**§6** 依頼の言葉と技術の対応・import サイクル具体例） | [TRAINING.md](docs/TRAINING.md) の **「Phase 4 学びの記録」**（修了条件の直前） |

手を動かしながら詰まったら、**該当 Step を TRAINING で確認したうえで**、上記の対応セクションを読むとつながりやすいです。

## 最初に読むもの

**[docs/TRAINING.md](docs/TRAINING.md)** — Phase 1 を **Step 0 から順に** 実行してください。

## より具体的なガイド（設計・テスト・実装を分離）

| 内容 | ドキュメント |
|------|----------------|
| レイヤー責務・依存の向き・処理の流れ | [docs/DESIGN.md](docs/DESIGN.md) |
| レイヤー別に何をテストするか・具体例・落とし穴 | [docs/TESTING.md](docs/TESTING.md) |
| ファイル順・各ファイルの書き方・よくあるバグ | [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) |
| Phase 4: イベント・イベントストア・メッセージ流しの概念 | [docs/EVENTS.md](docs/EVENTS.md) |
| Phase 4 の手順補足・学びの記録（ポート／アダプタ・Git 運用・**対話で確認した論点は §6**） | [docs/TRAINING.md](docs/TRAINING.md)（**Phase 4 学びの記録** — 修了条件の直前） |

## 事前準備

```bash
# モジュールは既に go.mod にある。別パスにしたい場合だけ go mod init し直す。
go test ./...   # 最初はパッケージが無くて失敗してよい
```

Phase 1 を終えると `go test ./...` と `go run ./cmd/shelf` が通る状態になります。

## 質問するとき

- **どの Step か**（例: Phase 1 Step 4）
- **`go test` または `go build` の全文**

を書いてもらえると答えやすいです。

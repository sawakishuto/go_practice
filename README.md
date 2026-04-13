# go_practice

Go と **ドメイン駆動設計（DDD）** を段階的に学ぶリポジトリです。

## 重要: 学び方

[TRAINING.md](docs/TRAINING.md) は **自分の手で 1 から追う前提**の手順です。  
すでにコードがある場合も、設計・テスト・実装の整理は [DESIGN.md](docs/DESIGN.md) などを参照してください。

## 最初に読むもの

**[docs/TRAINING.md](docs/TRAINING.md)** — Phase 1 を **Step 0 から順に** 実行してください。

## より具体的なガイド（設計・テスト・実装を分離）

| 内容 | ドキュメント |
|------|----------------|
| レイヤー責務・依存の向き・処理の流れ | [docs/DESIGN.md](docs/DESIGN.md) |
| レイヤー別に何をテストするか・具体例・落とし穴 | [docs/TESTING.md](docs/TESTING.md) |
| ファイル順・各ファイルの書き方・よくあるバグ | [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) |

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

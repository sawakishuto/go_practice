# go_practice

Go と **ドメイン駆動設計（DDD）** を段階的に学ぶリポジトリです。

## 重要: コードは自分で書く

このリポジトリには **実装済みの Go コードは含めていません**（`go.mod` と手順書のみ）。  
**1 からファイルを追加**してアプリを作ります。

## 最初に読むもの

**[docs/TRAINING.md](docs/TRAINING.md)** — Phase 1 を **Step 0 から順に** 実行してください。

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

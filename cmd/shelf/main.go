package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sawakishuto/go_practice/internal/adapter/memory"
	"github.com/sawakishuto/go_practice/internal/usecase"
)

func main() {
	repo := memory.NewBookRepository()
	shelf := usecase.NewShelfService(repo)

	// 本を登録する
	id, err := shelf.RegisterBook(context.Background(), "今日の本", "sawaki shuto")
	if err != nil {
		log.Fatalf("Failed to register book: %v", err)
	}
	fmt.Println("今日登録した本は", id, "です")

	// 本を借りる
	err = shelf.BorrowBook(context.Background(), id)
	if err != nil {
		log.Fatalf("Failed to borrow book: %v", err)
	}
	fmt.Println("本を借りました", id)

	// 本を返す
	err = shelf.ReturnBook(context.Background(), id)
	if err != nil {
		log.Fatalf("Failed to return book: %v", err)
	}
	fmt.Println("本を返しました", id)
}

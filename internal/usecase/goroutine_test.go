package usecase

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestMultiAccessFromUser(t *testing.T) {
	repo := NewFakeBookRepository()
	shelf := NewShelfService(repo)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, err := shelf.RegisterBook(context.Background(), "山の本", "sawashu")
			if err != nil {
				t.Fatalf("can't regist this book")
			}
			fmt.Println("登録した本のidは", id)
		}(i)
	}
	wg.Wait()
	num := len(repo.books)
	fmt.Println("登録した本の数は", num)

}

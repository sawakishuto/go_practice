package book
import "errors"

var (
	// AlreadyBorrowed はすでに貸出中の本に Borrow したときに返す。
	AlreadyBorrowed = errors.New("book: already borrowed")
	// NotBorrowed は貸出されていない本に Return したときに返す。
	NotBorrowed = errors.New("book: not borrowed")
	// BookNotFound はリポジトリが本を見つけられないときに返す（Step 3 以降で使用）。
	BookNotFound = errors.New("book: not found")
)

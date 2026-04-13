package book

// Book は蔵書（エンティティ）。
type Book struct {
	id         string
	title      string
	author     string
	isBorrowed bool
}

// NewBook は貸出可能な本を返す。
func NewBook(id, title, author string) *Book {
	return &Book{
		id: id, title: title, author: author, isBorrowed: false,
	}
}

func (b *Book) ID() string     { return b.id }
func (b *Book) Title() string  { return b.title }
func (b *Book) Author() string { return b.author }

// IsAvailable は貸出可能かどうか。
func (b *Book) IsAvailable() bool { return !b.isBorrowed }

// Borrow は本を貸し出す。
func (b *Book) Borrow() error {
	if b.isBorrowed {
		return AlreadyBorrowed
	}
	b.isBorrowed = true
	return nil
}

// Return は本を返却する。
func (b *Book) Return() error {
	if !b.isBorrowed {
		return NotBorrowed
	}
	b.isBorrowed = false
	return nil
}

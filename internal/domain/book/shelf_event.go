package book

import "time"

type BookRegistered struct {
	ID         string
	Title      Title
	Author     string
	OccurredAt time.Time
}

type BookBorrowed struct {
	ID         string
	OccurredAt time.Time
}

type ShelfEvent interface {
	shelfEvent()
}

func (*BookBorrowed) shelfEvent() {

}

func (*BookRegistered) shelfEvent() {

}

package book

import "time"

type BookRegistered struct {
	id string
	title Title
	author string
	occurredAt time.Time
}

type BookBorrowed struct {
	id string
	occurredAt time.Time

}

type ShelfEvent interface {
	shelfEvent()
}

func (*BookBorrowed) shelfEvent() {

}

func (*BookRegistered) shelfEvent() {

}

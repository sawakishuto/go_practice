package book

import "strings"

type Title struct {
	s string
}

func NewTitle(raw string) (Title, error) {

	r := raw

	if len(r) >= 200 {
		return Title{}, ErrBookTitleTooLong
	}
	if strings.Contains(r, "fuck") {
		return Title{}, BadTitle
	}
	return Title{s: r}, nil
}

func (t *Title) Title() (s string) {
	return t.s
}

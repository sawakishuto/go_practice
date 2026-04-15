package book

import (
	"fmt"
	"testing"
)

func TestTitle(t *testing.T) {
	var goodRawTitle = "goodTitle"


	gtitle, err := NewTitle(goodRawTitle)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(gtitle)


}

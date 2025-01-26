package template

import (
	"time"
)

type Template struct {
	Id         int
	Name       string
	CreatedAt  time.Time
	Landscape  bool
	Parameters []Parameter
	Images     []Image
	Texts      []Text
}

type Parameter struct {
	Id        int
	Name      string
	MaxLength int
}

type Image struct {
	Id            int
	Image         []byte
	X, Y          int
	Width, Height int
}

type Text struct {
	Id            int
	Text          string
	X, Y          int
	Width, Height int
}

package template

import (
  "time"
)

type Template struct {
  id int
  name string
  createdAt time.Time
  landscape bool
  parameters []Parameter
  images []Image
  texts []Text
}

type Parameter struct {
  id int
  name string
  maxLength int
}

type Image struct {
  id int
  image []byte
  x, y int
  width, height int
}

type Text struct {
  id int
  text string
  x, y int
  width, height int
}

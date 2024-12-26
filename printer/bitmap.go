package printer

import (
  "fmt"
  "image"
)

type Bitmap interface {
  Width() int
  Height() int
  GetBit(x int, y int) byte
}

type PixelBitmap struct {
  pixels [][]byte
  width, height int
}

func (b *PixelBitmap) Width() int {
  return b.width
}

func (b *PixelBitmap) Height() int {
  return b.height
}

func (b *PixelBitmap) GetBit(x int, y int) byte {
  return b.pixels[y][x]
}

func (b *PixelBitmap) String() string {
  return fmt.Sprintf("PixelBitmap(%d,%d)", b.width, b.height)
}

func BitmapFromPaletted(i *image.Paletted, colorMap []byte) (*PixelBitmap, error) {
  if len(colorMap) != len(i.Palette) {
    return nil, fmt.Errorf("colorMap should be same length as palette")
  }

  width, height := i.Bounds().Dx(), i.Bounds().Dy()
  pixels := make([][]byte, height)
  for y := range height {
    row := make([]byte, width)
    for x := range width {
      row[x] = colorMap[i.ColorIndexAt(x, y)]
    }

    pixels[y] = row
  }

  return &PixelBitmap{pixels, width, height}, nil
}

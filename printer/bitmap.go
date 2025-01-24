package printer

import (
	"fmt"
	"image"
	"image/color"
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

type ImageBitmap struct {
  image *image.Paletted
  colorMap [2]byte
}

func FromPaletted(i *image.Paletted) (*ImageBitmap, error) {
  if len(i.Palette) != 2 { 
    return nil, fmt.Errorf("Image passed to FromPaletted must have only 2 colours in palette")
  }

  var colorMap [2]byte

  // Determine which of the two colours in the image's palette is closest to white
  // colorMap[i] represents the bit value of the palette colour at index i
  if i.Palette.Index(color.White) == 0 {
    colorMap = [2]byte{0,1}
  } else {
    colorMap = [2]byte{1,0}
  }

  return &ImageBitmap{
    image: i,
    colorMap: colorMap,
  }, nil
}

func (b *ImageBitmap) Width() int {
  return b.image.Rect.Dx()
}

func (b *ImageBitmap) Height() int {
  return b.image.Rect.Dy()
}

func (b *ImageBitmap) GetBit(x int, y int) byte {
  return b.colorMap[b.image.ColorIndexAt(x, y)]
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

// colorMap: for each index i of a colour in i.Palette, colorMap[i] is the
// bit in the PackedBitmap which that colour maps to-so if the first colour
// in the image palette is white, then white in the image will be mapped to
// the bit value of colorMap[0]
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

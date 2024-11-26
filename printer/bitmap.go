package printer

import "fmt"
import "gotenburg/model"

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

func BitmapFromRequest(r *model.PrintingRequest) (*PixelBitmap, error) {
  pixels := make([][]byte, r.Height)
  if len(r.Data) != r.Width * r.Height {
    return nil, fmt.Errorf("Bitmap pixels not consistent with provided width and height (got %v, expecting %v*%v=%v",
      len(r.Data),
      r.Width,
      r.Height,
      r.Width * r.Height,
    )
  }

  for y := range r.Height {
    pixels[y] = r.Data[y * r.Width:(y + 1) * r.Width]
  }

  return &PixelBitmap{
    pixels: pixels,
    width: r.Width,
    height: r.Height,
  }, nil
}


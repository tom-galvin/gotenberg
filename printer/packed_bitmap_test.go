package printer

import (
  "fmt"
  "testing"
  "math/rand/v2"
)

func aRandomBitmap() *PixelBitmap {
  width, height := 1 + rand.IntN(400), 1 + rand.IntN(400)
  pixels := make([][]byte, height)
  for y := range height {
    row := make([]byte, width)
    for x := range width {
      row[x] = byte(rand.IntN(2))
    }
    pixels[y] = row
  }

  return &PixelBitmap{pixels, width, height}
}

func assertBitmapsIdentical(t *testing.T, b1 Bitmap, b2 Bitmap) {
  if b1.Width() != b2.Width() {
    t.Errorf("Bitmaps not of equal width: %s %s", b1, b2)
  }
  if b1.Height() != b2.Height() {
    t.Errorf("Bitmaps not of equal height: %s %s", b1, b2)
  }
  width, height := b1.Width(), b1.Height()

  for y := range height {
    for x := range width {
      bit1, bit2 := b1.GetBit(x, y), b2.GetBit(x, y)
      if bit1 != bit2 {
        t.Errorf("Bit at (%v, %v) doesn't match: %v vs %v", x, y, bit1, bit2)
      }
    }
  }
}

func TestPackBitmap(t *testing.T) {
  test := &PixelBitmap {
    pixels: [][]byte {
      {1, 0},
      {0, 1},
    },
    width: 2, height: 2,
  }

  copied := PackBitmap(test)
  assertBitmapsIdentical(t, test, copied)
}

func TestPackBitmapMany(t *testing.T) {
  const testCaseCount = 30

  for i := range testCaseCount {
    testBitmap := aRandomBitmap()
    t.Run(fmt.Sprintf("test %v: %s", i, testBitmap.String()), func (t *testing.T) {
      copiedBitmap := PackBitmap(testBitmap)
      assertBitmapsIdentical(t, testBitmap, copiedBitmap)
      copiedAgainBitmap := PackBitmap(copiedBitmap)
      assertBitmapsIdentical(t, copiedBitmap, copiedAgainBitmap)
    })
  }
}

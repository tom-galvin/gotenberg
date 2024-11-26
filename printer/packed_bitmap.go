package printer

import "fmt"

// a bitmap packed in memory
type PackedBitmap struct {
  data []byte
  width, height, stride int
}
const bitsPerWord = 8

func (b *PackedBitmap) Width() int {
  return b.width
}

func (b *PackedBitmap) Height() int {
  return b.height
}

func (b *PackedBitmap) Stride() int {
  return b.stride
}

func (b *PackedBitmap) Data() []byte {
  return b.data
}

func (b *PackedBitmap) GetBit(x int, y int) byte {
  bitIndex := x % bitsPerWord
  wordStartX := x - bitIndex
  bitsInThisWord := b.width - wordStartX
  if bitsInThisWord > 8 {
    bitsInThisWord = 8
  }

  index := (y * b.stride) + (x / bitsPerWord)
  return (b.data[index] >> (bitsInThisWord - 1 - bitIndex)) & 1
}

func (b *PackedBitmap) String() string {
  return fmt.Sprintf("PackedBitmap(%d,%d)", b.width, b.height)
}

func (b *PackedBitmap) Chunk(start int, height int) *PackedBitmap {
  return &PackedBitmap{
    data: b.data[b.stride * (start):b.stride*(start + height)],
    width: b.width,
    height: b.height,
    stride: b.stride,
  }
}

func PackBitmap(b Bitmap) *PackedBitmap {
  width, height, stride := b.Width(), b.Height(), (b.Width() + bitsPerWord - 1) / bitsPerWord
  data := make([]byte, stride * height)

  var p byte = 0
  for y := range height {
    for x := range width {
      p = (p << 1) | (b.GetBit(x, y) & 1)

      if x == width - 1 || x % bitsPerWord == bitsPerWord - 1 {
        index := y * stride + (x / bitsPerWord)
        data[index] = p
        p = 0
      }
    }
  }

  return &PackedBitmap{data, width, height, stride}
}

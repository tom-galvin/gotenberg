// This package defines an interface for a simple bitmap structure that has a
// width, height, and can get bits from the bitmap by (x,y) coordinate.
// It also defines a simple implementation PixelBitmap that stores each pixel
// in a byte in a 2D array format, which is use to test the PackedBitmap impl.
// Lastly it defines the PackedBitmap structure which is the format which the
// phomemo device consumes over the wire.
package bitmap

import (
	"fmt"
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

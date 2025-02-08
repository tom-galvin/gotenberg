package bitmap

import (
	"fmt"
	"image"
	"image/color"
	"golang.org/x/image/draw"
	"math"
	"github.com/makeworld-the-better-one/dither/v2"
)

type ImageBitmap struct {
	image *image.Paletted
	// colorMap[i] represents the bit value of the palette colour at index i.
	// If the first colour in the image is black, and a high bit in a bitmap
	// sent to the device will be printed as black, then colorMap[0] == 1.
	colorMap [2]byte
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

func FromPaletted(i *image.Paletted) (*ImageBitmap, error) {
	if len(i.Palette) != 2 { 
		return nil, fmt.Errorf("Image passed to FromPaletted must have only 2 colours in palette")
	}

	var colorMap [2]byte

	// Determine which of the two colours in the image's palette is closest to white.
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

// take an image, monochrome-ify using dithering, so it can be used for an ImageBitmap
func RenderForDevice(i image.Image) *image.Paletted {
	// TODO: if image less than printer max width, pad with white pixels
	// Phomemo T02 hardware seems to act unpredictably if input bitmap less than device width

	// determine width of bitmap to print, ready to scale
	maxWidth := 48 * 8
	newWidth := i.Bounds().Dx()
	if newWidth > maxWidth {
		newWidth = maxWidth
	}
	scaledBounds := image.Rect(0, 0, newWidth, i.Bounds().Dy() * newWidth / i.Bounds().Dx())
	scaledImage := image.NewRGBA(scaledBounds)
	// resize image using Catmull Rom scaling
	draw.CatmullRom.Scale(scaledImage, scaledBounds, i, i.Bounds(), draw.Over, nil)

	// turn full colour image into monochrome pixel by pixel
	monochromeImage := image.NewGray16(scaledBounds)
	for y := scaledBounds.Min.Y; y < scaledBounds.Max.Y; y++ {
		for x := scaledBounds.Min.X; x < scaledBounds.Max.X; x++ {
			originalColor := scaledImage.At(x, y)
			grayColor := color.Gray16Model.Convert(originalColor).(color.Gray16)
			grayValue := float64(grayColor.Y) / float64(0xFFFF)

			// apply a gamma correction of 0.5 otherwise image appears too dark with T02
			// no logic used to pick 0.5 as gamma factor, just looks empirically close to image on display
			scaledGrayValue := math.Pow(grayValue, 0.5)
			scaledGrayColor := color.Gray16{Y:uint16(scaledGrayValue * float64(0xFFFF))}
			monochromeImage.Set(x, y, scaledGrayColor)
		}
	}
	
	// dither monochrome image to black and white
	palette := []color.Color{color.Black, color.White}
	ditherer :=  dither.NewDitherer(palette)
	ditherer.Matrix = dither.FloydSteinberg
	ditherer.Serpentine = true
	ditheredImage := ditherer.DitherPaletted(monochromeImage)

	return ditheredImage
}

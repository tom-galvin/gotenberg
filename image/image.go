package phomemo

import (
  "image"
  "image/color"
  "golang.org/x/image/draw"
  "math"
  "github.com/makeworld-the-better-one/dither/v2"
)

// take an image, monochrome-ify using dithering, pack it into the Phomemo bitmap format to print
// image palette will be {black, white} in that order
func RenderForDevice(i image.Image) image.Image {
  // TODO: if image less than printer max width, pad with white pixels
  // Phomemo T02 hardware seems to act unpredictably if input bitmap less than device width

  // determine width of bitmap to print, ready to scale
  maxWidth := 48 * 8
  newWidth := i.Bounds().Dx()
  if newWidth > maxWidth {
    newWidth = maxWidth
  }
  b := image.Rect(0, 0, newWidth, i.Bounds().Dy() * newWidth / i.Bounds().Dx())
  scaled := image.NewRGBA(b)
  // resize image using Catmull Rom scaling
  draw.CatmullRom.Scale(scaled, b, i, i.Bounds(), draw.Over, nil)

  // turn full colour image into monochrome pixel by pixel
  monochromed := image.NewGray16(b)
  for y := b.Min.Y; y < b.Max.Y; y++ {
    for x := b.Min.X; x < b.Max.X; x++ {
      originalColor := scaled.At(x, y)
      grayColor := color.Gray16Model.Convert(originalColor).(color.Gray16)
      grayValue := float64(grayColor.Y) / float64(0xFFFF)

      // apply a gamma correction of 0.5 otherwise image appears too dark with T02
      // no logic used to pick 0.5 as gamma factor, just looks empirically close to image on display
      scaledGrayValue := math.Pow(grayValue, 0.5)
      scaledGrayColor := color.Gray16{Y:uint16(scaledGrayValue * float64(0xFFFF))}
      monochromed.Set(x, y, scaledGrayColor)
    }
  }
  
  // dither monochrome image to black and white
  palette := []color.Color{color.Black, color.White}
  ditherer :=  dither.NewDitherer(palette)
  ditherer.Matrix = dither.FloydSteinberg
  ditherer.Serpentine = true
  dithered := ditherer.DitherPaletted(monochromed)

  return dithered
}

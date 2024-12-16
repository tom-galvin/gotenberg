package phomemo

import (
  "image"
  "image/color"
  "golang.org/x/image/draw"
  "gotenburg/printer"
  "math"
  "github.com/makeworld-the-better-one/dither/v2"
)

func packImageToPhomemoBitmap(i image.Image) (*printer.PackedBitmap, error) {
  maxWidth := 48 * 8
  newWidth := i.Bounds().Dx()
  if newWidth > maxWidth {
    newWidth = maxWidth
  }
  b := image.Rect(0, 0, newWidth, i.Bounds().Dy() * newWidth / i.Bounds().Dx())
  scaled := image.NewRGBA(b)
  draw.CatmullRom.Scale(scaled, b, i, i.Bounds(), draw.Over, nil)

  monochromed := image.NewGray16(b)
  for y := b.Min.Y; y < b.Max.Y; y++ {
    for x := b.Min.X; x < b.Max.X; x++ {
      originalColor := scaled.At(x, y)
      grayColor := color.Gray16Model.Convert(originalColor).(color.Gray16)
      grayValue := float64(grayColor.Y) / float64(0xFFFF)
      scaledGrayValue := math.Pow(grayValue, 0.5)
      scaledGrayColor := color.Gray16{Y:uint16(scaledGrayValue * float64(0xFFFF))}
      monochromed.Set(x, y, scaledGrayColor)
    }
  }
  
  palette := []color.Color{color.Black, color.White}
  ditherer :=  dither.NewDitherer(palette)
  ditherer.Matrix = dither.FloydSteinberg
  ditherer.Serpentine = true
  dithered := ditherer.DitherPaletted(monochromed)

  bitmap, err := printer.BitmapFromPaletted(dithered, []byte{1,0})

  if err != nil {
    return nil, err
  }

  return printer.PackBitmap(bitmap), nil
}

package phomemo

import (
  "image"
  "image/color"
  "golang.org/x/image/draw"
  "gotenburg/printer"
  "github.com/makeworld-the-better-one/dither/v2"
)

func packImageToPhomemoBitmap(i image.Image) (*printer.PackedBitmap, error) {
  newWidth := i.Bounds().Dx()
  if newWidth > 48 {
    newWidth = 48
  }
  scaled := image.NewRGBA(image.Rect(0, 0, i.Bounds().Dx() * newWidth / i.Bounds().Dy(), newWidth))
  draw.CatmullRom.Scale(scaled, scaled.Bounds(), i, i.Bounds(), draw.Over, nil)
  
  palette := []color.Color{color.Black, color.White}
  ditherer :=  dither.NewDitherer(palette)
  ditherer.Matrix = dither.FloydSteinberg
  dithered := ditherer.DitherPaletted(scaled)
  bitmap, err := printer.BitmapFromPaletted(dithered)

  if err != nil {
    return nil, err
  }

  return printer.PackBitmap(bitmap), nil
}

package template

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Measure struct {
	X, Y          int
	Width, Height int
	OutOfBounds   bool
}

func wrapText(text string, maxWidth int, face font.Face) []string {
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return lines
	}

	var line string
	for _, word := range words {
		testLine := line
		if len(line) > 0 {
			testLine += " "
		}
		testLine += word

		// Measure text width
		width := font.MeasureString(face, testLine).Ceil()
		if width > maxWidth && len(line) > 0 && maxWidth > 0 {
			lines = append(lines, line)
			line = word
		} else {
			line = testLine
		}
	}

	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func measureAndDrawChildText(text *Text, i *image.RGBA64) Measure {
	wrappedText := wrapText(text.FilledText, text.Width, text.FontFace)
	var width, height int
	for _, line := range wrappedText {
		lineWidth := font.MeasureString(text.FontFace, line).Ceil()
		if lineWidth > width {
			width = lineWidth
		}
		height += text.FontFace.Metrics().Height.Ceil()
	}

	m := Measure{
		X:      text.X,
		Y:      text.Y,
		Width:  width,
		Height: height,
	}

	if height > text.Height && text.Height > 0 {
		m.OutOfBounds = true
		return m
	}

  if i != nil {
    d := &font.Drawer{
      Dst:  i,
      Src:  image.NewUniform(color.Black),
      Face: text.FontFace,
    }
    d.Dot = fixed.Point26_6{X: fixed.I(m.X), Y: fixed.I(m.Y)}
    for _, line := range wrappedText {
      d.Dot.X = fixed.I(m.X)
      d.Dot.Y += text.FontFace.Metrics().Ascent
      fmt.Println(line)
      fmt.Println(d.Dot)
      d.DrawString(line)
      d.Dot.Y += text.FontFace.Metrics().Descent
    }
  }
	return m
}

func measureAndDrawChildImage(img *Image, i *image.RGBA64) Measure {
	m := Measure{
		X:      img.X,
		Y:      img.Y,
		Width:  img.Width,
		Height: img.Height,
	}
  if i != nil {
    bounds := image.Rect(m.X, m.Y, m.X+m.Width, m.Y+m.Height)
    draw.CatmullRom.Scale(i, bounds, img.LoadedImage, img.LoadedImage.Bounds(), draw.Over, nil)
  }
  return m
}

// Measure the elements to be drawn for the template and determine the boundaries
// of the image. If the elements exceed the template bounds then return an error
func measureAndCheckBounds(t *Template) (int, int, error) {
  var width, height int
  if t.Landscape {
    width = t.MinSize
    height = deviceWidth
  } else {
    width = deviceWidth
    height = t.MinSize
  }
  for _, img := range t.Images {
    bounds := measureAndDrawChildImage(&img, nil)
    if bounds.X + bounds.Width > width {
      width = bounds.X + bounds.Width
    }
    if bounds.Y + bounds.Height > height {
      height = bounds.Y + bounds.Height
    }
  }
  for _, txt := range t.Texts {
    bounds := measureAndDrawChildText(&txt, nil)
    if bounds.OutOfBounds {
      return 0, 0, fmt.Errorf("Text out of bounds")
    }
    if bounds.X + bounds.Width > width {
      width = bounds.X + bounds.Width
    }
    if bounds.Y + bounds.Height > height {
      height = bounds.Y + bounds.Height
    }
  }
  if t.Landscape {
    if width > t.MaxSize && t.MaxSize > 0 {
      return 0, 0, fmt.Errorf("Out of width bounds")
    }
    if height > deviceWidth {
      return 0, 0, fmt.Errorf("Out of height bounds")
    }
  } else {
    if width > deviceWidth {
      return 0, 0, fmt.Errorf("Out of width bounds")
    }
    if height > t.MaxSize && t.MaxSize > 0 {
      return 0, 0, fmt.Errorf("Out of height bounds")
    }
  }

  return width, height, nil
}


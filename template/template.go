package template

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"golang.org/x/image/font"
)

type Template struct {
	Id               int
	Name             string
	CreatedAt        time.Time
	Landscape        bool
	MinSize, MaxSize int
	Parameters       []Parameter
	Images           []Image
	Texts            []Text
}

type Parameter struct {
	Id        int
	Name      string
	MaxLength int
}

type Image struct {
	Id            int
	Image         []byte
  LoadedImage   image.Image
	X, Y          int
	Width, Height int
}

type Text struct {
	Id            int
	Text          string
  FilledText    string
	X, Y          int
	Width, Height int
	FontFace      font.Face
}

const deviceWidth = 48 * 8
func RenderTemplate(t *Template, params map[string]string) (image.Image, error) {
  if err := loadFontsForTemplate(t); err != nil {
    return nil, err
  }
  if err := loadImagesForTemplate(t); err != nil {
    return nil, err
  }
  if err := insertParamsIntoTemplate(t, params); err != nil {
    return nil, err
  }
  var width, height int
  if t.Landscape {
    width = t.MinSize
    height = deviceWidth
  } else {
    width = deviceWidth
    height = t.MinSize
  }
  for _, img := range t.Images {
    bounds := measureAndDrawImage(&img, nil)
    if bounds.X + bounds.Width > width {
      width = bounds.X + bounds.Width
    }
    if bounds.Y + bounds.Height > height {
      height = bounds.Y + bounds.Height
    }
  }
  for _, txt := range t.Texts {
    bounds := measureAndDrawText(&txt, nil)
    if bounds.OutOfBounds {
      return nil, fmt.Errorf("Text out of bounds")
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
      return nil, fmt.Errorf("Out of width bounds")
    }
    if height > deviceWidth {
      return nil, fmt.Errorf("Out of height bounds")
    }
  } else {
    if width > deviceWidth {
      return nil, fmt.Errorf("Out of width bounds")
    }
    if height > t.MaxSize && t.MaxSize > 0 {
      return nil, fmt.Errorf("Out of height bounds")
    }
  }

  bounds := image.Rect(0, 0, width, height)
  img := image.NewRGBA64(bounds)
  white := color.RGBA{255, 255, 255, 255} // Define white color
	draw.Draw(img, img.Bounds(), &image.Uniform{C: white}, image.Point{}, draw.Src)
  for _, templateImg := range t.Images {
    measureAndDrawImage(&templateImg, img)
  }

  for _, templateTxt := range t.Texts {
    measureAndDrawText(&templateTxt, img)
  }

  if t.Landscape {
    return rotate90(img), nil
  } else {
    return img, nil
  }
}

func rotate90(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dy(), bounds.Dx() // Swapped dimensions
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newX := bounds.Max.Y - 1 - y // Transpose and reverse the Y-axis
			newY := x
			newImg.Set(newX, newY, img.At(x, y))
		}
	}

	return newImg
}

func insertParamsIntoTemplate(t *Template, params map[string]string) error {
  for _, tp := range t.Parameters {
    if _, exists := params[tp.Name]; !exists {
      return fmt.Errorf("No value for parameter %v", tp.Name)
    }
  }

  for i := 0; i < len(t.Texts); i++ {
    t.Texts[i].FilledText = insertParamsIntoString(t.Texts[i].Text, t, params)
  }

  return nil
}

func insertParamsIntoString(s string, t *Template, params map[string]string) string {
  var sReplaced = s

  for _, tp := range t.Parameters {
    key := fmt.Sprintf("{%v}", tp.Name)
    sReplaced = strings.ReplaceAll(sReplaced, key, params[tp.Name])
  }
  return sReplaced
}

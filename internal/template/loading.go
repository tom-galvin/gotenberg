package template

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/font"
	// "golang.org/x/image/font/gofont/goregular"
  "golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
)

func loadImagesForTemplate(t *Template) error {
  for i := 0; i < len(t.Images); i++ {
    reader := bytes.NewReader(t.Images[i].Image)
    loadedImage, _, err := image.Decode(reader)
    if err != nil {
      return fmt.Errorf("Couldn't load image for template image at index %v:\n%w", i, err)
    }
    t.Images[i].LoadedImage = loadedImage
  }
  return nil
}

func loadFontsForTemplate(t *Template) error {
  for i := 0; i < len(t.Texts); i++ {
    loadedFontFace, err := loadDefaultFont()
    if err != nil {
      return fmt.Errorf("Couldn't load font for template text at index %v:\n%w", i, err)
    }
    t.Texts[i].FontFace = loadedFontFace
  }
  return nil
}

func loadDefaultFont() (font.Face, error) {
  parsedFont, err := opentype.Parse(gomono.TTF)
	if err != nil {
    return nil, fmt.Errorf("Couldn't parse font:\n%w", err)
	}
  
  fontFace, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
    Size: 24,
    DPI: 72,
    Hinting: font.HintingFull,
  })
  if err != nil {
    return nil, fmt.Errorf("Couldn't create font face:\n%w", err)
  }

	return fontFace, nil
}

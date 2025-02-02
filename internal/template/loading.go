package template

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

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
		loadedFontFace, err := loadFont(&t.Texts[i].Font, t.Texts[i].FontSize)
		if err != nil {
			return fmt.Errorf("Couldn't load font for template text at index %v:\n%w", i, err)
		}
		t.Texts[i].FontFace = loadedFontFace
	}
	return nil
}

func getFontData(f *Font) ([]byte, error) {
	if len(f.BuiltinName) > 0 {
		switch f.BuiltinName {
		case "gomono":
			return gomono.TTF, nil
		case "goregular":
			return goregular.TTF, nil
		default:
			return nil, fmt.Errorf(`Unrecognised default font "%s"`, f.BuiltinName)
		}
	} else {
		return f.FontData, nil
	}
}

func loadFont(f *Font, size int) (font.Face, error) {
	fontData, err := getFontData(f)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get font data:\n%w", err)
	}
	parsedFont, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse font %s (%s):\n%w", f.Name, f.Uuid.String(), err)
	}

	fontFace, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("Couldn't create font face:\n%w", err)
	}

	return fontFace, nil
}

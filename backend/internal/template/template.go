package template

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/font"
)

type Template struct {
	Id               int
	Uuid             uuid.UUID
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
	Font          Font
	FontSize      int
	FontFace      font.Face
}

type Font struct {
	Id          int
	Uuid        uuid.UUID
	Name        string
	BuiltinName string
	FontData    []byte
}

const deviceWidth = 48 * 8

func RenderTemplate(t *Template, params map[string]string) (image.Image, error) {
	if err := loadFontsForTemplate(t); err != nil {
		return nil, fmt.Errorf("Couldn't load fonts for template:\n%w", err)
	}
	if err := loadImagesForTemplate(t); err != nil {
		return nil, fmt.Errorf("Couldn't load images for template:\n%w", err)
	}
	if err := insertParamsIntoTemplateChildText(t, params); err != nil {
		return nil, fmt.Errorf("Couldn't insert params into template:\n%w", err)
	}

	var width, height int
	var err error
	if width, height, err = measureAndCheckBounds(t); err != nil {
		return nil, fmt.Errorf("Template children failed boundary check:\n%w", err)
	}

	bounds := image.Rect(0, 0, width, height)
	img := image.NewRGBA64(bounds)
	imgBackgroundColor := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: imgBackgroundColor}, image.Point{}, draw.Src)

	for _, childImage := range t.Images {
		measureAndDrawChildImage(&childImage, img)
	}

	for _, childText := range t.Texts {
		measureAndDrawChildText(&childText, img)
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

func insertParamsIntoTemplateChildText(t *Template, params map[string]string) error {
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
	sReplaced := s

	for _, tp := range t.Parameters {
		key := fmt.Sprintf("{%v}", tp.Name)
		sReplaced = strings.ReplaceAll(sReplaced, key, params[tp.Name])
	}
	return sReplaced
}

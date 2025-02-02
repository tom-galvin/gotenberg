package server

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"tomgalvin.uk/phogoprint/api"
	"tomgalvin.uk/phogoprint/internal/template"
)

func mapTemplateToJson(t *template.Template) *api.Template {
	j := api.Template{
		Id: &t.Id,
		Name: t.Name,
		Landscape: t.Landscape,
		MinSize: t.MinSize,
		MaxSize: t.MaxSize,
	}

	parameters := make([]api.TemplateParameter, len(t.Parameters))
	texts := make([]api.TemplateText, len(t.Texts))
	images := make([]api.TemplateImage, len(t.Images))

	for i := 0; i < len(parameters); i++ {
	  mapParameterToJson(&t.Parameters[i], &parameters[i])
	}

	for i := 0; i < len(texts); i++ {
		mapTextToJson(&t.Texts[i], &texts[i])
	}

	for i := 0; i < len(images); i++ {
		mapImageToJson(&t.Images[i], &images[i])
	}

	j.Parameters, j.Images, j.Texts = &parameters, &images, &texts
	return &j
}

func (s *Server) mapTemplateFromJson(j *api.Template) (*template.Template, error) {
	t := template.Template{
		Id: 0,
		Name: j.Name,
		CreatedAt: time.Now(),
		Landscape: j.Landscape,
		MinSize: j.MinSize,
		MaxSize: j.MaxSize,
	}

	if j.Parameters != nil {
		t.Parameters = make([]template.Parameter, len(*j.Parameters))
		for i := 0; i < len(t.Parameters); i++ {
			mapParameterFromJson(&(*j.Parameters)[i], &t.Parameters[i])
		}
	}
	if j.Texts != nil {
		t.Texts = make([]template.Text, len(*j.Texts))
		for i := 0; i < len(t.Texts); i++ {
			if err := s.mapTextFromJson(&(*j.Texts)[i], &t.Texts[i]); err != nil {
				return nil, err
			}
		}
	}
	if j.Images != nil {
		t.Images = make([]template.Image, len(*j.Images))
		for i := 0; i < len(t.Images); i++ {
			if err := mapImageFromJson(&(*j.Images)[i], &t.Images[i]); err != nil {
				return nil, err
			}
		}	
	}

	return &t, nil
}

func mapParameterToJson(src *template.Parameter, dest *api.TemplateParameter) {
	dest.Name = src.Name
	dest.MaxLength = &src.MaxLength
}

func mapParameterFromJson(src *api.TemplateParameter, dest *template.Parameter) {
	dest.Name = src.Name
	if src.MaxLength != nil {
		dest.MaxLength = *src.MaxLength
	}
}

func mapTextToJson(src *template.Text, dest *api.TemplateText) {
	dest.Text = src.Text
	dest.Position.X = src.X
	dest.Position.Y = src.Y
	dest.FontSize = src.FontSize
	if src.Width > 0 {
		dest.Width = &src.Width
	}
	if src.Height > 0 {
		dest.Height = &src.Height
	}
	dest.FontUuid = src.Font.Uuid.String()
}

func (s *Server) mapTextFromJson(src *api.TemplateText, dest *template.Text) error {
	dest.Text = src.Text
	dest.X = src.Position.X
	dest.Y = src.Position.Y
	dest.FontSize = src.FontSize
	if src.Width != nil {
		dest.Width = *src.Width
	}
	if src.Height != nil {
		dest.Height = *src.Height
	}

	fontUuid, err := uuid.Parse(src.FontUuid)
	if err != nil {
		return fmt.Errorf("Font UUID is not valid:\n%w", err)
	}

	f, err := s.TemplateRepository.GetFont(fontUuid)
	if err != nil {
		return fmt.Errorf("Couldn't load font:\n%w", err)
	}
	if f == nil {
		return fmt.Errorf("Font UUID does not exist:\n%w", f)
	}

	dest.Font = *f
	return nil
}

func mapImageToJson(src *template.Image, dest *api.TemplateImage) {
	dest.Image = base64.StdEncoding.EncodeToString(src.Image)
	dest.Position.X = src.X
	dest.Position.Y = src.Y
	dest.Width = src.Width
	dest.Height = src.Height
}

func mapImageFromJson(src *api.TemplateImage, dest *template.Image) error {
	imageBase64Data, err := base64.StdEncoding.DecodeString(src.Image)
	dest.Image = imageBase64Data
	dest.X = src.Position.X
	dest.Y = src.Position.Y
	dest.Width = src.Width
	dest.Height = src.Height
	return err
}

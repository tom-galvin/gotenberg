package cmd

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"tomgalvin.uk/phogoprint/api"
	"tomgalvin.uk/phogoprint/internal/printer"
	"tomgalvin.uk/phogoprint/internal/template"
)

var _ api.StrictServerInterface = (*Server)(nil)

type Server struct {
	Connection printer.Connection
	TemplateRepository template.TemplateRepository
}

func (s *Server) GetTemplate(ctx context.Context, request api.GetTemplateRequestObject) (api.GetTemplateResponseObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (s *Server) CreateTemplate(ctx context.Context, request api.CreateTemplateRequestObject) (api.CreateTemplateResponseObject, error) {
	r := s.TemplateRepository

	t, err := mapTemplate(request.Body)
	if err != nil {
		return nil, err
	}
	err = r.Transact(func(tx *sql.Tx) error {
		return r.Create(tx, t)
	})
	return api.CreateTemplate201Response{}, nil
}

func mapTemplate(j *api.CreateTemplateJSONRequestBody) (*template.Template, error) {
	t := template.Template{
		Id: 0,
		Name: j.Name,
		CreatedAt: time.Now(),
		Landscape: j.Landscape,
		Parameters: make([]template.Parameter, len(*j.Parameters)),
		Texts: make([]template.Text, len(*j.Texts)),
		Images: make([]template.Image, len(*j.Images)),
	}

	for i := 0; i < len(t.Parameters); i++ {
		mapParameter(&(*j.Parameters)[i], &t.Parameters[i])
	}
	for i := 0; i < len(t.Texts); i++ {
		mapText(&(*j.Texts)[i], &t.Texts[i])
	}
	for i := 0; i < len(t.Images); i++ {
		if err := mapImage(&(*j.Images)[i], &t.Images[i]); err != nil {
			return nil, err
		}
	}

	return &t, nil
}

func mapParameter(src *api.TemplateParameter, dest *template.Parameter) {
	dest.Name = src.Name
	dest.MaxLength = src.MaxLength
}

func mapText(src *api.TemplateText, dest *template.Text) {
	dest.Text = src.Text
	dest.X = src.Position.X
	dest.Y = src.Position.Y
	if src.Width != nil {
		dest.Width = *src.Width
	}
	if src.Height != nil {
		dest.Height = *src.Height
	}
}

func mapImage(src *api.TemplateImage, dest *template.Image) error {
	imageBase64Data, err := base64.StdEncoding.DecodeString(src.Image)
	if err != nil {
		panic(err)
	}
	dest.Image = imageBase64Data
	dest.X = src.Position.X
	dest.Y = src.Position.Y
	dest.Width = src.Width
	dest.Height = src.Height
	return err
}

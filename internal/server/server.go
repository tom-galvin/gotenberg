package server

import (
	"context"
	"database/sql"
	"fmt"

	"tomgalvin.uk/phogoprint/api"
	"tomgalvin.uk/phogoprint/internal/printer"
	"tomgalvin.uk/phogoprint/internal/template"
)

var _ api.StrictServerInterface = (*Server)(nil)

type Server struct {
	Connection printer.Connection
	TemplateRepository *template.TemplateRepository
}

func (s *Server) GetTemplate(ctx context.Context, request api.GetTemplateRequestObject) (api.GetTemplateResponseObject, error) {
	r := s.TemplateRepository
	t, err := r.Get(request.Id)
	if err != nil {
		return nil, fmt.Errorf("Couldn't fetch template:\n%w", err)
	}
	if t != nil {
		return api.GetTemplate200JSONResponse(*mapTemplateToJson(t)), nil
	} else {
		return api.GetTemplate404Response{}, nil
	}
}

func (s *Server) CreateTemplate(ctx context.Context, request api.CreateTemplateRequestObject) (api.CreateTemplateResponseObject, error) {
	fmt.Println("create")
	r := s.TemplateRepository

	t, err := mapTemplateFromJson(request.Body)
	if err != nil {
		return nil, err
	}
	err = r.Transact(func(tx *sql.Tx) error {
		return r.Create(tx, t)
	})
	if err != nil {
		return nil, err
	}
	return api.CreateTemplate201Response{}, nil
}


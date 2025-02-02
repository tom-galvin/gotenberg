package server

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"log/slog"

	_ "image/jpeg"
	_ "image/png"

	"tomgalvin.uk/phogoprint/api"
	"tomgalvin.uk/phogoprint/internal/printer"
	"tomgalvin.uk/phogoprint/internal/template"
)

var _ api.StrictServerInterface = (*Server)(nil)

type Server struct {
	Connection         printer.Connection
	TemplateRepository *template.TemplateRepository
}

func (s *Server) PrintImage(ctx context.Context, request api.PrintImageRequestObject) (api.PrintImageResponseObject, error) {
	imageData, err := request.Body.Data.Bytes()
	if err != nil {
		return api.PrintImage422Response{}, nil
	}
	image, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return api.PrintImage422Response{}, nil
	}

	fmt.Printf("Received %s image\n", format)

	if err := s.Connection.Connect(); err != nil {
		slog.Error("Couldn't connect to printer", "error", err)
		return api.PrintImage503Response{}, nil
	} else {
		err = s.Connection.GetPrinter().WriteImage(image)
		if err != nil {
			slog.Error("Couldn't write image to printer", "error", err)
			return api.PrintImage503Response{}, nil
		}

		return api.PrintImage202Response{}, nil
	}
}

func (s *Server) PrintTemplate(ctx context.Context, request api.PrintTemplateRequestObject) (api.PrintTemplateResponseObject, error) {
	r := s.TemplateRepository
	t, err := r.Get(request.Id)
	if err != nil {
		return nil, fmt.Errorf("Couldn't fetch template:\n%w", err)
	}
	if t == nil {
		return api.PrintTemplate404Response{}, nil
	}

	paramsMap := make(map[string]string, len(t.Parameters))

	// this part could be more clever
	for _, param := range t.Parameters {
		isPresent := false
		for _, requestParam := range request.Body.ParameterValues {
			if param.Name == requestParam.ParameterName {
				paramsMap[param.Name] = requestParam.Value
				isPresent = true
				break
			}
		}
		if !isPresent {
			return api.PrintTemplate422JSONResponse{
				Reason: fmt.Sprintf(`Missing parameter "%s"`, param.Name),
			}, nil
		}
	}

	img, err := template.RenderTemplate(t, paramsMap)
	if err != nil {
		return api.PrintTemplate422JSONResponse{
			Reason: err.Error(),
		}, nil
	}

	if err := s.Connection.Connect(); err != nil {
		slog.Error("Couldn't connect to printer", "error", err)
		return api.PrintTemplate503Response{}, nil
	} else {
		err = s.Connection.GetPrinter().WriteImage(img)
		if err != nil {
			slog.Error("Couldn't write image to printer", "error", err)
			return api.PrintTemplate503Response{}, nil
		}

		return api.PrintTemplate202Response{}, nil
	}
}

func (s *Server) ListFont(ctx context.Context, request api.ListFontRequestObject) (api.ListFontResponseObject, error) {
	fs, err := s.TemplateRepository.ListFonts()
	if err != nil {
		panic(err)
	}
	fsJson := make([]api.Font, len(fs))
	for i := 0; i < len(fs); i++ {
		fsJson[i].Name = fs[i].Name
		fsJson[i].Uuid = fs[i].Uuid.String()
	}
	return api.ListFont200JSONResponse(fsJson), nil
}

func (s *Server) GetPrinterInfo(ctx context.Context, request api.GetPrinterInfoRequestObject) (api.GetPrinterInfoResponseObject, error) {
	if !s.Connection.GetPrinter().IsConnected() {
		return api.GetPrinterInfo503Response{}, nil
	} else {
		info := s.Connection.GetPrinter().Info()
		return api.GetPrinterInfo200JSONResponse{
			BatteryLevel:    info.BatteryLevel,
			State:           mapDeviceStateToJson(info.State),
			FirmwareVersion: info.FirmwareVersion,
		}, nil
	}
}

func mapDeviceStateToJson(s printer.DeviceState) api.DeviceState {
	switch s {
	case printer.Disconnected:
		return api.DISCONNECTED
	case printer.Connecting:
		return api.CONNECTING
	case printer.Ready:
		return api.READY
	case printer.Busy:
		return api.BUSY
	case printer.OutOfPaper:
		return api.OUTOFPAPER
	default:
		panic(fmt.Errorf("Unknown device state %v", s))
	}
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

func (s *Server) ListTemplate(ctx context.Context, request api.ListTemplateRequestObject) (api.ListTemplateResponseObject, error) {
	ts, err := s.TemplateRepository.List()
	if err != nil {
		panic(err)
	}
	tsJson := make([]api.Template, len(ts))
	for i := 0; i < len(ts); i++ {
		tsJson[i] = *mapTemplateToJson(&ts[i])
	}
	return api.ListTemplate200JSONResponse(tsJson), nil
}

func (s *Server) CreateTemplate(ctx context.Context, request api.CreateTemplateRequestObject) (api.CreateTemplateResponseObject, error) {
	r := s.TemplateRepository

	t, err := s.mapTemplateFromJson(request.Body)
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

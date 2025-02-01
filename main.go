package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"log/slog"
	"net/http"
	"os"

	"tomgalvin.uk/phogoprint/api"
	"tomgalvin.uk/phogoprint/internal/server"
	"tomgalvin.uk/phogoprint/internal/model"
	"tomgalvin.uk/phogoprint/internal/printer"
)

//go:embed Banana.jpg
var img []byte

func main() {
	fmt.Println("Hello, Phogoprint!")
	r := NewRepository()

	var conn *printer.BluetoothConnection
	/* conn, err := printer.FromBluetoothName("T02")
	if err != nil {
		slog.Error("Couldn't find printer", "err", err)
		return
	} */

	// conn.Connect()

	// defer conn.Disconnect()

	

	mux := http.NewServeMux()
	si := server.Server{
		TemplateRepository: r,
		Connection:         conn,
	}
	sh := api.NewStrictHandler(&si, nil)
	h := http.StripPrefix("/api", api.Handler(sh))


	mux.Handle("/api/", h)

	mux.Handle("/", http.FileServer(http.Dir("resources/web")))

	mux.HandleFunc("/print", func(w http.ResponseWriter, r *http.Request) {
		handlePrint(conn, w, r)
	})

	mux.HandleFunc("/battery", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET method is supported", http.StatusMethodNotAllowed)
			return
		}
		if !conn.GetPrinter().IsConnected() {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Not connected")
		} else {
			info := conn.GetPrinter().Info()

			infoData, err := json.Marshal(model.FromDeviceInfo(info))
			if err != nil {
				panic("fuck!")
			}

			w.WriteHeader(http.StatusOK)
			if _, err = w.Write(infoData); err != nil {
				slog.Error("Couldn't write HTTP response", "error", err)
			}
		}
	})

	port := "8080"
	fmt.Printf("Starting server on port %s...\n", port)
	server := http.Server{Addr:":"+port,Handler:mux}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}

func handlePrint(p printer.Connection, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "image/png" && contentType != "image/jpeg" {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	image, format, err := image.Decode(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Couldn't read %s data: %v", contentType, err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("Received %s image\n", format)

	if err := p.Connect(); err != nil {
		slog.Error("Couldn't connect to printer!", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		err = p.GetPrinter().WriteImage(image)

		if err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}

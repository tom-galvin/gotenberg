package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"tomgalvin.uk/phogoprint/api"
	"tomgalvin.uk/phogoprint/internal/server"
	"tomgalvin.uk/phogoprint/internal/printer"
)

func main() {
	fmt.Println("Hello, Phogoprint!")

	r := NewRepository()
	defer r.Close()

	var conn *printer.BluetoothConnection
	conn, err := printer.FromBluetoothName("T02")
	if err != nil {
		slog.Error("Couldn't find printer", "err", err)
		return
	}

	mux := http.NewServeMux()
	logger := slog.Default()
	si := server.NewServer(logger.With("src", "server"), conn, r)
	sh := api.NewStrictHandler(si, nil)
	h := http.StripPrefix("/api", api.Handler(sh))


	mux.Handle("/api/", h)

	mux.Handle("/", http.FileServer(http.Dir("resources/web")))

	port := "8080"
	fmt.Printf("Starting server on port %s...\n", port)
	server := http.Server{Addr:":"+port,Handler:mux}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
  "fmt"
  "log/slog"
  "os"
  "net/http"
  "image"
  _ "image/png"
  _ "image/jpeg"
  "gotenburg/printer"
  "gotenburg/printer/phomemo"
)

func main() {
  fmt.Println("Hello, Gotenburg!")
  provider, err := phomemo.CreateProvider()

  fmt.Println("Scanning for devices...")
  if err = provider.FindDevice("T02"); err != nil {
    slog.Error("Couldn't find printer", "err", err)
    return
  }

  defer provider.Disconnect()

  http.Handle("/", http.FileServer(http.Dir("http")))

  http.HandleFunc("/print", func(w http.ResponseWriter, r *http.Request) {
    handlePrint(provider, w, r)
  })

  http.HandleFunc("/battery", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
      http.Error(w, "Only GET method is supported", http.StatusMethodNotAllowed)
      return
    }
    if !provider.GetPrinter().IsConnected() {
      w.WriteHeader(http.StatusServiceUnavailable)
      fmt.Fprintf(w, "Not connected")
    } else {
      level, err := provider.GetPrinter().GetBatteryLevel()

      if err == nil {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "%v", level)
      } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        fmt.Fprintf(w, "Printer disconnected during read")
      }
    }
  })

  port := "8080"
  fmt.Printf("Starting server on port %s...\n", port)
  if err := http.ListenAndServe(":"+port, nil); err != nil {
    fmt.Printf("Error starting server: %v\n", err)
    os.Exit(1)
  }
}

func handlePrint(p printer.PrinterProvider, w http.ResponseWriter, r *http.Request) {
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

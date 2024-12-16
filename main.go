package main

import (
  "fmt"
  "log/slog"
  "io"
  "os"
  "net/http"
  "encoding/json"
  "gotenburg/model"
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

  _, err = provider.GetPrinter()

  if err != nil {
    fmt.Println("Couldn't connect to printer", err)
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
    if pr, err := provider.GetPrinter(); err != nil {
      slog.Error("Couldn't connect to printer")
      w.WriteHeader(http.StatusServiceUnavailable)
      fmt.Fprintf(w, "printer not connected")
    } else {
      level, err := pr.GetBatteryLevel()

      if err == nil {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "%v", level)
      } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        fmt.Fprintf(w, "printer not connected")
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

  if r.Header.Get("Content-Type") != "application/octet-stream" {
    http.Error(w, "Invalid content type", http.StatusBadRequest)
    return
  }

  body, err := io.ReadAll(r.Body)
  if err != nil {
    http.Error(w, "Failed to read request body", http.StatusInternalServerError)
    return
  }
  defer r.Body.Close()

  fmt.Printf("Received %d bytes\n", len(body))

  var request model.PrintingRequest
  if err = json.Unmarshal(body, &request); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
  }

  var bitmap printer.Bitmap
  if bitmap, err = printer.BitmapFromRequest(&request); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
  }

  packedBitmap := printer.PackBitmap(bitmap)

  if pr, err := p.GetPrinter(); err != nil {
    slog.Error("Couldn't connect to printer!", "error", err)
    w.WriteHeader(http.StatusServiceUnavailable)
  } else {
    err = pr.WriteBitmap(packedBitmap)

    if err == nil {
      w.WriteHeader(http.StatusOK)
    } else {
      w.WriteHeader(http.StatusServiceUnavailable)
    }
  }
}

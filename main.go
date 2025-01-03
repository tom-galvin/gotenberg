package main

import (
  "fmt"
  "log/slog"
  "os"
  "net/http"
  "image"
  "encoding/json"
  _ "image/png"
  _ "image/jpeg"
  "gotenburg/model"
  "gotenburg/printer"
  "gotenburg/printer/phomemo"
)

func main() {
  fmt.Println("Hello, Gotenburg!")
  provider, err := phomemo.FromBluetoothName("T02")

  if err != nil {
    slog.Error("Couldn't find printer", "err", err)
    return
  }

  provider.Connect()

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
      info := provider.GetPrinter().Info()

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

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

  "tomgalvin.uk/phogoprint/model"
  "tomgalvin.uk/phogoprint/printer"
  "tomgalvin.uk/phogoprint/template"
)

//go:embed Banana.jpg
var img []byte

func templateTest() *template.Template {
  t := template.Template{
    Texts: []template.Text{
      {
        Text:  "hello world the quick brown {param1} {param2} {param1} jumps over the lazy dog jackdaws love my big sphinx of quartz",
        X:     10,
        Y:     10,
        Width: 48 * 7,
      },
    },
    Images: []template.Image{
      {
        X:      30,
        Y:      30,
        Width:  100,
        Height: 100,
        Image:  img,
      },
    },
    Parameters: []template.Parameter{
      {
        Name: "param1",
      },
      {
        Name: "param2",
      },
    },
    Landscape: false,
    MinSize:   100,
    MaxSize:   200,
  }
  return &t
}

func main() {
  fmt.Println("Hello, Phogoprint!")
  t := templateTest()
  DbConnect(t)
  conn, err := printer.FromBluetoothName("T02")
  if err != nil {
    slog.Error("Couldn't find printer", "err", err)
    return
  }

  conn.Connect()

  defer conn.Disconnect()
  conn.Connect()

  http.Handle("/", http.FileServer(http.Dir("resources/http")))

  http.HandleFunc("/print", func(w http.ResponseWriter, r *http.Request) {
    handlePrint(conn, w, r)
  })

  http.HandleFunc("/battery", func(w http.ResponseWriter, r *http.Request) {
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
  if err := http.ListenAndServe(":"+port, nil); err != nil {
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

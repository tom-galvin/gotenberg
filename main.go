package main

import (
	"fmt"
  "io"
  "os"
  "net/http"
  "gotenburg/printer/phomemo"

	"tinygo.org/x/bluetooth"
)

func main() {
	fmt.Println("Hello, Gotenburg!")

	adapter := bluetooth.DefaultAdapter

	err := adapter.Enable()
	if err != nil {
		fmt.Println("Failed to enable Bluetooth: ", err)
		return
	}

	fmt.Println("Scanning for devices...")

  provider := phomemo.PhomemoPrinterProvider{}
  printer, err := provider.GetPrinter(adapter)

  if err != nil {
    fmt.Println("Couldn't get printer", err)
    return
  }

  defer printer.Close()

  http.Handle("/", http.FileServer(http.Dir("http")))

  http.HandleFunc("/print", func(w http.ResponseWriter, r *http.Request) {
		// Ensure the request is a POST
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
			return
		}

		// Check the Content-Type
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			http.Error(w, "Invalid content type", http.StatusBadRequest)
			return
		}

		// Read the body as a byte array
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Log the size of the data received
		fmt.Printf("Received %d bytes\n", len(body))

    err = printer.WriteData(body)

    if err == nil {
      // Respond to the client
      w.WriteHeader(http.StatusOK)
      w.Write([]byte("Upload successful"))
    } else {
      w.WriteHeader(http.StatusServiceUnavailable)
      w.Write([]byte(err.Error()))
    }
	})

	// Start the server on port 8080
	port := "8080"
	fmt.Printf("Starting server on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}

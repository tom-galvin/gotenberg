package phomemo

import (
  "fmt"
  "errors"
  "gotenburg/printer"
  "tinygo.org/x/bluetooth"
)

type DeviceType byte
const (
  Service DeviceType = 0x00
  Writer DeviceType = 0x02
  Notifier DeviceType = 0x03
)

func getUUID(t DeviceType) bluetooth.UUID {
  return bluetooth.NewUUID([16]byte{
    0x00, 0x00, 0xff, byte(t), 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5f, 0x9b, 0x34, 0xfb,
  })
}

type PhomemoPrinter struct {
  device bluetooth.Device
  writer bluetooth.DeviceCharacteristic
}

func (p *PhomemoPrinter) Close() error {
  p.device.Disconnect()
  return nil
}

func (p *PhomemoPrinter) WriteData(data []byte) error {
  _, err := p.writer.WriteWithoutResponse(data)
  if err != nil {
    return err
  }
  return nil
}

type PhomemoPrinterProvider struct {
}

func (p *PhomemoPrinterProvider) GetPrinter(adapter *bluetooth.Adapter) (printer.Printer, error) {
  devices := make(chan bluetooth.ScanResult, 1)

  go func() {
    err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
      if result.LocalName() == "T02" {
        fmt.Println("Found device:", result.LocalName())
        devices <- result
        adapter.StopScan()
      }
    })
    if err != nil {
      fmt.Println("Failed to scan for devices:", err)
      close(devices)
    }
  }()

  dev, ok := <-devices

  if !ok {
    fmt.Println("No devices found.")
    return nil, errors.New("No devices found")
  }

  fmt.Println("Connecting to device...")
  peripheral, err := adapter.Connect(dev.Address, bluetooth.ConnectionParams{})
  if err != nil {
    fmt.Println("Failed to connect to device:", err)
    return nil, err
  }

  // Discover the primary service (UUID 0xFF00)
  fmt.Println("Discovering service...")
  services, err := peripheral.DiscoverServices([]bluetooth.UUID{getUUID(Service)})
  if err != nil {
    fmt.Println("Failed to discover service:", err)
    return nil, err
  }

  fmt.Println("Discovering characteristics...")
  characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{getUUID(Writer), getUUID(Notifier)})
  if err != nil {
    fmt.Println("Failed to discover characteristics:", err)
    return nil, err
  }

  writer, notifier := characteristics[0], characteristics[1]

  printer := PhomemoPrinter {
    device: peripheral,
    writer: writer,
  }
  err = notifier.EnableNotifications(func (d []byte) {
    fmt.Printf("Received notification:%x\n",d)
  })
  if err != nil {
    fmt.Println("Couldn't enable notifications:",err)
  }

  return &printer, nil
}

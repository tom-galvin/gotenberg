package phomemo

import (
  "log/slog"
  "errors"
  "time"
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


type BluetoothProvider struct {
}

func (p *BluetoothProvider) GetPrinter(adapter *bluetooth.Adapter) (printer.Printer, error) {
  devices := make(chan bluetooth.ScanResult, 1)

  go func() {
    err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
      if result.LocalName() == "T02" {
        slog.Info("Found device:",
          "deviceName", result.LocalName(),
        )
        devices <- result
        adapter.StopScan()
      }
    })
    if err != nil {
      slog.Error("Failed to scan for devices:",
        "err", err,
      )
      close(devices)
    }
  }()

  dev, ok := <-devices

  if !ok {
    return nil, errors.New("No devices found")
  }

  slog.Debug("Connecting to device...")
  peripheral, err := adapter.Connect(dev.Address, bluetooth.ConnectionParams{})
  if err != nil {
    slog.Error("Failed to connect to device:",
      "err", err,
    )
    return nil, err
  }

  // Discover the primary service (UUID 0xFF00)
  slog.Debug("Discovering service...")
  services, err := peripheral.DiscoverServices([]bluetooth.UUID{getUUID(Service)})
  if err != nil {
    slog.Error("Failed to discover service:",
      "err", err,
    )
    peripheral.Disconnect()
    return nil, err
  }

  slog.Debug("Discovering characteristics...")
  characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{getUUID(Writer), getUUID(Notifier)})
  if err != nil {
    slog.Error("Failed to discover characteristics:",
      "err", err,
    )
    peripheral.Disconnect()
    return nil, err
  }

  writer, notifier := characteristics[0], characteristics[1]

  printer := PhomemoBluetoothPrinter {
    device: peripheral,
    writer: writer,
    queue: make(chan PhomemoAction),
    batteryLevel: -1,
    batteryLevelChannel: make(chan int),
    statusTicker: time.NewTicker(10 * time.Second),
    ready: make(chan bool),
  }
  err = notifier.EnableNotifications(func (d []byte) {
    printer.handleData(d)
  })
  if err != nil {
    slog.Error("Couldn't enable notifications:",
      "err", err,
    )
    peripheral.Disconnect()
    return nil, err
  }

  go printer.startWriteQueue()
  go printer.statusTickerFunc()

  return &printer, nil
}

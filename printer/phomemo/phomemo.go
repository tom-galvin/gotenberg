package phomemo

import (
  "log/slog"
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

type BluetoothProvider struct {
  adapter *bluetooth.Adapter
  address bluetooth.Address
  printer *BluetoothPrinter
  device bluetooth.Device
}

func CreateProvider() (*BluetoothProvider, error) {
  adapter := bluetooth.DefaultAdapter

  err := adapter.Enable()
  if err != nil {
    slog.Error("Failed to enable Bluetooth: ", "err", err)
    return nil, err
  }

  printer := BluetoothPrinter{connected:false}

  adapter.SetConnectHandler(func(d bluetooth.Device, connected bool) {
    if connected {
      slog.Info("Connected!")
    } else {
      slog.Info("Disconnected!")
      printer.uninitialise()
    }
  })
  
  return &BluetoothProvider{adapter:adapter, printer:&printer}, nil
}

func (p *BluetoothProvider) Disconnect() error {
  if p.printer.IsConnected() {
    p.device.Disconnect()
  }
  return nil
}

func (p *BluetoothProvider) Connect() error {
  if !p.printer.IsConnected() {
    var err error
    if err = p.connect(); err != nil {
      slog.Error("Couldn't connect to bluetooth printer", "error", err)
      return err
    }
    if err = p.printer.initialise(); err != nil {
      slog.Error("Couldn't initialise bluetooth printer after connect", "error", err)
      return err
    }
  }
  return nil
}

func (p *BluetoothProvider) GetPrinter() printer.Printer {
  return p.printer
}

func (p *BluetoothProvider) FindDevice(name string) error {
  devices := make(chan bluetooth.ScanResult, 1)

  go func() {
    err := p.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
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
    return errors.New("No devices found")
  }

  p.address = dev.Address
  return nil
}

func (p *BluetoothProvider) connect() error {
  slog.Debug("Connecting to device...")
  device, err := p.adapter.Connect(p.address, bluetooth.ConnectionParams{})
  if err != nil {
    slog.Error("Failed to connect to device:",
      "err", err,
    )
    return err
  }

  // Discover the primary service (UUID 0xFF00)
  slog.Debug("Discovering service...")
  services, err := device.DiscoverServices([]bluetooth.UUID{getUUID(Service)})
  if err != nil {
    slog.Error("Failed to discover service:",
      "err", err,
    )
    device.Disconnect()
    return err
  }

  slog.Debug("Discovering characteristics...")
  characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{getUUID(Writer), getUUID(Notifier)})
  if err != nil {
    slog.Error("Failed to discover characteristics:",
      "err", err,
    )
    device.Disconnect()
    return err
  }

  p.printer.device = device
  p.printer.writer, p.printer.notifier = characteristics[0], characteristics[1]
  return nil
}

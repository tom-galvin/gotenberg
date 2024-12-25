package phomemo

import (
  "log/slog"
  "errors"
  "fmt"
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
  printer *PhomemoPrinter
  writer bluetooth.DeviceCharacteristic
  notifier bluetooth.DeviceCharacteristic
  device bluetooth.Device
  address bluetooth.Address
}

func CreateProvider() (*BluetoothProvider, error) {
  adapter := bluetooth.DefaultAdapter

  err := adapter.Enable()
  if err != nil {
    slog.Error("Failed to enable Bluetooth: ", "err", err)
    return nil, err
  }

  printer := &PhomemoPrinter{connected:false}

  provider := &BluetoothProvider{adapter:adapter, printer:printer}
  adapter.SetConnectHandler(func(d bluetooth.Device, connected bool) {
    if connected {
      slog.Info("Connected!")
    } else {
      if d.Address == provider.address && printer.IsConnected() {
        slog.Info("Disconnected!")
        printer.uninitialise()
      } else {
        slog.Info("Disconnected event fired but printer is not connected or address doesn't match")
      }
    }
  })

  return provider, nil
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

func (p *BluetoothProvider) Write(data []byte) error {
  _, err := p.writer.WriteWithoutResponse(data)

  if err != nil {
    slog.Error("Couldn't write data", "error", err)
  } else {
    slog.Debug("Wrote data to device", "size", len(data))
  }

  return err
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
    // connect to bluetooth device & get characteristics
    if err = p.connect(); err != nil {
      slog.Error("Couldn't connect to bluetooth printer", "error", err)
      return err
    }

    // initialise controller stuff like channels
    if err = p.printer.initialise(p); err != nil {
      slog.Error("Couldn't initialise bluetooth printer after connect", "error", err)
      return err
    }

    // enable notifications from device to receive ready notification/battery info etc
    err = p.notifier.EnableNotifications(func (d []byte) {
      onBluetoothDataReceived(p.printer, d)
    })

    if err != nil {
      slog.Error("Couldn't enable notifications:",
        "error", err,
      )

      return err
    }
  
  }
  return nil
}

func (p *BluetoothProvider) GetPrinter() printer.Printer {
  return p.printer
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
  p.writer = characteristics[0]
  p.notifier = characteristics[1]

  p.device = device
  return nil
}

func onBluetoothDataReceived(p *PhomemoPrinter, d []byte) {
  switch {
  case hasPrefix(d, 0x1a, 0x0f, 0x0c):
    p.onReady()
  case hasPrefix(d, 0x1a, 0x3b, 0x04):
    // only happens with later firmware versions
    slog.Debug("Printer info:", "info", d[3:])
  case hasPrefix(d, 0x1a, 0x04):
    p.onBatteryLevelChange(int(d[2]))
  case hasPrefix(d, 0x1a, 0x07):
    p.onFirmwareVersionReceived(fmt.Sprintf("%v.%v.%v", d[2], d[3], d[4]))
  case hasPrefix(d, 0x1a, 0x06) && (d[2] == 0x88 || d[2] == 0x89):
    p.onPaperStatusChange(d[2] & 1 == 1)
  case hasPrefix(d, 0x01, 0x01):
    slog.Debug("Read command successfully")
  case hasPrefix(d, 0x02, 0xb6, 0x00):
    // only happens with later firmware versions
    p.onReady()
  default:
    slog.Info("Received unknown notification:",
      "data", fmt.Sprintf("%x", d),
    )
  }
}

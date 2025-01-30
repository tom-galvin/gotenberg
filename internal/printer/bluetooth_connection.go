// This package is built with the assumption that the server will only be
// connected to a single bluetooth device at a time; this will need to be
// ripped up if we want to manage e.g. multiple bluetooth devices at once
package printer

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"tinygo.org/x/bluetooth"
)

type DeviceType byte
const (
  Service DeviceType = 0x00
  Writer DeviceType = 0x02
  Notifier DeviceType = 0x03
)

type BluetoothConnection struct {
  device bluetooth.Device
  adapter *bluetooth.Adapter
  writer bluetooth.DeviceCharacteristic
  notifier bluetooth.DeviceCharacteristic
  printer PhomemoPrinter
  address bluetooth.Address
}

func getUUID(t DeviceType) bluetooth.UUID {
  return bluetooth.NewUUID([16]byte{
    0x00, 0x00, 0xff, byte(t), 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5f, 0x9b, 0x34, 0xfb,
  })
}

func newBluetoothConnection() (*BluetoothConnection, error) {
  adapter := bluetooth.DefaultAdapter

  err := adapter.Enable()
  if err != nil {
    slog.Error("Failed to enable Bluetooth: ", "err", err)
    return nil, err
  }

  conn := &BluetoothConnection{adapter:adapter}
  adapter.SetConnectHandler(func(d bluetooth.Device, connected bool) {
    if connected {
      slog.Info("Connected!")
    } else {
      if d.Address == conn.address && conn.printer.IsConnected() {
        slog.Info("Disconnected!")
        conn.printer.uninitialise()
      } else {
        slog.Info("Disconnected event fired but printer is not connected or address doesn't match")
      }
    }
  })

  return conn, nil
}

func FromBluetoothName(name string) (*BluetoothConnection, error) {
  p, err := newBluetoothConnection()

  if err != nil {
    slog.Error("Couldn't initialise conn", "error", err)
    return nil, err
  }

  devices := make(chan bluetooth.ScanResult, 1)

  go func() {
    err := p.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
      if result.LocalName() == name {
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

  p.address = dev.Address
  return p, nil
}

func FromBluetoothAddress(address bluetooth.Address) (*BluetoothConnection, error) {
  p, err := newBluetoothConnection()

  if err != nil {
    slog.Error("Couldn't initialise connection", "error", err)
    return nil, err
  }

  p.address = address
  return p, nil
}

func (p *BluetoothConnection) Write(data []byte) error {
  _, err := p.writer.WriteWithoutResponse(data)

  if err != nil {
    slog.Error("Couldn't write data", "error", err)
  } else {
    slog.Debug("Wrote data to device", "size", len(data))
  }

  return err
}

func (p *BluetoothConnection) Disconnect() error {
  if p.printer.IsConnected() {
    p.device.Disconnect()
  }
  return nil
}

func (p *BluetoothConnection) Connect() error {
  if !p.printer.IsConnected() {
    var err error
    // connect to bluetooth device & get characteristics
    if err = p.connect(); err != nil {
      slog.Error("Couldn't connect to bluetooth printer", "error", err)
      return err
    }

    c := make(chan bool)
    p.printer = initialise(p, c)

    // enable notifications from device to receive ready notification/battery info etc
    err = p.notifier.EnableNotifications(func (data []byte) {
      handleBluetoothDataFromPrinter(data, &p.printer)
    })

    if err != nil {
      slog.Error("Couldn't enable notifications:",
        "error", err,
      )
      p.device.Disconnect()
      return err
    }

    if !<-c {
      return fmt.Errorf("Printer disconnected before becoming ready")
    }

    close(c)
  }
  return nil
}

func (p *BluetoothConnection) GetPrinter() Printer {
  return &p.printer
}

func (p *BluetoothConnection) connect() error {
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

func hasPrefix(d []byte, p ...byte) bool {
  return len(d) >= len(p) && bytes.Equal(d[:len(p)], p)
}

func handleBluetoothDataFromPrinter(d []byte, p *PhomemoPrinter) {
  switch {
  case hasPrefix(d, 0x02, 0xb6, 0x00):
    p.onReady()
  case hasPrefix(d, 0x1a, 0x0f, 0x0c):
    p.onFinished()
  case hasPrefix(d, 0x1a, 0x3b, 0x04):
    // only seen this with later firmware version
    slog.Debug("Printer info:", "info", d[3:])
  case hasPrefix(d, 0x1a, 0x04):
    p.onBatteryLevelChange(int(d[2]))
  case hasPrefix(d, 0x1a, 0x07):
    p.onFirmwareVersionReceived(fmt.Sprintf("%v.%v.%v", d[2], d[3], d[4]))
  case hasPrefix(d, 0x1a, 0x06) && (d[2] == 0x88 || d[2] == 0x89):
    p.onPaperStatusChange(d[2] & 1 == 1)
  case hasPrefix(d, 0x01, 0x01):
    slog.Debug("Read command successfully")
  default:
    slog.Info("Received unknown notification:",
      "data", fmt.Sprintf("%x", d),
    )
  }
}

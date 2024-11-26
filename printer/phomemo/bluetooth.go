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

type PhomemoBluetoothPrinter struct {
  device bluetooth.Device
  writer bluetooth.DeviceCharacteristic
}

func (p *PhomemoBluetoothPrinter) Close() error {
  p.device.Disconnect()
  return nil
}

func (p *PhomemoBluetoothPrinter) WriteData(data []byte) error {
  _, err := p.writer.WriteWithoutResponse(data)
  if err != nil {
    return err
  }
  return nil
}

const maxBitmapHeight = 256
func (p *PhomemoBluetoothPrinter) WriteBitmap(b *printer.PackedBitmap) error {
  if b.Stride() > 0x30 {
    return fmt.Errorf("Bitmap too wide for printer: %s", b)
  }
  strideU8 := byte(b.Stride())

  commands := [][]byte{
    initPrinter(),
    setJustify(Centre),
    setLaserIntensity(Low),
  }

  for bitmapStart := 0; bitmapStart < b.Height(); bitmapStart += maxBitmapHeight {
    bitmapEnd := bitmapStart + maxBitmapHeight

    if bitmapEnd >= b.Height() {
      bitmapEnd = b.Height()
    }

    slice := b.Chunk(bitmapStart, bitmapEnd - bitmapStart)
    sliceHeightU16 := uint16(slice.Height())

    commands = append(commands,
      printBitmap(strideU8, sliceHeightU16),
      slice.Data(),
    )
  }

  commands = append(commands,
    feedLines(4),
    queryBatteryStatus(),
  )

  for _, command := range commands {
    _, err := p.writer.WriteWithoutResponse(command)
    if err != nil {
      return err
    }
  }

  return nil
}

type BluetoothProvider struct {
}

func (p *BluetoothProvider) GetPrinter(adapter *bluetooth.Adapter) (printer.Printer, error) {
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

  printer := PhomemoBluetoothPrinter {
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

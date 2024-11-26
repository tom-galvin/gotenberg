
package phomemo

import (
  "fmt"
  "time"
  "log/slog"
  "gotenburg/printer"
  "tinygo.org/x/bluetooth"
)

type Action struct {
  bitmapToPrint *printer.PackedBitmap
  fetchStatus bool
}

type BluetoothPrinter struct {
  device bluetooth.Device
  writer bluetooth.DeviceCharacteristic
  queue chan Action
  batteryLevelChannel chan int
  ready chan bool
  statusTicker *time.Ticker
  batteryLevel int
}

func NewPrinter(device bluetooth.Device, writer bluetooth.DeviceCharacteristic, notifier bluetooth.DeviceCharacteristic) (*BluetoothPrinter, error) {
  printer := BluetoothPrinter {
    device: device,
    writer: writer,
    queue: make(chan Action),
    batteryLevel: -1,
    batteryLevelChannel: make(chan int),
    statusTicker: time.NewTicker(10 * time.Second),
    ready: make(chan bool),
  }
  err := notifier.EnableNotifications(func (d []byte) {
    printer.handleData(d)
  })
  if err != nil {
    slog.Error("Couldn't enable notifications:",
      "err", err,
    )
    device.Disconnect()
    return nil, err
  }

  go printer.startWriteQueue()
  go printer.statusTickerFunc()

  return &printer, nil
}

func (p *BluetoothPrinter) Close() error {
  p.device.Disconnect()
  p.statusTicker.Stop()
  return nil
}

func (p *BluetoothPrinter) GetBatteryLevel() int {
  if p.batteryLevel < 0 {
    slog.Info("Battery level queried before it's ready, fetching now")
    p.pollBatteryLevel()
    return <-p.batteryLevelChannel
  } else {
    return p.batteryLevel
  }
}

func (p *BluetoothPrinter) WriteBitmap(b *printer.PackedBitmap) {
  p.queue <- Action{
    bitmapToPrint: b,
  }
}

func (p *BluetoothPrinter) statusTickerFunc() {
  for range p.statusTicker.C {
    slog.Info("Polling for battery status")
    p.pollBatteryLevel()
  }
}

func (p *BluetoothPrinter) pollBatteryLevel() {
  p.queue <- Action{
    fetchStatus: true,
  }
}

func (p *BluetoothPrinter) handleData(d []byte) {
  if len(d) > 2 && d[0] == 0x1A {
    switch d[1] {
    case 0x0f:
      if len(d) == 3 && d[2] == 0x0c {
        slog.Info("Printer finished printing")
        p.ready <- true
        return
      }
    case 0x3b:
      if len(d) > 3 && d[2] == 0x04 {
        slog.Info("Printer ready to accept data")
        // I think this is a ready packet?
        p.ready <- true
        return
      }
    case 0x04:
      if len(d) == 3 {
        batteryLevel := (int(d[2]))
        slog.Info("Battery level:",
          "level", batteryLevel,
        )
        if p.batteryLevel < 0 {
          // if this is the first time we've recorded the battery level then push to a channel in
          // case anything is waiting for it
          p.batteryLevelChannel <- batteryLevel
        }
        p.batteryLevel = batteryLevel
        return
      }
    case 0x07:
      if len(d) == 5 {
        slog.Info("Firmware version:",
          "firmwareVersion", fmt.Sprintf("%v.%v.%v", d[2], d[3], d[4]),
        )
        return
      }
    case 0x06:
      if len(d) == 3 {
        switch d[2] {
        case 0x88, 0x89:
          slog.Info("Paper status:",
            "loaded", d[2] & 1 == 1,
          )
          return
        }
      }
    }
  } else if len(d) == 2 && d[0] == 0x01 && d[1] == 0x01 {
    // Think the printer outputs this whenever it receives data successfully
    return
  }

  slog.Info("Received unknown notification:",
    "data", fmt.Sprintf("%x", d),
  )
}

func (p *BluetoothPrinter) startWriteQueue() {
  for action := range p.queue {
    commands := [][]byte{
      initPrinter(),
    }

    if action.bitmapToPrint != nil {
      slog.Info("Waiting for printer to become ready...")
      <-p.ready
      slog.Info("Executing action to print bitmap")
      commands = append(commands,
        setJustify(Centre),
        setLaserIntensity(Low),
      )
      writeBitmapAsCommands(action.bitmapToPrint, &commands)
      commands = append(commands,
        feedLines(4),
      )
    }

    if action.fetchStatus {
      slog.Info("Executing action to fetch status")
      commands = append(commands,
        queryBatteryStatus(),
        queryFirmwareVersion(),
      )
    }

    for _, command := range commands {
      _, err := p.writer.WriteWithoutResponse(command)
      if err != nil {
        slog.Error("Couldn't write command data",
          "err", err,
        )
        break
      }
    }
  }
}

const maxBitmapHeight = 256
func writeBitmapAsCommands(b *printer.PackedBitmap, commands *[][]byte) error {
  if b.Stride() > 0x30 {
    return fmt.Errorf("Bitmap too wide for printer: %s", b)
  }
  strideU8 := byte(b.Stride())

  for bitmapStart := 0; bitmapStart < b.Height(); bitmapStart += maxBitmapHeight {
    bitmapEnd := bitmapStart + maxBitmapHeight

    if bitmapEnd >= b.Height() {
      bitmapEnd = b.Height()
    }

    slice := b.Chunk(bitmapStart, bitmapEnd - bitmapStart)
    sliceHeightU16 := uint16(slice.Height())

    *commands = append(*commands,
      printBitmap(strideU8, sliceHeightU16),
      slice.Data(),
    )
  }

  return nil
}

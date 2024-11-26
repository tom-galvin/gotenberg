
package phomemo

import (
  "fmt"
  "time"
  "log/slog"
  "gotenburg/printer"
  "tinygo.org/x/bluetooth"
)

type PhomemoAction struct {
  bitmapToPrint *printer.PackedBitmap
  fetchStatus bool
}

type PhomemoBluetoothPrinter struct {
  device bluetooth.Device
  writer bluetooth.DeviceCharacteristic
  queue chan PhomemoAction
  batteryLevelChannel chan int
  ready chan bool
  statusTicker *time.Ticker
  batteryLevel int
}

func (p *PhomemoBluetoothPrinter) Close() error {
  p.device.Disconnect()
  p.statusTicker.Stop()
  return nil
}

func (p *PhomemoBluetoothPrinter) GetBatteryLevel() int {
  if p.batteryLevel < 0 {
    slog.Info("Battery level queried before it's ready, fetching now")
    p.pollBatteryLevel()
    return <-p.batteryLevelChannel
  } else {
    return p.batteryLevel
  }
}

func (p *PhomemoBluetoothPrinter) WriteBitmap(b *printer.PackedBitmap) {
  p.queue <- PhomemoAction{
    bitmapToPrint: b,
    fetchStatus: false,
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

func (p *PhomemoBluetoothPrinter) statusTickerFunc() {
  for range p.statusTicker.C {
    slog.Info("Polling for battery status")
    p.pollBatteryLevel()
  }
}

func (p *PhomemoBluetoothPrinter) pollBatteryLevel() {
  p.queue <- PhomemoAction{
    bitmapToPrint: nil,
    fetchStatus: true,
  }
}

func (p *PhomemoBluetoothPrinter) handleData(d []byte) {
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
        batteryLevel := (int(d[2]) * 100 / 0xFF)
        slog.Info("Battery level:",
          "level", batteryLevel,
        )
        p.batteryLevel = batteryLevel
        p.batteryLevelChannel <- batteryLevel
        return
      }
    }
  } else if len(d) == 2 && d[0] == 0x01 && d[1] == 0x01 {
    // Think the print outputs this whenever it receives data successfully
    return
  }

  slog.Info("Received unknown notification:",
    "data", d,
  )
}

func (p *PhomemoBluetoothPrinter) startWriteQueue() {
  for action := range p.queue {
    commands := [][]byte{
      initPrinter(),
      setJustify(Centre),
      setLaserIntensity(Low),
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

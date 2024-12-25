package phomemo

import (
  "fmt"
  "bytes"
  "time"
  "log/slog"
  "image"
  "gotenburg/printer"
)

type PhomemoPrinter struct {
  ready chan bool
  printChannel chan printer.PackedBitmap
  statusTicker *time.Ticker
  batteryLevel int
  connected bool
}

type DeviceWriter interface {
  Write(data []byte) error
}

func (p *PhomemoPrinter) initialise(w DeviceWriter) error {
  p.ready = make(chan bool)
  p.connected = true

  // start printer event loop
  p.statusTicker = time.NewTicker(10 * time.Second)
  go p.eventLoop(w)
  return nil
}

func (p *PhomemoPrinter) uninitialise() error {
  p.statusTicker.Stop()
  close(p.ready)
  p.connected = false

  return nil
}

func (p *PhomemoPrinter) IsConnected() bool {
  return p.connected
}

func (p *PhomemoPrinter) GetBatteryLevel() (int, error) {
  return p.batteryLevel, nil
}

func (p *PhomemoPrinter) WriteImage(i image.Image) error {
  if pb, err := packImageToPhomemoBitmap(i); err != nil {
    slog.Error("Image couldn't be packed to bitmap", "error", err)
    return err
  } else {
    return p.writePackedBitmap(pb)
  }
}

func (p *PhomemoPrinter) writePackedBitmap(b *printer.PackedBitmap) error {
  if !p.connected {
    return fmt.Errorf("Printer is not connected")
  }
  select {
  case p.printChannel <- *b:
    return nil
  default:
    return fmt.Errorf("Device disconnected while waiting for print operation")
  }
}

func (p *PhomemoPrinter) onReady() {
  select {
  case p.ready <- true:
    slog.Info("Printer finished action")
  default:
    slog.Info("Printer wasn't waiting")
  }
}

func (p *PhomemoPrinter) onBatteryLevelChange(level int) {
  slog.Debug("Battery level:",
    "level", level,
  )
  p.batteryLevel = level
}

func (p *PhomemoPrinter) onPaperStatusChange(loaded bool) {
  slog.Info("Reader: paper status changed",
    "loaded", loaded,
  )
}

func (p *PhomemoPrinter) onFirmwareVersionReceived(version string) {
  slog.Info("Reader: ping received",
    "firmwareVersion", version,
  )
}

func hasPrefix(d []byte, b ...byte) bool {
  return len(d) >= len(b) && bytes.Equal(d[:len(b)], b)
}

func (p *PhomemoPrinter) eventLoop(w DeviceWriter) {
  slog.Info("Writer: Waiting for printer to become ready after connect")

  for <-p.ready {
    commands := [][]byte{initPrinter()}

    select {
    case bitmapToPrint := <-p.printChannel:
      slog.Info("Writer: Sending bitmap data to printer")
      commands = append(commands,
        setJustify(Centre),
        setLaserIntensity(Low),
      )
      writeBitmapAsCommands(&bitmapToPrint, &commands)
      commands = append(commands,
        feedLines(4),
      )
    case <-p.statusTicker.C:
      slog.Info("Writer: Pinging printer status")
      commands = append(commands,
        queryBatteryStatus(),
        queryFirmwareVersion(),
      )
    }

    for _, command := range commands {
      if err := w.Write(command); err != nil {
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

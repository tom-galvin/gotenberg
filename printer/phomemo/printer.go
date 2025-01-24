package phomemo

import (
  "fmt"
  "time"
  "log/slog"
  "image"
  "sync"
  "gotenburg/printer"
  "gotenburg/render"
)

type PhomemoPrinter struct {
  connected chan bool
  finished chan bool
  writer DeviceWriter
  statusTicker *time.Ticker
  info printer.DeviceInfo
  printLock sync.Mutex
}

type DeviceWriter interface {
  Write(data []byte) error
}

func initialise(w DeviceWriter, c chan bool) PhomemoPrinter {
  info := printer.DeviceInfo{
    State: printer.Connecting,
  }
  return PhomemoPrinter{
    connected: c,
    finished: make(chan bool),
    statusTicker: time.NewTicker(10 * time.Second),
    writer: w,
    info: info,
  }
}

func (p *PhomemoPrinter) uninitialise() error {
  p.statusTicker.Stop()
  close(p.finished)
  p.info.State = printer.Disconnected

  return nil
}

func (p *PhomemoPrinter) IsConnected() bool {
  return p.info.State != printer.Disconnected
}

func (p *PhomemoPrinter) Info() printer.DeviceInfo {
  return p.info
}

func (p *PhomemoPrinter) pollStatus() error {
  if p.info.State != printer.Disconnected && p.info.State != printer.Busy {
    p.printLock.Lock()
    defer p.printLock.Unlock()
    if p.info.State != printer.Disconnected && p.info.State != printer.Busy {
      slog.Debug("Polling device status")
      data := initPrinter()
      data = append(data, queryBatteryStatus()...)
      data = append(data, queryPaperStatus()...)
      data = append(data, queryFirmwareVersion()...)
      return p.writer.Write(data)
    }
  }

  // control falls through to this if either of the Ready checks fail
  // the mutex unlock was also deferred so that'll happen if needed
  return fmt.Errorf("Printer is not in ready state")
}

func (p *PhomemoPrinter) WriteImage(i image.Image) error {
  ig := render.RenderForDevice(i)
  b, err := printer.FromPaletted(ig)
  pb := printer.PackBitmap(b)

  if err != nil {
    slog.Error("Couldn't create packed bitmap from paletted image", "error", err)
    return err
  }

  slog.Debug("Acquiring lock on printer state")
  if p.info.State == printer.Ready {
    p.printLock.Lock()
    defer p.printLock.Unlock()
    if p.info.State == printer.Ready {
      p.info.State = printer.Busy

      if err := p.sendPackedBitmapToPrinter(pb); err != nil {
        return err
      }

      // The device sometimes outputs an early "finished printing" signal
      // right after the bitmap data is written, as well as the later one
      // which the device outputs after printing is finished.
      // A small delay is added here before waiting for the signal to 
      // ignore the initial spurious one.
      // This could probably be more elegant, but printing anything takes
      // at least 1 second anyway, so sleeping for 100ms doesn't introduce
      // any additional delay to the process.
      time.Sleep(100 * time.Millisecond)
      slog.Info("Waiting for printer to finish printing")
      if !<-p.finished {
        // TODO: add a timeout so it doesn't block forever and deadlock if the
        // printer gets stuck?
        return fmt.Errorf("Printer didn't finish successfully")
      }

      slog.Info("Printer finished printing")
      p.info.State = printer.Ready

      return nil
    }
  }

  // Control falls through to this if either of the Ready checks fail.
  // The mutex unlock was also deferred, so that'll happen now if needed
  return fmt.Errorf("Printer is not in ready state")
}

func (p *PhomemoPrinter) WriteImageOld(i image.Image) error {
  if pb, err := packImageToPhomemoBitmap(i); err != nil {
    slog.Error("Image couldn't be packed to bitmap", "error", err)
    return err
  } else {
    slog.Debug("Acquiring lock on printer state")
    if p.info.State == printer.Ready {
      p.printLock.Lock()
      defer p.printLock.Unlock()
      if p.info.State == printer.Ready {
        p.info.State = printer.Busy

        if err := p.sendPackedBitmapToPrinter(pb); err != nil {
          return err
        }

        // The device sometimes outputs an early "finished printing" signal
        // right after the bitmap data is written, as well as the later one
        // which the device outputs after printing is finished.
        // A small delay is added here before waiting for the signal to 
        // ignore the initial spurious one.
        // This could probably be more elegant, but printing anything takes
        // at least 1 second anyway, so sleeping for 100ms doesn't introduce
        // any additional delay to the process.
        time.Sleep(100 * time.Millisecond)
        slog.Info("Waiting for printer to finish printing")
        if !<-p.finished {
          // TODO: add a timeout so it doesn't block forever and deadlock if the
          // printer gets stuck?
          return fmt.Errorf("Printer didn't finish successfully")
        }

        slog.Info("Printer finished printing")
        p.info.State = printer.Ready

        return nil
      }
    }

    // Control falls through to this if either of the Ready checks fail.
    // The mutex unlock was also deferred, so that'll happen now if needed
    return fmt.Errorf("Printer is not in ready state")
  }
}

func (p *PhomemoPrinter) sendPackedBitmapToPrinter(b *printer.PackedBitmap) error {
  data := initPrinter()
  data = append(data, setJustify(Centre)...)
  data = append(data, setLaserIntensity(Low)...)
  splitBitmapIntoCommands(b, &data)
  data = append(data, feedLines(4)...)
  return p.writer.Write(data)
}

func (p *PhomemoPrinter) onReady() {
  slog.Info("Printer ready for printing")

  if err := p.pollStatus(); err != nil {
    slog.Error("Couldn't poll status", "error", err)
  }

  // start consuming ticker to periodically refresh device details
  go (func() {
    for range p.statusTicker.C {
      if err := p.pollStatus(); err != nil {
        slog.Error("Couldn't poll status", "error", err)
      }
    }
  })()
}

func (p *PhomemoPrinter) onFinished() {
  select {
  case p.finished <- true:
    // unblocks WriteImage if we're waiting to finish printing something
  default:
    // otherwise just ignore the signal
  }
}

func (p *PhomemoPrinter) onBatteryLevelChange(level int) {
  p.info.BatteryLevel = level
}

func (p *PhomemoPrinter) onPaperStatusChange(loaded bool) {
  oldState := p.info.State
  if loaded && p.info.State != printer.Busy {
    p.info.State = printer.Ready
  } else if !loaded {
    p.info.State = printer.OutOfPaper
  }
  if oldState == printer.Connecting {
    p.connected <- true
  }
}

func (p *PhomemoPrinter) onFirmwareVersionReceived(version string) {
  p.info.FirmwareVersion = version
}

// TODO: to support the other phomemo devices this shouldn't be hardcoded
const maxBitmapHeight = 256

// writes the bitmap into the provided buffer, splitting the bitmap up into
// several if the height is greater than the max supported bitmap height for
// the device
func splitBitmapIntoCommands(b *printer.PackedBitmap, d *[]byte) error {
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

    *d = append(*d,
      printBitmap(strideU8, sliceHeightU16)...,
    )

    *d = append(*d,
      slice.Data()...,
    )
  }

  return nil
}

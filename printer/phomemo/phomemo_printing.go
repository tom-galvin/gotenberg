package phomemo

import (
  "fmt"
  "bytes"
  "time"
  "log/slog"
  "gotenburg/printer"
  "tinygo.org/x/bluetooth"
)

type Action struct {
  bitmapToPrint *printer.PackedBitmap
  fetchStatus bool
  isBlocking bool
}

type BluetoothPrinter struct {
  device bluetooth.Device
  writer bluetooth.DeviceCharacteristic
  notifier bluetooth.DeviceCharacteristic
  queue chan Action
  succeeded chan bool
  ready chan bool
  statusTicker *time.Ticker
  batteryLevel int
  connected bool
}

func (p *BluetoothPrinter) initialise() error {
  p.queue = make(chan Action)
  p.ready = make(chan bool)
  p.succeeded = make(chan bool)
  p.statusTicker = time.NewTicker(10 * time.Second)

  err := p.notifier.EnableNotifications(func (d []byte) {
    p.handleData(d)
  })
  if err != nil {
    slog.Error("Couldn't enable notifications:",
      "err", err,
    )

    p.uninitialise()
    return err
  }

  p.connected = true

  go p.startWriteQueue()
  go p.statusTickerFunc()
  return nil
}

func (p *BluetoothPrinter) uninitialise() error {
  p.device.Disconnect()
  p.statusTicker.Stop()
  close(p.queue)
  close(p.ready)
  close(p.succeeded)
  p.connected = false

  return nil
}

func (p *BluetoothPrinter) IsConnected() bool {
  return p.connected
}

func (p *BluetoothPrinter) GetBatteryLevel() (int, error) {
  if !p.connected {
    return -1, fmt.Errorf("Device not connected")
  }

  if p.batteryLevel < 0 {
    slog.Info("Battery level queried before it's ready, fetching now")
    p.pollBatteryLevel(true)

    succeeded := <-p.succeeded
    if !succeeded {
      return -1, fmt.Errorf("Device disconnected before battery level received")
    }
  }

  return p.batteryLevel, nil
}

func (p *BluetoothPrinter) WriteBitmap(b *printer.PackedBitmap) error {
  if !p.connected {
    return fmt.Errorf("Printer is not connected")
  }
  p.queue <- Action{
    bitmapToPrint: b,
    isBlocking: true,
  }
  succeeded := <-p.succeeded
  if !succeeded {
    return fmt.Errorf("Device disconnected while waiting for print operation")
  }
  return nil
}

func (p *BluetoothPrinter) statusTickerFunc() {
  for range p.statusTicker.C {
    slog.Debug("Polling for battery status")
    p.pollBatteryLevel(false)
  }
}

func (p *BluetoothPrinter) pollBatteryLevel(block bool) {
  p.queue <- Action{
    fetchStatus: true,
    isBlocking: block,
  }
}

func (p *BluetoothPrinter) unblock(action string) {
  select {
  case p.ready <- true:
    slog.Info("Printer finished action:",
      "action", action)
  default:
    slog.Debug("Printer wasn't waiting:",
      "action", action)
  }
}

func hasPrefix(d []byte, b ...byte) bool {
  return len(d) >= len(b) && bytes.Equal(d[:len(b)], b)
}

func (p *BluetoothPrinter) handleData(d []byte) {
  switch {
  case hasPrefix(d, 0x1a, 0x0f, 0x0c):
    p.unblock("Print")
  case hasPrefix(d, 0x1a, 0x3b, 0x04):
    slog.Info("Printer info:", "info", d[3:])
  case hasPrefix(d, 0x1a, 0x04):
    batteryLevel := (int(d[2]))
    slog.Info("Battery level:",
      "level", batteryLevel,
    )
    p.batteryLevel = batteryLevel
  case hasPrefix(d, 0x1a, 0x07):
    slog.Info("Firmware version:",
      "firmwareVersion", fmt.Sprintf("%v.%v.%v", d[2], d[3], d[4]),
    )
  case hasPrefix(d, 0x1a, 0x06) && (d[2] == 0x88 || d[2] == 0x89):
    slog.Info("Paper status:",
      "loaded", d[2] & 1 == 1,
    )
  case hasPrefix(d, 0x01, 0x01):
    slog.Debug("Read command successfully")
  case hasPrefix(d, 0x02, 0xb6, 0x00):
    p.unblock("Connect")
  default:
    slog.Info("Received unknown notification:",
      "data", fmt.Sprintf("%x", d),
    )
  }
}

func (p *BluetoothPrinter) startWriteQueue() {
  slog.Info("Waiting for printer to become ready after connect")
  <-p.ready
  counter := 0
  slog.Info("Waiting for action", "counter", counter)
  for action := range p.queue {
    commands := [][]byte{
      initPrinter(),
    }

    if action.bitmapToPrint != nil {
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

    if action.isBlocking {
      slog.Info("Waiting for printer to become ready after action", "counter", counter)
      _, ok := <-p.ready

      if p.connected && ok {
        slog.Info("Completing action")
        p.succeeded <- true
      } else {
        slog.Warn("Printer disconnected while waiting for command completion")

        if p.connected != ok {
          slog.Warn("Potential race condition?",
            "connected", p.connected,
            "ok", ok)
        }

        p.succeeded <- false
      }
    }
    counter += 1
    slog.Info("Waiting for action", "counter", counter)
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

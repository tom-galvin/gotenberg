package printer

import (
  "image"
)

type Printer interface {
  WriteImage(image.Image) error
  GetBatteryLevel() (int, error)
  IsConnected() bool
}

type PrinterProvider interface {
  GetPrinter() Printer
  Connect() error
  Disconnect() error
}

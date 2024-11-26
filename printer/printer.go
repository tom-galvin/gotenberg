package printer

import (
  "tinygo.org/x/bluetooth"
)

type Printer interface {
  WriteBitmap(b *PackedBitmap)
  GetBatteryLevel() int

  Close() error
}

type PrinterProvider interface {
  GetPrinter(adapter *bluetooth.Adapter) (Printer, error)
}

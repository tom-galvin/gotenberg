package printer

import (
  "tinygo.org/x/bluetooth"
)

type Printer interface {
  WriteData(data []byte) error
  WriteBitmap(b *PackedBitmap) error

  Close() error
}

type PrinterProvider interface {
  GetPrinter(adapter *bluetooth.Adapter) (Printer, error)
}

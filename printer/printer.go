package printer

type Printer interface {
  WriteBitmap(b *PackedBitmap) error
  GetBatteryLevel() (int, error)
  IsConnected() bool
}

type PrinterProvider interface {
  GetPrinter() (Printer, error)
  Disconnect() error
}

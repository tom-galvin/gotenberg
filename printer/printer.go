package printer

type Printer interface {
  WriteBitmap(b *PackedBitmap) error
  GetBatteryLevel() (int, error)
  IsConnected() bool
}

type PrinterProvider interface {
  GetPrinter() Printer
  Connect() error
  Disconnect() error
}

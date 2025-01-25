package printer

import (
  "image"
)

type DeviceState int

const (
  Disconnected DeviceState = iota
  Connecting
  Ready
  Busy
  OutOfPaper
)

func (s DeviceState) String() string {
  switch s {
  case Disconnected: return "Disconnected"
  case Connecting: return "Connecting"
  case Ready: return "Ready"
  case Busy: return "Busy"
  case OutOfPaper: return "OutOfPaper"
  default: return "Unknown"
  }
}

type DeviceInfo struct {
  FirmwareVersion string
  BatteryLevel int
  State DeviceState
}

type Printer interface {
  WriteImage(image.Image) error
  Info() DeviceInfo
  IsConnected() bool
}

type Connection interface {
  GetPrinter() Printer
  Connect() error
  Disconnect() error
}

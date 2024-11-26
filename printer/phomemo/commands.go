package phomemo

// These aren't used yet, the frontend still orchestrates all of the commands
// TODO: make the backend do the interesting bit & make the frontend just do
// the image rendering

const (
  Esc = 0x1B
  GS = 0x1D
  US = 0x1F
)

type Justify byte
const (
  Left Justify = 0x00
  Centre Justify = 0x01
  Right Justify = 0x02
)

type LaserIntensity byte
const (
  Low LaserIntensity = 0x01
  Medium LaserIntensity = 0x03
  High LaserIntensity = 0x04
)

func initPrinter() []byte {
  return []byte{Esc, 0x40}
}

func setJustify(justify Justify) []byte {
  return []byte{Esc, 0x61, byte(justify)}
}

func setLaserIntensity(intensity LaserIntensity) []byte {
  return []byte{US, 0x11, 0x02, byte(intensity)}
}

func printBitmap(widthBytes byte, heightBits uint16) []byte {
  return []byte{
    GS, 0x76, 0x30, 0x00,
    widthBytes, 0x00,
    byte(heightBits & 0xFF), byte(heightBits >> 8),
  }
}

func feedLines(n byte) []byte {
  return []byte{Esc, 0x64, n}
}

func queryDeviceTimer() []byte {
  return []byte{US, 0x11, 0x0E}
}

func queryBatteryStatus() []byte {
  return []byte{US, 0x11, 0x08}
}

func queryPaperStatus() []byte {
  return []byte{US, 0x11, 0x11}
}

func queryFirmwareVersion() []byte {
  return []byte{US, 0x11, 0x07}
}

func queryDeviceSerial() []byte {
  return []byte{US, 0x11, 0x09}
}

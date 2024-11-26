// This file implements the various Epson ESC/POS command byte sequences that can be
// written to Phomemo T02/M02/T02S printers.
package phomemo

// Control characters
const (
  Esc = 0x1B
  GS = 0x1D
  US = 0x1F
)

// Type alias for the image alignment of a printed bitmap
type Justify byte
const (
  Left Justify = 0x00
  Centre Justify = 0x01
  Right Justify = 0x02
)

// Type alias for the laser intensity of a printed bitmap
type LaserIntensity byte
const (
  Low LaserIntensity = 0x01
  Medium LaserIntensity = 0x03
  High LaserIntensity = 0x04
)

// Initialises the printer & prepares it to accept commands
func initPrinter() []byte {
  return []byte{Esc, 0x40}
}

// Sets the image alignment/justification of the bitmap to print.
// Note: only centre alignment seems to work on T02 printers
func setJustify(justify Justify) []byte {
  return []byte{Esc, 0x61, byte(justify)}
}

// Sets the laser intensity which affects the clarity/quality of the printed image
func setLaserIntensity(intensity LaserIntensity) []byte {
  return []byte{US, 0x11, 0x02, byte(intensity)}
}

// Prepares the printer to print bitmap data specified by the width and height passed in.
// widthBytes specifies the width of the bitmap data in bytes, with 8 pixels packed into 1 byte.
// heightBits specifies the height of the bitmap data in rows.
// After this command is written, (widthBytes * heightBits) bytes of data must then be written
func printBitmap(widthBytes byte, heightBits uint16) []byte {
  return []byte{
    GS, 0x76, 0x30, 0x00,
    widthBytes, 0x00,
    byte(heightBits & 0xFF), byte(heightBits >> 8),
  }
}

// Makes the printer spool through a number of blank lines.
func feedLines(n byte) []byte {
  return []byte{Esc, 0x64, n}
}

// Queries the amount of time remaining before the printer automatically powers off.
// TODO: output format
func queryDeviceTimer() []byte {
  return []byte{US, 0x11, 0x0E}
}

// Queries the battery status of the printer.
// TODO: output format
func queryBatteryStatus() []byte {
  return []byte{US, 0x11, 0x08}
}

// Queries the status of the paper loaded & whether the top lid is open or not.
// TODO: output format
func queryPaperStatus() []byte {
  return []byte{US, 0x11, 0x11}
}

// Queries the version of the firmware running on the device.
// TODO: output format
func queryFirmwareVersion() []byte {
  return []byte{US, 0x11, 0x07}
}

// Queries the serial number of the device.
// TODO: output format
func queryDeviceSerial() []byte {
  return []byte{US, 0x11, 0x09}
}

const CONTROL_CHARS = {
  ESC: 0x1B,
  GS: 0x1D,
  US: 0x1F
};

export const COMMANDS = {
  initPrinter: () => // ESC @
    new Uint8Array([CONTROL_CHARS.ESC, 0x40]),
  setJustify: (mode) => // ESC a <byte: mode>
    new Uint8Array([CONTROL_CHARS.ESC, 0x61, mode & 0xFF]),
  setLaserIntensity: (intensity) => // US 0x11 0x02 <byte: intensity>
    new Uint8Array([CONTROL_CHARS.US, 0x11, 0x02, intensity & 0xFF]),
  printBitmap: (widthBytes, heightBits) => // GS v 0 <byte: mode> <u16: widthBytes> <u16: heightBits>
    new Uint8Array([CONTROL_CHARS.GS, 0x76, 0x30, 0x00, widthBytes & 0xFF, widthBytes >> 8, heightBits & 0xFF, heightBits >> 8]),
  feedLines: (n) => // ESC d <byte: n>
    new Uint8Array([CONTROL_CHARS.ESC, 0x64, n & 0xFF]),
  queryDeviceTimer: () => // US 0x11 0x0E
    new Uint8Array([CONTROL_CHARS.US, 0x11, 0x0E]),
  queryBatteryStatus: () => // US 0x11 0x08
    new Uint8Array([CONTROL_CHARS.US, 0x11, 0x08]),
  queryPaperStatus: () => // US 0x11 0x11
    new Uint8Array([CONTROL_CHARS.US, 0x11, 0x11]),
  queryFirmwareVersion: () => // US 0x11 0x07
    new Uint8Array([CONTROL_CHARS.US, 0x11, 0x07]),
  queryDeviceId: () => // US 0x11 0x09
    new Uint8Array([CONTROL_CHARS.US, 0x11, 0x09]),
  JUSTIFY: {
    LEFT: 0x00, CENTER: 0x01, RIGHT: 0x02
  },
  LASER_INTENSITY: {
    LOW: 0x01, MEDIUM: 0x03, HIGH: 0x04
  }
};

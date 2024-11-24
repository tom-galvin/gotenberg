import { COMMANDS } from './commands.js';
const FONT_HEIGHT = 17;
const FONT_NAME = 'Iosevka';

const setPrintHeight = (y) => {
  canvas.height = y;
  document.getElementById("bitmap-height").innerHTML = y;
};

const canvas = document.getElementById('theCanvas');
canvas.width = 384; 
canvas.height = 128; 

document.getElementById('text').addEventListener('input', (event) => {
  let text = event.target.value;
  let lines = text.split('\n');
  setPrintHeight(lines.length * (FONT_HEIGHT + 1) + 1);

  const ctx = canvas.getContext('2d');

  ctx.fillStyle = 'white';
  ctx.fillRect(0, 0, 384, canvas.height);
  
  ctx.fillStyle = 'black'; 
  
  ctx.font = `400 ${FONT_HEIGHT}px ${FONT_NAME}`; 
  ctx.textAlign = 'left'; 
  ctx.textBaseline = 'top'; 

  for (let i = 0; i < lines.length; i++) {
    ctx.fillText(lines[i], 1, 1 + (FONT_HEIGHT + 1) * i);
  }
});

var dither = [
  [0, 1, 0, 1],
  [2, 3, 2, 3],
  [0, 1, 0, 1],
  [2, 3, 2, 3]
];
const convertImageToBitmapData = () => {

};
const createCommandsForImage = (device) => {
  /**
   * This turns the image into the canvas into one or more commands to the printer to print a bitmap.
   * The printer supports printing bitmaps of a specific maximium height. Any images taller than this
   * must be split into more bitmaps-the printer prints these end-on-end with no gap, so it functions
   * the same as printing a single taller image.
   * We could change the laser intensity between each bitmap if we wanted to.
   */
  const ctx = canvas.getContext('2d');
  const imgw = canvas.width, imgh = canvas.height;
  const imageData = ctx.getImageData(0, 0, imgw, imgh);
  const pixelArray = imageData.data;
  const widthInBytes = Math.ceil(imgw / 8); // row width is in bytes, round up if it's not multiple of 8

  let remainingRows = imgh;
  let commands = [];
  do {
    console.log(`${remainingRows} remaining rows`);
    const currentYInImage = imgh - remainingRows;
    const bitmapHeight = Math.min(device.MAX_BITMAP_HEIGHT, remainingRows);
    const bitBuffer = [];
    for(let j = 0; j < bitmapHeight; j++) {
      for(let i = 0; i < widthInBytes; i++) {
        let p = 0;
        for(let b = 0; b < 8 && (8 * i) + b < imgw; b++) {
          // basic bitch dithering (this is actually quite shit)
          const x = 8 * i + b, y = currentYInImage + j;

          const idx = 4 * ((48 * 8) * y + x);
          const ditherThreshold = 32 + 64 * dither[y % 4][x % 4];

          if (pixelArray[idx] < ditherThreshold) {
            p |= (1 << (7 - b));
          }
        }
        bitBuffer.push(p);
      }
    }

    commands.push(COMMANDS.printBitmap(widthInBytes, bitmapHeight));
    commands.push(new Uint8Array(bitBuffer));
    console.log(`Pushed ${bitmapHeight}/${imgh} rows to bitmap`);

    remainingRows -= bitmapHeight;
  } while(remainingRows > 0);

  return commands;
};

function mergeUint8Arrays(arrays) {
  // Calculate the total length of all Uint8Arrays
  const totalLength = arrays.reduce((sum, arr) => sum + arr.length, 0);

  // Create a new Uint8Array with the total length
  const result = new Uint8Array(totalLength);

  // Copy each Uint8Array into the result
  let offset = 0;
  for (const arr of arrays) {
    result.set(arr, offset);
    offset += arr.length;
  }

  return result;
}

const DEVICE_TYPES = {
  // add more as necessary, this is the only one I ahve
  PHOMEMO_T02: {
    MAX_WIDTH_BYTES: 0x30,
    MAX_BITMAP_HEIGHT: 0xFF
  }
};
const printImageServer = async () => {
  const commands = [
    COMMANDS.initPrinter(),
    COMMANDS.setJustify(COMMANDS.JUSTIFY.CENTER),
    COMMANDS.setLaserIntensity(COMMANDS.LASER_INTENSITY.HIGH),
    ...createCommandsForImage(DEVICE_TYPES.PHOMEMO_T02),
    COMMANDS.feedLines(4),
    COMMANDS.queryBatteryStatus(),
    COMMANDS.queryDeviceId(),
    COMMANDS.queryFirmwareVersion(),
    COMMANDS.queryDeviceTimer(),
    COMMANDS.queryPaperStatus()
  ];

  try {
    // Perform the POST request
    const dataToSend = mergeUint8Arrays(commands);
    console.log(dataToSend);
    const response = await fetch("/print", {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
      },
      body: dataToSend, // Send the Uint8Array as the payload
    });

    // Ensure the response is OK
    if (!response.ok) {
      logger.error(`HTTP error! Status: ${response.status}`);
    }

    // Read the response as text
    const responseText = await response.text();

    // Return or store the response as a string
    alert(responseText);
    return responseText;
  } catch (error) {
    console.error("Error during POST request:", error);
    throw error; // Rethrow or handle appropriately
  }
};

const printImageToBluetooth = async () => {
  let primaryService, printer, notifier;
  try {
    const device = await navigator.bluetooth.requestDevice({
      filters: [
        { name: "T02" }
      ],
      optionalServices: [0xFF00, 0xFF02],
    });

    const gattServer = await device.gatt.connect();
    primaryService = await gattServer.getPrimaryService(0xFF00); 
    printer = await primaryService.getCharacteristic(0xFF02);      
    notifier = await primaryService.getCharacteristic(0xFF03);      
  } catch (error) {
    throw error;
  }

  const commands = [
    COMMANDS.initPrinter(),
    COMMANDS.setJustify(COMMANDS.JUSTIFY.CENTER),
    COMMANDS.setLaserIntensity(COMMANDS.LASER_INTENSITY.HIGH),
    ...createCommandsForImage(DEVICE_TYPES.PHOMEMO_T02),
    COMMANDS.feedLines(4),
    COMMANDS.queryBatteryStatus(),
    COMMANDS.queryDeviceId(),
    COMMANDS.queryFirmwareVersion(),
    COMMANDS.queryDeviceTimer(),
    COMMANDS.queryPaperStatus()
  ];

  let receivedPackets = [], pendingResolvers = [];
  (await notifier.startNotifications()).addEventListener("characteristicvaluechanged", async (e) => {
    const buffer = e.target.value, bytes = [];
    for (let i = 0; i < buffer.byteLength; i++) {
      bytes.push(buffer.getUint8(i));
    }
    receivedPackets.push(e.target.value);
    if (pendingResolvers.length > 0) {
      const resolve = pendingResolvers.shift();
      resolve(receivedPackets.shift());
    }
  });
      
  const getNextPacket = () => {
    if(receivedPackets.length > 0) {
      return Promise.resolve(receivedPackets.shift());
    } else {
      return new Promise(resolve => pendingResolvers.push(resolve));
    }
  };

  await writeCommandsToDevice(commands, printer);
  console.log('Waiting for response from device');
  let packet;
  while(packet = await getNextPacket()) {
    if(packet.getUint8(0) == 0x1A) { // device update
      const type = packet.getUint8(1);
      if (type === 0x0F) break;
      // TODO: do something interesting with the other types
    }
  }
  console.log('Done');
};

const writeCommandsToDevice = async (commands, device) => {
  const blob = new Blob(commands);
  const arrayBuffer = await blob.arrayBuffer();
  const length = arrayBuffer.byteLength, sliceSize = BLE_MAX_PACKET_SIZE;
  const chunks = [];
  for(let i = 0; i < length; i += sliceSize) {
    const j = Math.min(length, i + sliceSize);
    await device.writeValueWithoutResponse(arrayBuffer.slice(i, j));
  }
}

document.getElementById("idButton").onclick = async() => {
  await printImageServer();
};

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
  const ctx = canvas.getContext('2d');
  const imgw = canvas.width, imgh = canvas.height;
  const imageData = ctx.getImageData(0, 0, imgw, imgh);
  const pixelArray = imageData.data;

  const data = [];
  for (let y = 0; y < imgh; y++) {
    for (let x = 0; x < imgw; x++) {
      const ditherThreshold = 32 + 64 * dither[y % 4][x % 4];
      const idx = 4 * (imgw * y + x);
      data.push(pixelArray[idx] < ditherThreshold ? 1 : 0);
    }
  }
  return {
    width: imgw,
    height: imgh,
    data: btoa((new TextDecoder('utf8')).decode(new Uint8Array(data)))
  };
};

const DEVICE_TYPES = {
  // add more as necessary, this is the only one I ahve
  PHOMEMO_T02: {
    MAX_WIDTH_BYTES: 0x30,
    MAX_BITMAP_HEIGHT: 0xFF
  }
};

const printImageServer = async () => {
  try {
    // Perform the POST request
    const dataToSend = JSON.stringify(convertImageToBitmapData());
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

document.getElementById("idButton").onclick = async() => {
  await printImageServer();
};

const FONT_HEIGHT = 20;
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

const fetchBatteryLevel = async () => {
  try {
    // Perform the POST request
    const response = await fetch("/api/printer/info", {method: "GET"});

    // Ensure the response is OK
    if (!response.ok) {
      console.error(`HTTP error! Status: ${response.status}`);
    }

    // Read the response as text
    const responseText = await response.text();

    if (response.ok) {
      return `<span style="color: green">Connected: Battery ${responseText}%</span>`;
    } else {
      return `<span style="color: red">Disconnected: ${responseText}</span>`;
    }
  } catch (error) {
    console.error("Error during POST request:", error);
    throw error; // Rethrow or handle appropriately
  }
};

const updateBatteryLevel = async () => {
  document.getElementById("battery-level").innerHTML = await fetchBatteryLevel();
};

const fileToBase64 = (file) => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result.split(",")[1]); // Extract Base64
    reader.onerror = (error) => reject(error);
  });
}
const printImageServer = async (imageData, imageType) => {
  try {
    const body = {
      "data": await fileToBase64(imageData),
      "contentType": imageType
    };
    console.log(JSON.stringify(body));
    const response = await fetch("/api/printer", {
      method: "POST",
      headers: {
        "Content-Type": imageType,
      },
      body: JSON.stringify(body)
    });

    // Ensure the response is OK
    if (!response.ok) {
      console.error(`HTTP error! Status: ${response.status}`);

      // Read the response as text
      const responseText = await response.text();

      alert(`HTTP ${response.status}: ${responseText}`);
    }
  } catch (error) {
    console.error("Error during POST request:", error);
    throw error; // Rethrow or handle appropriately
  }
};

const printCanvas = async () => {
  const imageType = 'image/png';
  const dataToSend = await new Promise(resolve => canvas.toBlob(blob => resolve(blob), imageType));
  await printImageServer(dataToSend, imageType);
};

document.getElementById("idButton").onclick = async() => {
  await printCanvas();
};

const batteryInterval = setInterval(updateBatteryLevel, 1000);

document.addEventListener("DOMContentLoaded", updateBatteryLevel);

const form = document.getElementById('uploadForm');
const fileInput = document.getElementById('fileInput');

form.addEventListener('submit', async (event) => {
  event.preventDefault(); // Prevent the default form submission

  const file = fileInput.files[0];
  if (!file) {
    alert('Please select a file');
    return;
  }

  await printImageServer(file, file.type);
});

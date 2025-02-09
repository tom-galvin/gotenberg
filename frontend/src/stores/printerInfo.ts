import { ref } from 'vue'
import { defineStore } from 'pinia'

export type PrinterInfo = {
  batteryLevel: number,
  firmwareVersion: string
};

export const usePrinterInfoStore = defineStore('printerInfo', () => {
  const info = ref<PrinterInfo>({ batteryLevel: 0, firmwareVersion: "0.0.0" })

  const setInfo = (batteryLevel: number, firmwareVersion: string) => {
    info.value = { batteryLevel, firmwareVersion }
  }

  return { info, setInfo };
})

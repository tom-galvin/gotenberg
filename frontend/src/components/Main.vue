<script setup lang="ts">
  import { getPrinterInfo } from "../api/sdk.gen";
  import { usePrinterInfoStore } from "../stores/printerInfo";
  import { client } from "../api";
  defineProps<{
    msg: string
  }>()

  const counterStore = usePrinterInfoStore();

  const fetchPrinterInfo = async () => {
    const e = await getPrinterInfo({ client });
    if (!e.error && e.data) {
      counterStore.setInfo(e.data.batteryLevel, e.data.firmwareVersion);
    } else {
      alert("Error! " + e.error);
    }

    alert(JSON.stringify(e));
  };
</script>

<template>
  <div class="greetings">
    <h1 class="green">{{ msg }}</h1>
    <h2 class="green">Firmware version: {{ counterStore.info.firmwareVersion }}</h2>
    <h2 class="green">Battery level: {{ counterStore.info.batteryLevel }}</h2>
    <h3>
      You’ve successfully created a project with
      <a href="https://vite.dev/" target="_blank" rel="noopener">Vite</a> +
      <a href="https://vuejs.org/" target="_blank" rel="noopener">Vue 3</a>. What's next?
      <a href="javascript:void(0)" @click="fetchPrinterInfo()">Test</a>
    </h3>
  </div>
</template>

<style scoped>
h1 {
  font-weight: 500;
  font-size: 2.6rem;
  position: relative;
  top: -10px;
}

h3 {
  font-size: 1.2rem;
}

.greetings h1,
.greetings h3 {
  text-align: center;
}

@media (min-width: 1024px) {
  .greetings h1,
  .greetings h3 {
    text-align: left;
  }
}
</style>

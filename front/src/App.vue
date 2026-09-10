<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import CameraPlayer from './components/CameraPlayer.vue';
import { loadCameras } from './models/camera.js';

const cameras = ref([]);
const loading = ref(true);
const error = ref('');
// При уходе со страницы отменяем загрузку каталога, не показывая ошибку пользователю.
const controller = new AbortController();

onMounted(async () => {
  try {
    cameras.value = await loadCameras(
      AbortSignal.any([controller.signal, AbortSignal.timeout(10000)]),
    );
  } catch (cause) {
    if (!controller.signal.aborted) {
      error.value = cause.message;
    }
  } finally {
    loading.value = false;
  }
});
onBeforeUnmount(() => controller.abort());
</script>

<template>
  <main>
    <p v-if="loading" role="status">Загрузка камер…</p>
    <p v-else-if="error" role="alert">{{ error }}</p>
    <div v-else-if="cameras.length" class="camera-grid">
      <CameraPlayer v-for="camera in cameras" :key="camera.id" :camera="camera" />
    </div>
    <p v-else role="status">Нет камер.</p>
  </main>
</template>

<style>
:root {
  color-scheme: light dark;
  font-family: system-ui, sans-serif;
}
body {
  max-width: 1100px;
  margin: auto;
  padding: 24px;
}
header {
  margin-bottom: 32px;
}
h1 {
  margin-bottom: 8px;
  font-size: 2rem;
}
header p {
  margin: 0;
  opacity: 0.7;
}
label {
  display: grid;
  gap: 6px;
  width: fit-content;
  margin-bottom: 20px;
}
select {
  font: inherit;
  padding: 10px 14px;
}
video {
  display: block;
  width: 100%;
  aspect-ratio: 16/9;
  background: #111;
  border-radius: 8px;
}
.camera-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 420px), 1fr));
  gap: 24px;
  align-items: start;
}
[role='status'] {
  min-height: 1.5em;
}
</style>

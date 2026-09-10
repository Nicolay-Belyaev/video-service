<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { openCameraStream } from '../services/webrtc.js';
import { retryConnection } from '../services/retryConnection.js';
import { selectStreamQuality } from '../services/streamQuality.js';

const props = defineProps({ camera: { type: Object, required: true } });
const video = ref(null);
const player = ref(null);
const fullscreen = ref(false);
const fullscreenError = ref('');
const quality = computed(() => selectStreamQuality(props.camera.qualities, fullscreen.value));
const started = ref(false);
const status = ref('');
// У каждого плеера свой цикл подключения и повторов.
let stopConnection;

function closeConnection() {
  stopConnection?.();
  stopConnection = undefined;
}

// Перед сменой потока освобождаем старую сессию и отменяем её ожидающие повторы.
function start() {
  closeConnection();
  if (!quality.value) {
    return;
  }
  started.value = true;
  stopConnection = retryConnection(
    (callbacks) => openCameraStream(props.camera, quality.value, video.value, callbacks),
    (message) => {
      status.value = message;
    },
    () => {
      started.value = false;
    },
  );
}

function stop() {
  closeConnection();
  started.value = false;
  status.value = '';
}

function syncFullscreen() {
  const element = document.fullscreenElement;
  // Учитываем нашу кнопку и штатное разворачивание видео средствами браузера.
  fullscreen.value = Boolean(element && player.value?.contains(element));
}

async function toggleFullscreen() {
  fullscreenError.value = '';
  try {
    if (fullscreen.value) {
      await document.exitFullscreen();
    } else {
      await player.value.requestFullscreen();
    }
  } catch {
    fullscreenError.value = 'Не удалось изменить полноэкранный режим.';
  }
}

// Смена режима переключает только уже запущенную пользователем камеру.
watch(
  () => quality.value?.id,
  () => {
    if (started.value) {
      start();
    }
  },
);
onMounted(() => {
  document.addEventListener('fullscreenchange', syncFullscreen);
  window.addEventListener('pagehide', stop);
});
onBeforeUnmount(() => {
  stop();
  document.removeEventListener('fullscreenchange', syncFullscreen);
  window.removeEventListener('pagehide', stop);
});
</script>

<template>
  <section :aria-label="camera.name">
    <h2>{{ camera.name }}</h2>
    <div ref="player" class="player">
      <video
        ref="video"
        autoplay
        muted
        playsinline
        :controls="started"
        :aria-label="camera.name"
      ></video>
      <button
        v-if="!started"
        class="play-overlay"
        type="button"
        :disabled="!quality"
        :aria-label="`Смотреть ${camera.name}`"
        @click="start"
      >
        <span aria-hidden="true">▶</span>
        {{ quality ? 'Нажмите для просмотра' : 'Нет доступного потока' }}
      </button>
      <button
        v-if="started"
        class="fullscreen-button"
        type="button"
        :aria-label="fullscreen ? `Свернуть ${camera.name}` : `Развернуть ${camera.name}`"
        @click="toggleFullscreen"
      >
        {{ fullscreen ? 'Выйти из полного экрана' : 'На весь экран' }}
      </button>
    </div>
    <p v-if="fullscreenError" role="alert">{{ fullscreenError }}</p>
    <p role="status" aria-live="polite">{{ status }}</p>
  </section>
</template>

<style scoped>
h2 {
  margin: 0 0 16px;
  font-size: 1.2rem;
}
section {
  min-width: 0;
}

.player {
  position: relative;
}
.player:fullscreen {
  display: grid;
  place-items: center;
  background: #111;
}
.player:fullscreen video {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.fullscreen-button {
  position: absolute;
  top: 12px;
  right: 12px;
  padding: 8px 12px;
  border: 1px solid #777;
  border-radius: 6px;
  background: #111c;
  color: white;
  font: inherit;
  cursor: pointer;
}
.play-overlay {
  position: absolute;
  inset: 0;
  width: 100%;
  border: 0;
  border-radius: 8px;
  background: #111;
  color: #eee;
  font: inherit;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  cursor: pointer;
}
.play-overlay span {
  font-size: 2rem;
}
.play-overlay:hover:not(:disabled) {
  background: #202020;
}
.play-overlay:focus-visible {
  outline: 3px solid #6baaff;
  outline-offset: 3px;
}
.play-overlay:disabled {
  cursor: default;
  color: #aaa;
}
</style>

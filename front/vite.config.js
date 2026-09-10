import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// В разработке и при просмотре сборки направляем API-запросы в Go-сервис.
const proxy = { '/cameras': 'http://127.0.0.1:80' };

export default defineConfig({
  plugins: [vue()],
  server: { proxy },
  preview: { proxy },
});

/**
 * @typedef {{ id: string, label: string, available: boolean }} StreamQuality
 * @typedef {{ id: number, name: string, available: boolean, qualities: StreamQuality[] }} Camera
 * available — результат проверки при запуске сервера, а не текущее состояние камеры.
 */

/** @returns {Promise<Camera[]>} */
export async function loadCameras(signal) {
  const response = await fetch('/cameras', { signal });
  if (!response.ok) throw new Error(`Ошибка загрузки камер: HTTP ${response.status}.`);
  const catalog = await response.json();
  return catalog.cameras;
}

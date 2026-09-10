/** Выбираем качество по ID из конфига. Камера с одним потоком продолжает использовать его. */
export function selectStreamQuality(qualities, fullscreen) {
  const preferredId = fullscreen ? 'high' : 'low';
  return (
    qualities.find((quality) => quality.id === preferredId) ??
    qualities.find((quality) => quality.id === 'low') ??
    qualities[0]
  );
}

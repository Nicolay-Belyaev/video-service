export const MAX_RETRIES = 3;
export const RETRY_DELAY_MS = 3000;

/** Первая попытка подключения и не более трёх повторов после ошибок. */
export function retryConnection(openConnection, onStatus, onExhausted = () => {}) {
  let stopped = false;
  let retries = 0;
  let timer;
  let closeConnection;

  function attempt() {
    if (stopped) {
      return;
    }
    onStatus('Подключение…');
    let failed = false;

    closeConnection = openConnection({
      connected() {
        if (stopped || failed) {
          return;
        }
        // После успешного подключения следующий обрыв получает новый лимит повторов.
        retries = 0;
        onStatus('Прямой эфир');
      },
      failed(error) {
        if (stopped || failed) {
          return;
        }
        failed = true;
        closeConnection?.();
        if (retries === MAX_RETRIES) {
          onStatus(
            `Не удалось подключиться после ${MAX_RETRIES} повторных попыток. ${error.message}`,
          );
          onExhausted();
          return;
        }
        retries += 1;
        onStatus(`Повторная попытка ${retries}/${MAX_RETRIES} через 3 секунды. ${error.message}`);
        timer = setTimeout(attempt, RETRY_DELAY_MS);
      },
    });
    // Ошибка может прийти ещё до возврата функции очистки из openConnection.
    if (failed) {
      closeConnection();
    }
  }

  attempt();
  // Остановка плеера должна отменить и соединение, и ещё не запущенный повтор.
  return () => {
    stopped = true;
    clearTimeout(timer);
    closeConnection?.();
  };
}

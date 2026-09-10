const CONNECTION_TIMEOUT_MS = 30000;

// Сервер принимает полный SDP одним запросом: сначала дожидаемся сбора ICE-кандидатов.
function waitForICE(peer, signal) {
  return new Promise((resolve, reject) => {
    function cleanup() {
      peer.removeEventListener('icegatheringstatechange', check);
      signal.removeEventListener('abort', cancel);
    }
    function check() {
      if (peer.iceGatheringState !== 'complete') {
        return;
      }
      cleanup();
      resolve();
    }
    function cancel() {
      cleanup();
      reject(new DOMException('Отменено', 'AbortError'));
    }
    peer.addEventListener('icegatheringstatechange', check);
    signal.addEventListener('abort', cancel, { once: true });
    if (signal.aborted) {
      cancel();
    } else {
      check();
    }
  });
}

/** Открывает одну WebRTC-сессию и возвращает функцию её закрытия. Повторы выполняет вызывающий код. */
export function openCameraStream(camera, quality, video, { connected, failed }) {
  const controller = new AbortController();
  let peer;
  let closed = false;
  const timeout = setTimeout(
    () => fail(new Error('Истекло время подключения.')),
    CONNECTION_TIMEOUT_MS,
  );

  // Очистка может вызываться из таймаута, ошибки и компонента — выполняем её один раз.
  function close() {
    if (closed) {
      return;
    }
    closed = true;
    clearTimeout(timeout);
    controller.abort();
    peer?.close();
    video.srcObject = null;
  }

  function fail(error) {
    if (closed) {
      return;
    }
    close();
    failed(error);
  }

  async function connect() {
    if (closed) {
      return;
    }
    try {
      peer = new RTCPeerConnection();
      peer.ontrack = ({ track }) => {
        if (!closed) {
          video.srcObject = new MediaStream([track]);
        }
      };
      peer.onconnectionstatechange = () => {
        if (closed) {
          return;
        }
        if (peer.connectionState === 'connected') {
          clearTimeout(timeout);
          connected();
        } else if (['failed', 'disconnected', 'closed'].includes(peer.connectionState)) {
          fail(new Error('Соединение потеряно.'));
        }
      };
      // Только принимаем видео: доступ к камере и микрофону пользователя не нужен.
      peer.addTransceiver('video', { direction: 'recvonly' });
      await peer.setLocalDescription(await peer.createOffer());
      await waitForICE(peer, controller.signal);
      const response = await fetch(
        `/cameras/${encodeURIComponent(camera.id)}/webrtc/${encodeURIComponent(quality.id)}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(peer.localDescription),
          signal: controller.signal,
        },
      );
      if (!response.ok) throw new Error(`Ошибка сервиса: HTTP ${response.status}.`);
      const answer = await response.json();
      if (!closed) {
        await peer.setRemoteDescription(answer);
      }
    } catch (error) {
      fail(error);
    }
  }

  // Откладываем подключение, чтобы вызывающий код успел получить функцию закрытия.
  queueMicrotask(connect);
  return close;
}

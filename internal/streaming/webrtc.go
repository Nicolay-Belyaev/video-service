package streaming

import (
	"errors"
	"sync"

	"github.com/pion/webrtc/v4"
)

func StartWebRTC(stream *Stream, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	sender, err := stream.AddPeer(peer)
	if err != nil {
		peer.Close()
		return webrtc.SessionDescription{}, err
	}

	// Ошибка сигнализации, остановка RTSP и событие WebRTC могут совпасть.
	// Удаляем зрителя и освобождаем соединение ровно один раз.
	var cleanupOnce sync.Once
	peerDone := make(chan struct{})
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(peerDone)
			stream.RemovePeer(peer)
			_ = peer.Close()
		})
	}

	// Регистрируем обработчик до начала согласования соединения.
	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			// Close может вызвать новое событие: не блокируем callback очисткой.
			go cleanup()
		}
	})

	go func() {
		select {
		case <-stream.Done():
			// Закрытие peer также разблокирует чтение RTCP ниже.
			cleanup()
		case <-peerDone:
			// Зритель ушёл раньше источника; наблюдатель больше не нужен.
		}
	}()

	// Читаем RTCP от браузера.
	go func() {
		buf := make([]byte, 1500)

		for {
			if _, _, err := sender.Read(buf); err != nil {
				return
			}
		}
	}()

	if err := peer.SetRemoteDescription(offer); err != nil {
		cleanup()
		return webrtc.SessionDescription{}, err
	}

	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		cleanup()
		return webrtc.SessionDescription{}, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(peer)

	if err := peer.SetLocalDescription(answer); err != nil {
		cleanup()
		return webrtc.SessionDescription{}, err
	}

	// Не ждём ICE после завершения источника или соединения.
	select {
	case <-gatherComplete:
	case <-stream.Done():
		cleanup()
		return webrtc.SessionDescription{}, errors.New("stream stopped")
	case <-peerDone:
		return webrtc.SessionDescription{}, errors.New("peer closed")
	}

	// Если одновременно готовы несколько каналов, select мог выбрать ICE.
	select {
	case <-stream.Done():
		cleanup()
		return webrtc.SessionDescription{}, errors.New("stream stopped")
	case <-peerDone:
		return webrtc.SessionDescription{}, errors.New("peer closed")
	default:
	}

	return *peer.LocalDescription(), nil
}

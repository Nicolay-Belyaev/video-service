package streaming

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	"video-service/internal/domain"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type streamKey struct {
	CameraID int
	Quality  domain.Quality
}

type StreamManager struct {
	mu      sync.Mutex
	ctx     context.Context
	streams map[streamKey]*Stream
	timeout time.Duration
}

func NewStreamManager(ctx context.Context, timeout time.Duration) *StreamManager {
	return &StreamManager{
		ctx:     ctx,
		streams: make(map[streamKey]*Stream),
		timeout: timeout,
	}
}

func (s *StreamManager) GetByCamAndQuality(camera domain.Camera, quality domain.Quality) (*Stream, error) {
	key := streamKey{
		CameraID: camera.ID,
		Quality:  quality,
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	str, ok := s.streams[key]
	if ok {
		select {
		case <-str.Done():
			// Удаляющая горутина могла ещё не получить блокировку.
			delete(s.streams, key)
		default:
			return str, nil
		}
	}

	stream, err := NewStream(s.ctx, camera, quality, s.timeout)
	if err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}

	s.streams[key] = stream

	go func() {
		<-stream.Done()

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.streams[key] == stream {
			delete(s.streams, key)
		}
	}()

	return stream, nil
}

type Stream struct {
	// Пара Camera и Quality задают физический RTSP-источник для Stream.
	Camera  domain.Camera
	Quality domain.Quality

	// Останавливает ReadH264 и закрывает RTSP-поток
	cancel context.CancelFunc
	done   <-chan struct{}

	// таймер остановки Stream запускаемый когда количество зрителей становится равно нулю.
	// Нужен, чтобы держать живое RTSP-соединение после отключения последнего зрителя некоторое время.
	// Таким образом, если единственный зритель обновляет страницу, он получит тот же Stream.
	offTimer *time.Timer
	timeout  time.Duration

	Track *webrtc.TrackLocalStaticRTP

	// Зрители добавляются и удаляются конкурентно
	mu    sync.Mutex
	peers map[*webrtc.PeerConnection]struct{}
}

func NewStream(ctx context.Context, camera domain.Camera, quality domain.Quality, timeout time.Duration) (*Stream, error) {
	source, ok := camera.Stream(quality)
	if !ok {
		return nil, errors.New("invalid quality")
	}

	streamContext, cnsl := context.WithCancel(ctx)

	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: 90000,
		},
		"video",
		"camera",
	)
	if err != nil {
		cnsl()
		return nil, err
	}

	s := &Stream{
		Camera:  camera,
		Quality: quality,
		Track:   track,
		timeout: timeout,
		cancel:  cnsl,
		done:    streamContext.Done(),
		peers:   make(map[*webrtc.PeerConnection]struct{}),
	}

	go func() {
		defer cnsl()

		err := ReadH264(streamContext, source.URL, func(packet *rtp.Packet) {
			if err := track.WriteRTP(packet); err != nil {
				log.Printf("can't write RTP into track: %v", err)
			}
		})
		if err != nil {
			log.Printf("camera %d stream %s: RTSP read failed", camera.ID, quality)
		}
	}()

	return s, nil
}

func (s *Stream) AddPeer(peer *webrtc.PeerConnection) (*webrtc.RTPSender, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		return nil, errors.New("stream closed")
	default:
	}

	if _, ok := s.peers[peer]; ok {
		return nil, errors.New("peer already added")
	}

	// Отдаем новому зрителю наш трек
	sender, err := peer.AddTrack(s.Track)
	if err != nil {
		return nil, err
	}

	// Отменяем остановку только после успешного добавления трека:
	// ошибка AddTrack не должна оставлять поток без зрителей и таймера.
	if s.offTimer != nil {
		s.offTimer.Stop()
		s.offTimer = nil
	}

	s.peers[peer] = struct{}{}
	return sender, nil
}

// RemovePeer не делает peer.Close(). Управление жизненным циклом зрителей является ответственностью вызывающего кода.
func (s *Stream) RemovePeer(peer *webrtc.PeerConnection) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.peers[peer]; !ok {
		log.Printf("Remove peer: attempting to remove non-existing peer %v", peer)
		return
	}
	delete(s.peers, peer)

	if len(s.peers) != 0 {
		return
	}

	select {
	case <-s.done:
		return
	default:
	}

	// Когда все зрители отключаются от потока Stream ждет еще timeout секунд и умирает.
	var timer *time.Timer
	timer = time.AfterFunc(s.timeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Stop не отменяет уже запущенный callback, ожидающий s.mu.
		// Старый таймер не должен останавливать поток после смены зрителей.
		if s.offTimer != timer {
			return
		}

		s.offTimer = nil
		if len(s.peers) == 0 {
			// Проверка и отмена атомарны относительно AddPeer.
			s.cancel()
		}
	})
	s.offTimer = timer

}

func (s *Stream) Done() <-chan struct{} {
	return s.done
}

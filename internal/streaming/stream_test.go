package streaming

import (
	"context"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

// Создаём поток без RTSP-камеры, чтобы проверять только жизненный цикл.
func newLifecycleTestStream(t *testing.T) *Stream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
		"video", "camera",
	)
	if err != nil {
		t.Fatal(err)
	}
	return &Stream{
		Track:  track,
		cancel: cancel,
		done:   ctx.Done(),
		peers:  make(map[*webrtc.PeerConnection]struct{}),
	}
}

func TestAddPeerRejectsStoppedStream(t *testing.T) {
	s := newLifecycleTestStream(t)
	s.cancel()
	// Проверка остановки должна произойти до обращения к peer.
	if _, err := s.AddPeer(nil); err == nil {
		t.Fatal("expected stopped stream error")
	}
}

func TestFailedAddPeerPreservesIdleTimer(t *testing.T) {
	s := newLifecycleTestStream(t)
	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
	timer := time.AfterFunc(time.Hour, func() {})
	defer timer.Stop()
	s.offTimer = timer

	if _, err := s.AddPeer(peer); err == nil {
		t.Fatal("expected AddTrack to reject a closed peer")
	}
	if s.offTimer != timer {
		t.Fatal("failed connection discarded the idle timer")
	}
	if !timer.Stop() {
		t.Fatal("failed connection stopped the idle timer")
	}
}

func TestInvalidOfferRemovesPeerAndStopsIdleStream(t *testing.T) {
	s := newLifecycleTestStream(t)
	_, err := StartWebRTC(s, webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP: "invalid SDP",
	})
	if err == nil {
		t.Fatal("expected invalid offer error")
	}
	s.mu.Lock()
	count := len(s.peers)
	s.mu.Unlock()
	if count != 0 {
		t.Fatalf("failed signaling left %d peers", count)
	}
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("empty stream did not stop after failed signaling")
	}
}

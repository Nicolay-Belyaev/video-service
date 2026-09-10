package streaming

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"
	"video-service/internal/domain"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
)

// Probe checks authentication, DESCRIBE and H264 support, without starting playback.
func Probe(ctx context.Context, streamURL string) error {
	u, err := base.ParseURL(streamURL)
	if err != nil {
		return errors.New("invalid RTSP URL")
	}
	client := &gortsplib.Client{
		Scheme: u.Scheme,
		Host:   u.Host,
		DialContext: func(_ context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, address)
		},
	}
	if err := client.Start(); err != nil {
		return errors.New("RTSP client start failed")
	}
	defer client.Close()
	stopCancellation := context.AfterFunc(ctx, client.Close)
	defer stopCancellation()
	desc, _, err := client.Describe(u)
	if err != nil {
		return errors.New("RTSP describe failed")
	}
	var h264 *format.H264
	if desc.FindFormat(&h264) == nil {
		return errors.New("H264 video not found")
	}
	return ctx.Err()
}

// CheckCameras runs before HTTP serving starts, so availability needs no shared-state lock.
func CheckCameras(ctx context.Context, cameras map[int]*domain.Camera, timeout time.Duration) {
	var pending sync.WaitGroup
	slots := make(chan struct{}, 4)
	for _, camera := range cameras {
		for i := range camera.Streams {
			slots <- struct{}{}
			pending.Add(1)
			go func(camera *domain.Camera, index int) {
				defer pending.Done()
				defer func() { <-slots }()
				probeContext, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()
				stream := &camera.Streams[index]
				err := Probe(probeContext, stream.URL)
				stream.Available = err == nil
				// Only safe identifiers and a boolean: library errors may contain credentials.
				log.Printf("camera %d stream %s: startup available=%t", camera.ID, stream.ID, stream.Available)
			}(camera, i)
		}
	}
	pending.Wait()
}

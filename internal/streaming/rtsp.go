package streaming

import (
	"context"
	"errors"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
)

import (
	"github.com/pion/rtp"
)

func ReadH264(ctx context.Context, streamURL string, onPacket func(*rtp.Packet)) error {
	// Парсим RTSP-ссылку на камеру
	u, err := base.ParseURL(streamURL)
	if err != nil {
		return err
	}

	// <<< Запускаем клиент и проверяем, что камера внутри RTSP отдает H.264
	client := &gortsplib.Client{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	if err = client.Start(); err != nil {
		return err
	}
	defer client.Close()

	desc, _, err := client.Describe(u)
	if err != nil {
		return err
	}

	var h264Format *format.H264
	media := desc.FindFormat(&h264Format)
	if media == nil {
		return errors.New("unable to find h264 format")
	}

	// Просим камеру отдать выбранный поток
	_, err = client.Setup(desc.BaseURL, media, 0, 0)
	if err != nil {
		return err
	}

	// Callback'ом вызываем функцию, которая будет обрабатывать поступающие RTP-пакеты и запускаем Play у клиента
	// Только после Play начинает идти реальный трафик.
	client.OnPacketRTP(media, h264Format, func(packet *rtp.Packet) {
		onPacket(packet)
	})

	_, err = client.Play(nil)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- client.Wait()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

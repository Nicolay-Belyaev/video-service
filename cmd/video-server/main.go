package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"
	"video-service/internal"
	"video-service/internal/config"
	"video-service/internal/streaming"
)

func main() {
	configPath := flag.String("config", "cameras.json", "path to private camera JSON config")
	listen := flag.String("listen", ":80", "HTTP listen address")
	probeTimeout := flag.Duration("probe-timeout", 5*time.Second, "timeout per startup RTSP probe")
	flag.Parse()
	if *probeTimeout <= 0 {
		log.Fatal("probe-timeout must be positive")
	}
	cameras, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	streaming.CheckCameras(context.Background(), cameras, *probeTimeout)

	API := internal.CameraAPI{
		Cameras: cameras,
		// Держим источник 10 секунд после ухода последнего зрителя.
		StreamManager: streaming.NewStreamManager(context.Background(), 10*time.Second),
	}

	http.HandleFunc("POST /cameras/{id}/webrtc/{quality}", API.WebRTC)
	http.HandleFunc("GET /cameras", API.List)
	http.Handle("GET /", http.FileServer(http.Dir("front/dist")))

	log.Printf("listening on %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

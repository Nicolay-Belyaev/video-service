package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"video-service/internal/domain"
	"video-service/internal/streaming"

	"github.com/pion/webrtc/v4"
)

type CameraAPI struct {
	Cameras       map[int]*domain.Camera
	StreamManager *streaming.StreamManager
}

type qualityInfo struct {
	ID        domain.Quality `json:"id"`
	Label     string         `json:"label"`
	Available bool           `json:"available"`
}
type cameraInfo struct {
	ID        int           `json:"id"`
	Name      string        `json:"name"`
	Available bool          `json:"available"`
	Qualities []qualityInfo `json:"qualities"`
}

func (c *CameraAPI) WebRTC(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid camera id", http.StatusBadRequest)
		return
	}
	camera, ok := c.Cameras[id]
	if !ok {
		http.Error(w, "camera not found", http.StatusNotFound)
		return
	}

	quality := domain.Quality(r.PathValue("quality"))
	if _, ok := camera.Stream(quality); !ok {
		http.Error(w, "invalid quality", http.StatusBadRequest)
		return
	}
	// Startup availability is informational: always allow a new connection attempt.

	var offer webrtc.SessionDescription

	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Все зрители одной камеры и качества используют общий RTSP-поток.
	stream, err := c.StreamManager.GetByCamAndQuality(*camera, quality)
	if err != nil {
		http.Error(w, "unable to start camera stream", http.StatusBadGateway)
		return
	}

	answer, err := streaming.StartWebRTC(stream, offer)
	if err != nil {
		http.Error(w, "unable to start camera stream", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(answer); err != nil {
		log.Printf("encode WebRTC answer: %v", err)
	}
}

func (c *CameraAPI) List(w http.ResponseWriter, r *http.Request) {
	cameras := make([]cameraInfo, 0, len(c.Cameras))
	for _, camera := range c.Cameras {
		info := cameraInfo{ID: camera.ID, Name: camera.Name, Qualities: make([]qualityInfo, 0, len(camera.Streams))}
		for _, stream := range camera.Streams {
			info.Qualities = append(info.Qualities, qualityInfo{ID: stream.ID, Label: stream.Label, Available: stream.Available})
			info.Available = info.Available || stream.Available
		}
		cameras = append(cameras, info)
	}
	sort.Slice(cameras, func(i, j int) bool { return cameras[i].ID < cameras[j].ID })
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(struct {
		Cameras []cameraInfo `json:"cameras"`
	}{cameras}); err != nil {
		log.Printf("encode cameras: %v", err)
	}
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"video-service/internal/domain"
)

var streamID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func Load(path string) (map[int]*domain.Camera, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open camera config")
	}
	defer file.Close()
	return Decode(file)
}

func Decode(reader io.Reader) (map[int]*domain.Camera, error) {
	var document struct {
		Cameras []domain.Camera `json:"cameras"`
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		// JSON errors may contain user-supplied field names or values, including secrets.
		return nil, errors.New("invalid camera config JSON or unknown fields")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("camera config must contain one JSON document")
	}
	if document.Cameras == nil {
		return nil, errors.New("cameras must be an array")
	}
	cameras := make(map[int]*domain.Camera, len(document.Cameras))
	for i := range document.Cameras {
		camera := &document.Cameras[i]
		if camera.ID <= 0 {
			return nil, fmt.Errorf("camera at index %d: id must be positive", i)
		}
		if _, exists := cameras[camera.ID]; exists {
			return nil, fmt.Errorf("duplicate camera id %d", camera.ID)
		}
		if strings.TrimSpace(camera.Name) == "" || len(camera.Streams) == 0 {
			return nil, fmt.Errorf("camera %d: name and at least one stream are required", camera.ID)
		}
		seen := make(map[domain.Quality]bool)
		for j, stream := range camera.Streams {
			if !streamID.MatchString(string(stream.ID)) || seen[stream.ID] || strings.TrimSpace(stream.Label) == "" {
				return nil, fmt.Errorf("camera %d stream %d: unique URL-safe id and label are required", camera.ID, j)
			}
			seen[stream.ID] = true
			u, err := url.Parse(stream.URL)
			if err != nil || u == nil || (u.Scheme != "rtsp" && u.Scheme != "rtsps") || u.Hostname() == "" || u.Fragment != "" {
				return nil, fmt.Errorf("camera %d stream %d: invalid RTSP URL", camera.ID, j)
			}
			if u.Port() != "" {
				port, err := strconv.Atoi(u.Port())
				if err != nil || port < 1 || port > 65535 {
					return nil, fmt.Errorf("camera %d stream %d: invalid RTSP port", camera.ID, j)
				}
			}
		}
		cameras[camera.ID] = camera
	}
	return cameras, nil
}

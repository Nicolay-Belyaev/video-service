package internal

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"video-service/internal/domain"
)

func TestPublicCatalog(t *testing.T) {
	api := CameraAPI{Cameras: map[int]*domain.Camera{
		2: {ID: 2, Name: "Back", Streams: []domain.CameraStream{{ID: "medium", Label: "1080p", URL: "rtsp://user:SECRET@private-host/video"}}},
		1: {ID: 1, Name: "Front", Streams: []domain.CameraStream{{ID: "low", Label: "720p", Available: true}, {ID: "high", Label: "4K"}}},
	}}
	response := httptest.NewRecorder()
	api.List(response, httptest.NewRequest("GET", "/cameras", nil))
	for _, secret := range []string{"SECRET", "rtsp", "private-host", `"url"`} {
		if strings.Contains(response.Body.String(), secret) {
			t.Fatalf("catalog leaked %s", secret)
		}
	}
	var result struct {
		Cameras []struct {
			ID        int
			Available bool
			Qualities []struct {
				ID        string
				Available bool
			}
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Cameras) != 2 || result.Cameras[0].ID != 1 || !result.Cameras[0].Available || result.Cameras[1].Available {
		t.Fatal("invalid camera catalog")
	}
	if result.Cameras[0].Qualities[1].Available || result.Cameras[1].Qualities[0].ID != "medium" {
		t.Fatal("invalid quality catalog")
	}
}

func TestEmptyCatalog(t *testing.T) {
	api := CameraAPI{}
	response := httptest.NewRecorder()
	api.List(response, httptest.NewRequest("GET", "/cameras", nil))
	if strings.TrimSpace(response.Body.String()) != `{"cameras":[]}` {
		t.Fatal(response.Body.String())
	}
}

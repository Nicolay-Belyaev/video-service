package config

import (
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	valid := `{"cameras":[{"id":1,"name":"Entrance","streams":[{"id":"medium","label":"1080p","url":"rtsp://user:SECRET@localhost/video"}]}]}`
	cameras, err := Decode(strings.NewReader(valid))
	if err != nil {
		t.Fatal(err)
	}
	if cameras[1].Streams[0].ID != "medium" || cameras[1].Streams[0].Available {
		t.Fatal("invalid parsed stream")
	}
	for name, input := range map[string]string{
		"duplicate camera": strings.Replace(valid, `}]}`, `},{"id":1,"name":"Duplicate","streams":[]}]}`, 1),
		"invalid port":     strings.Replace(valid, "localhost/video", "localhost:99999/video", 1),
		"invalid URL":      strings.Replace(valid, "rtsp://", "https://", 1),
		"invalid id":       strings.Replace(valid, `"medium"`, `"bad/id"`, 1),
		"missing name":     strings.Replace(valid, `"Entrance"`, `""`, 1),
		"unknown field":    strings.Replace(valid, `"label"`, `"SECRET"`, 1),
		"trailing JSON":    valid + `{}`,
		"missing cameras":  `{}`,
		"null":             `null`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(input))
			if err == nil {
				t.Fatal("expected validation error")
			}
			if strings.Contains(err.Error(), "SECRET") {
				t.Fatal("error leaked secret")
			}
		})
	}
	if _, err := Decode(strings.NewReader(`{"cameras":[]}`)); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateStream(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"cameras":[{"id":1,"name":"Test","streams":[{"id":"low","label":"A","url":"rtsp://localhost/a"},{"id":"low","label":"B","url":"rtsp://localhost/b"}]}]}`))
	if err == nil {
		t.Fatal("duplicate stream accepted")
	}
}

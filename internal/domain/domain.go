package domain

// Camera contains private connection settings. Never serialize it into an API response.
type Camera struct {
	ID      int            `json:"id"`
	Name    string         `json:"name"`
	Streams []CameraStream `json:"streams"`
}

type CameraStream struct {
	ID        Quality `json:"id"`
	Label     string  `json:"label"`
	URL       string  `json:"url"`
	Available bool    `json:"-"` // Result of the startup probe; not a live health guarantee.
}

type Quality string

func (c Camera) Stream(quality Quality) (CameraStream, bool) {
	for _, stream := range c.Streams {
		if stream.ID == quality {
			return stream, true
		}
	}
	return CameraStream{}, false
}

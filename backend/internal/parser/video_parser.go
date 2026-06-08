package parser

import "fmt"

type FrameInfo struct {
	FrameIndex int     `json:"frame_index"`
	Timestamp  float64 `json:"timestamp"`
	ObjectKey  string  `json:"object_key"`
}

type VideoParser struct{}

func NewVideoParser() *VideoParser {
	return &VideoParser{}
}

func (p *VideoParser) ParseVideo(videoPath string, sceneID string) ([]FrameInfo, error) {
	frameRate := 30.0
	duration := 10.0
	frameCount := int(duration * frameRate)
	frames := make([]FrameInfo, frameCount)

	for i := 0; i < frameCount; i++ {
		frames[i] = FrameInfo{
			FrameIndex: i,
			Timestamp:  float64(i) / frameRate,
			ObjectKey:  fmt.Sprintf("scenes/%s/frames/frame_%06d.jpg", sceneID, i),
		}
	}
	return frames, nil
}

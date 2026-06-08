package model

import "time"

type SceneStatus string

const (
	SceneStatusUploading SceneStatus = "uploading"
	SceneStatusParsing   SceneStatus = "parsing"
	SceneStatusReady     SceneStatus = "ready"
	SceneStatusAnnotated SceneStatus = "annotated"
	SceneStatusExported  SceneStatus = "exported"
)

type SceneType string

const (
	SceneTypeHighway SceneType = "highway"
	SceneTypeUrban   SceneType = "urban"
	SceneTypeRural   SceneType = "rural"
	SceneTypeParking SceneType = "parking"
)

type Scene struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	VideoPath   string      `json:"video_path"`
	CANLogPath  string      `json:"can_log_path"`
	Duration    float64     `json:"duration"`
	FrameCount  int         `json:"frame_count"`
	FrameRate   float64     `json:"frame_rate"`
	Status      SceneStatus `json:"status"`
	SceneType   SceneType   `json:"scene_type"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Tags        []string    `json:"tags"`
}

type SceneFilter struct {
	SceneType SceneType `form:"scene_type"`
	Status    SceneStatus `form:"status"`
	Tag       string    `form:"tag"`
	Page      int       `form:"page"`
	PageSize  int       `form:"page_size"`
}

type SceneStats struct {
	TotalCount           int                `json:"total_count"`
	AnnotationCompletion float64            `json:"annotation_completion"`
	SceneTypeDistribution map[string]int     `json:"scene_type_distribution"`
}

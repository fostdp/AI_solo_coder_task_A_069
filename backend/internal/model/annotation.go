package model

import "time"

type BoundingBoxLabel string

const (
	LabelVehicle    BoundingBoxLabel = "vehicle"
	LabelPedestrian BoundingBoxLabel = "pedestrian"
	LabelBicycle    BoundingBoxLabel = "bicycle"
	LabelOther      BoundingBoxLabel = "other"
)

type BoundingBox struct {
	X         float64               `json:"x" bson:"x"`
	Y         float64               `json:"y" bson:"y"`
	Width     float64               `json:"width" bson:"width"`
	Height    float64               `json:"height" bson:"height"`
	Label     BoundingBoxLabel      `json:"label" bson:"label"`
	TrackID   string                `json:"track_id" bson:"track_id"`
	Attributes map[string]string    `json:"attributes" bson:"attributes"`
}

type Annotation struct {
	ID         string        `json:"id" bson:"_id"`
	SceneID    string        `json:"scene_id" bson:"scene_id"`
	FrameIndex int           `json:"frame_index" bson:"frame_index"`
	Timestamp  float64       `json:"timestamp" bson:"timestamp"`
	Boxes      []BoundingBox `json:"boxes" bson:"boxes"`
	CreatedBy  string        `json:"created_by" bson:"created_by"`
	CreatedAt  time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at" bson:"updated_at"`
}

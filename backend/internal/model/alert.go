package model

import "time"

type AlertType string

const (
	AlertTypeTimeSync         AlertType = "time_sync"
	AlertTypeAnnotationQuality AlertType = "annotation_quality"
)

type AlertSeverity string

const (
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

type Alert struct {
	ID        string                 `json:"id"`
	SceneID   string                 `json:"scene_id"`
	Type      AlertType              `json:"type"`
	Severity  AlertSeverity          `json:"severity"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Resolved  bool                   `json:"resolved"`
	CreatedAt time.Time              `json:"created_at"`
}

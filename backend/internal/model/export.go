package model

import "time"

type ExportFormat string

const (
	ExportFormatOpenSCENARIO ExportFormat = "openscenario"
	ExportFormatROSBag       ExportFormat = "rosbag"
)

type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
)

type ExportTask struct {
	ID          string       `json:"id"`
	SceneID     string       `json:"scene_id"`
	Format      ExportFormat `json:"format"`
	Status      ExportStatus `json:"status"`
	FilePath    string       `json:"file_path"`
	CreatedAt   time.Time    `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at"`
}

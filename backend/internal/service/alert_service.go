package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/storage"

	"github.com/google/uuid"
)

type AlertService struct {
	pg *storage.PostgresStore
}

func NewAlertService(pg *storage.PostgresStore) *AlertService {
	return &AlertService{pg: pg}
}

func (s *AlertService) CheckTimeSyncAlert(ctx context.Context, sceneID string, frameTimestamps []float64, canSignals []model.CANSignal) (*model.Alert, error) {
	if len(frameTimestamps) == 0 || len(canSignals) == 0 {
		return nil, nil
	}

	var maxDrift float64
	canIdx := 0
	for _, ft := range frameTimestamps {
		for canIdx < len(canSignals)-1 && canSignals[canIdx+1].Timestamp < ft {
			canIdx++
		}
		if canIdx < len(canSignals) {
			drift := math.Abs(ft - canSignals[canIdx].Timestamp)
			if drift > maxDrift {
				maxDrift = drift
			}
		}
	}

	if maxDrift > 0.05 {
		severity := model.AlertSeverityWarning
		if maxDrift > 0.2 {
			severity = model.AlertSeverityCritical
		}
		alert := &model.Alert{
			ID:        uuid.New().String(),
			SceneID:   sceneID,
			Type:      model.AlertTypeTimeSync,
			Severity:  severity,
			Message:   fmt.Sprintf("Time sync drift detected: %.3fs (threshold: 0.05s)", maxDrift),
			Details:   map[string]interface{}{"max_drift": maxDrift, "threshold": 0.05},
			Resolved:  false,
			CreatedAt: time.Now(),
		}
		if err := s.pg.CreateAlert(ctx, alert); err != nil {
			return nil, err
		}
		return alert, nil
	}
	return nil, nil
}

func (s *AlertService) CheckAnnotationQuality(ctx context.Context, annotation *model.Annotation) (*model.Alert, error) {
	minArea := 100.0
	for _, box := range annotation.Boxes {
		area := box.Width * box.Height
		if area < minArea {
			alert := &model.Alert{
				ID:        uuid.New().String(),
				SceneID:   annotation.SceneID,
				Type:      model.AlertTypeAnnotationQuality,
				Severity:  model.AlertSeverityWarning,
				Message:   fmt.Sprintf("Bounding box area %.1f is below threshold %.1f", area, minArea),
				Details:   map[string]interface{}{"annotation_id": annotation.ID, "box_area": area, "threshold": minArea, "frame_index": annotation.FrameIndex},
				Resolved:  false,
				CreatedAt: time.Now(),
			}
			if err := s.pg.CreateAlert(ctx, alert); err != nil {
				return nil, err
			}
			return alert, nil
		}
	}
	return nil, nil
}

func (s *AlertService) GetAlerts(ctx context.Context, sceneID string) ([]*model.Alert, error) {
	if sceneID != "" {
		return s.pg.GetAlerts(ctx, sceneID)
	}
	return s.pg.GetAllAlerts(ctx)
}

func (s *AlertService) ResolveAlert(ctx context.Context, id string) error {
	return s.pg.ResolveAlert(ctx, id)
}

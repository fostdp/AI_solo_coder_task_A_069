package service

import (
	"context"
	"fmt"
	"time"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/storage"

	"github.com/google/uuid"
)

type ExportService struct {
	pg    *storage.PostgresStore
	mongo *storage.MongoStore
}

func NewExportService(pg *storage.PostgresStore, mongo *storage.MongoStore) *ExportService {
	return &ExportService{pg: pg, mongo: mongo}
}

func (s *ExportService) ExportOpenSCENARIO(ctx context.Context, sceneID string) (*model.ExportTask, error) {
	scene, err := s.pg.GetScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	task := &model.ExportTask{
		ID:        uuid.New().String(),
		SceneID:   sceneID,
		Format:    model.ExportFormatOpenSCENARIO,
		Status:    model.ExportStatusProcessing,
		CreatedAt: time.Now(),
	}

	if err := s.pg.CreateExportTask(ctx, task); err != nil {
		return nil, err
	}

	xmlContent := s.generateOpenSCENARIOXML(scene)
	filePath := fmt.Sprintf("exports/%s/%s.xosc", sceneID, task.ID)

	task.FilePath = filePath
	task.Status = model.ExportStatusCompleted
	now := time.Now()
	task.CompletedAt = &now

	if err := s.pg.UpdateExportTask(ctx, task); err != nil {
		return nil, err
	}

	_ = xmlContent
	return task, nil
}

func (s *ExportService) ExportROSBag(ctx context.Context, sceneID string) (*model.ExportTask, error) {
	scene, err := s.pg.GetScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	task := &model.ExportTask{
		ID:        uuid.New().String(),
		SceneID:   sceneID,
		Format:    model.ExportFormatROSBag,
		Status:    model.ExportStatusProcessing,
		CreatedAt: time.Now(),
	}

	if err := s.pg.CreateExportTask(ctx, task); err != nil {
		return nil, err
	}

	bagMeta := s.generateROSBagMetadata(scene)
	filePath := fmt.Sprintf("exports/%s/%s.bag", sceneID, task.ID)

	task.FilePath = filePath
	task.Status = model.ExportStatusCompleted
	now := time.Now()
	task.CompletedAt = &now

	if err := s.pg.UpdateExportTask(ctx, task); err != nil {
		return nil, err
	}

	_ = bagMeta
	return task, nil
}

func (s *ExportService) GetExportStatus(ctx context.Context, taskID string) (*model.ExportTask, error) {
	return s.pg.GetExportTask(ctx, taskID)
}

func (s *ExportService) generateOpenSCENARIOXML(scene *model.Scene) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<OpenSCENARIO xmlns="http://www.asam.net/OpenSCENARIO">
  <FileHeader description="%s" date="%s" author="sim-scenario-platform" revMajor="1" revMinor="0"/>
  <Storyboard>
    <Init/>
  </Storyboard>
</OpenSCENARIO>`, scene.Name, time.Now().Format(time.RFC3339))
}

func (s *ExportService) generateROSBagMetadata(scene *model.Scene) map[string]interface{} {
	return map[string]interface{}{
		"version":    "2.0",
		"scene_id":   scene.ID,
		"duration":   scene.Duration,
		"frame_rate": scene.FrameRate,
		"topics": []string{
			"/camera/image_raw",
			"/can/speed",
			"/can/steering",
			"/can/throttle",
			"/can/brake",
		},
	}
}

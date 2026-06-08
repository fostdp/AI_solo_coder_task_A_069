package service

import (
	"context"
	"time"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/storage"

	"github.com/google/uuid"
)

type AnnotationService struct {
	pg    *storage.PostgresStore
	mongo *storage.MongoStore
}

func NewAnnotationService(pg *storage.PostgresStore, mongo *storage.MongoStore) *AnnotationService {
	return &AnnotationService{pg: pg, mongo: mongo}
}

func (s *AnnotationService) CreateAnnotation(ctx context.Context, sceneID string, annotation *model.Annotation) (*model.Annotation, error) {
	annotation.ID = uuid.New().String()
	annotation.SceneID = sceneID
	annotation.CreatedAt = time.Now()
	annotation.UpdatedAt = time.Now()

	if err := s.mongo.CreateAnnotation(ctx, annotation); err != nil {
		return nil, err
	}

	s.pg.UpdateSceneStatus(ctx, sceneID, model.SceneStatusAnnotated)
	return annotation, nil
}

func (s *AnnotationService) UpdateAnnotation(ctx context.Context, id string, annotation *model.Annotation) (*model.Annotation, error) {
	existing, err := s.mongo.GetAnnotation(ctx, id)
	if err != nil {
		return nil, err
	}
	annotation.ID = existing.ID
	annotation.SceneID = existing.SceneID
	annotation.CreatedAt = existing.CreatedAt
	annotation.UpdatedAt = time.Now()

	if err := s.mongo.UpdateAnnotation(ctx, id, annotation); err != nil {
		return nil, err
	}
	return annotation, nil
}

func (s *AnnotationService) GetAnnotationsByScene(ctx context.Context, sceneID string) ([]*model.Annotation, error) {
	return s.mongo.GetAnnotationsByScene(ctx, sceneID)
}

func (s *AnnotationService) DeleteAnnotation(ctx context.Context, id string) error {
	return s.mongo.DeleteAnnotation(ctx, id)
}

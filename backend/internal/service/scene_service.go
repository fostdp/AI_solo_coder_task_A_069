package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/parser"
	"sim-scenario-platform/internal/storage"

	"github.com/google/uuid"
)

type SceneService struct {
	pg    *storage.PostgresStore
	mongo *storage.MongoStore
	minio *storage.MinioStore
	video *parser.VideoParser
	can   *parser.CANParser
}

func NewSceneService(pg *storage.PostgresStore, mongo *storage.MongoStore, minio *storage.MinioStore) *SceneService {
	return &SceneService{
		pg:    pg,
		mongo: mongo,
		minio: minio,
		video: parser.NewVideoParser(),
		can:   parser.NewCANParser(),
	}
}

func (s *SceneService) UploadScene(ctx context.Context, name, desc, sceneType string, tags []string, videoFile io.Reader, videoSize int64, canFile io.Reader, canSize int64) (*model.Scene, error) {
	id := uuid.New().String()
	now := time.Now()

	scene := &model.Scene{
		ID:          id,
		Name:        name,
		Description: desc,
		Status:      model.SceneStatusUploading,
		SceneType:   model.SceneType(sceneType),
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        tags,
	}

	if err := s.pg.CreateScene(ctx, scene); err != nil {
		return nil, fmt.Errorf("failed to create scene: %w", err)
	}

	videoPath := fmt.Sprintf("scenes/%s/video.mp4", id)
	if err := s.minio.UploadFile(ctx, videoPath, videoFile, videoSize, "video/mp4"); err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}
	scene.VideoPath = videoPath

	s.pg.UpdateSceneStatus(ctx, id, model.SceneStatusParsing)

	canLogPath := fmt.Sprintf("scenes/%s/can_log.csv", id)
	if canFile != nil {
		if err := s.minio.UploadFile(ctx, canLogPath, canFile, canSize, "text/csv"); err != nil {
			return nil, fmt.Errorf("failed to upload CAN log: %w", err)
		}
		scene.CANLogPath = canLogPath

		obj, err := s.minio.DownloadFile(ctx, canLogPath)
		if err == nil {
			canLog, err := s.can.ParseCANLog(obj, id)
			if err == nil {
				s.mongo.SaveCANLog(ctx, canLog)
			}
			obj.Close()
		}
	}

	frames, err := s.video.ParseVideo(videoPath, id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse video: %w", err)
	}

	frameRate := 30.0
	duration := float64(len(frames)) / frameRate

	scene.Duration = duration
	scene.FrameCount = len(frames)
	scene.FrameRate = frameRate
	scene.Status = model.SceneStatusReady
	scene.UpdatedAt = time.Now()

	if err := s.pg.UpdateScene(ctx, scene); err != nil {
		return nil, fmt.Errorf("failed to update scene: %w", err)
	}

	return scene, nil
}

func (s *SceneService) GetScene(ctx context.Context, id string) (*model.Scene, error) {
	return s.pg.GetScene(ctx, id)
}

func (s *SceneService) ListScenes(ctx context.Context, filter *model.SceneFilter) ([]*model.Scene, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	return s.pg.ListScenes(ctx, filter)
}

func (s *SceneService) DeleteScene(ctx context.Context, id string) error {
	scene, err := s.pg.GetScene(ctx, id)
	if err != nil {
		return err
	}

	if scene.VideoPath != "" {
		s.minio.DeleteFile(ctx, scene.VideoPath)
	}
	if scene.CANLogPath != "" {
		s.minio.DeleteFile(ctx, scene.CANLogPath)
	}

	files, _ := s.minio.ListFiles(ctx, fmt.Sprintf("scenes/%s/frames/", id))
	for _, f := range files {
		s.minio.DeleteFile(ctx, f)
	}

	return s.pg.DeleteScene(ctx, id)
}

func (s *SceneService) GetSceneStats(ctx context.Context) (*model.SceneStats, error) {
	return s.pg.GetSceneStats(ctx)
}

func (s *SceneService) GetCANSignals(ctx context.Context, sceneID string) (*model.CANLog, error) {
	return s.mongo.GetCANLog(ctx, sceneID)
}

func (s *SceneService) GetFrameURL(ctx context.Context, sceneID string, frameIndex int) (string, error) {
	objectKey := fmt.Sprintf("scenes/%s/frames/frame_%06d.jpg", sceneID, frameIndex)
	return s.minio.GetPresignedURL(ctx, objectKey)
}

func (s *SceneService) GetReplayData(ctx context.Context, sceneID string, startTime, endTime float64) (map[string]interface{}, error) {
	scene, err := s.pg.GetScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	canLog, _ := s.mongo.GetCANLog(ctx, sceneID)

	var filteredSignals []model.CANSignal
	if canLog != nil {
		for _, sig := range canLog.Signals {
			if sig.Timestamp >= startTime && sig.Timestamp <= endTime {
				filteredSignals = append(filteredSignals, sig)
			}
		}
	}

	startFrame := int(startTime * scene.FrameRate)
	endFrame := int(endTime * scene.FrameRate)
	if startFrame < 0 {
		startFrame = 0
	}
	if endFrame >= scene.FrameCount {
		endFrame = scene.FrameCount - 1
	}

	var frameURLs []string
	for i := startFrame; i <= endFrame; i++ {
		u, err := s.minio.GetPresignedURL(ctx, fmt.Sprintf("scenes/%s/frames/frame_%06d.jpg", sceneID, i))
		if err == nil {
			frameURLs = append(frameURLs, u)
		}
	}

	return map[string]interface{}{
		"scene_id":    sceneID,
		"start_time":  startTime,
		"end_time":    endTime,
		"frame_urls":  frameURLs,
		"can_signals": filteredSignals,
		"frame_rate":  scene.FrameRate,
	}, nil
}

package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sim-scenario-platform/internal/config"
	"sim-scenario-platform/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, cfg *config.Config) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS scenes (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			video_path VARCHAR(500) DEFAULT '',
			can_log_path VARCHAR(500) DEFAULT '',
			duration DOUBLE PRECISION DEFAULT 0,
			frame_count INTEGER DEFAULT 0,
			frame_rate DOUBLE PRECISION DEFAULT 0,
			status VARCHAR(50) NOT NULL DEFAULT 'uploading',
			scene_type VARCHAR(50) NOT NULL DEFAULT 'highway',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			tags JSONB DEFAULT '[]'
		)`,
		`CREATE TABLE IF NOT EXISTS export_tasks (
			id VARCHAR(36) PRIMARY KEY,
			scene_id VARCHAR(36) NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
			format VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			file_path VARCHAR(500) DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id VARCHAR(36) PRIMARY KEY,
			scene_id VARCHAR(36) NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			details JSONB DEFAULT '{}',
			resolved BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}
	for _, q := range queries {
		if _, err := s.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) CreateScene(ctx context.Context, scene *model.Scene) error {
	tags, _ := json.Marshal(scene.Tags)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO scenes (id, name, description, video_path, can_log_path, duration, frame_count, frame_rate, status, scene_type, created_at, updated_at, tags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		scene.ID, scene.Name, scene.Description, scene.VideoPath, scene.CANLogPath,
		scene.Duration, scene.FrameCount, scene.FrameRate, scene.Status, scene.SceneType,
		scene.CreatedAt, scene.UpdatedAt, tags,
	)
	return err
}

func (s *PostgresStore) GetScene(ctx context.Context, id string) (*model.Scene, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, description, video_path, can_log_path, duration, frame_count, frame_rate, status, scene_type, created_at, updated_at, tags
		 FROM scenes WHERE id = $1`, id)

	var scene model.Scene
	var tags []byte
	err := row.Scan(&scene.ID, &scene.Name, &scene.Description, &scene.VideoPath, &scene.CANLogPath,
		&scene.Duration, &scene.FrameCount, &scene.FrameRate, &scene.Status, &scene.SceneType,
		&scene.CreatedAt, &scene.UpdatedAt, &tags)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tags, &scene.Tags)
	return &scene, nil
}

func (s *PostgresStore) ListScenes(ctx context.Context, filter *model.SceneFilter) ([]*model.Scene, error) {
	query := `SELECT id, name, description, video_path, can_log_path, duration, frame_count, frame_rate, status, scene_type, created_at, updated_at, tags
			  FROM scenes WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filter.SceneType != "" {
		query += fmt.Sprintf(" AND scene_type = $%d", argIdx)
		args = append(args, filter.SceneType)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Tag != "" {
		query += fmt.Sprintf(" AND tags @> $%d", argIdx)
		t, _ := json.Marshal([]string{filter.Tag})
		args = append(args, t)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.PageSize > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenes []*model.Scene
	for rows.Next() {
		var scene model.Scene
		var tags []byte
		if err := rows.Scan(&scene.ID, &scene.Name, &scene.Description, &scene.VideoPath, &scene.CANLogPath,
			&scene.Duration, &scene.FrameCount, &scene.FrameRate, &scene.Status, &scene.SceneType,
			&scene.CreatedAt, &scene.UpdatedAt, &tags); err != nil {
			return nil, err
		}
		json.Unmarshal(tags, &scene.Tags)
		scenes = append(scenes, &scene)
	}
	return scenes, nil
}

func (s *PostgresStore) UpdateScene(ctx context.Context, scene *model.Scene) error {
	tags, _ := json.Marshal(scene.Tags)
	_, err := s.pool.Exec(ctx,
		`UPDATE scenes SET name=$2, description=$3, video_path=$4, can_log_path=$5, duration=$6, frame_count=$7,
		 frame_rate=$8, status=$9, scene_type=$10, updated_at=$11, tags=$12 WHERE id=$1`,
		scene.ID, scene.Name, scene.Description, scene.VideoPath, scene.CANLogPath,
		scene.Duration, scene.FrameCount, scene.FrameRate, scene.Status, scene.SceneType,
		scene.UpdatedAt, tags,
	)
	return err
}

func (s *PostgresStore) DeleteScene(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM scenes WHERE id = $1`, id)
	return err
}

func (s *PostgresStore) GetSceneStats(ctx context.Context) (*model.SceneStats, error) {
	var totalCount int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM scenes`).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `SELECT scene_type, COUNT(*) FROM scenes GROUP BY scene_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[string]int)
	for rows.Next() {
		var st string
		var cnt int
		if err := rows.Scan(&st, &cnt); err != nil {
			return nil, err
		}
		dist[st] = cnt
	}

	return &model.SceneStats{
		TotalCount:            totalCount,
		AnnotationCompletion:  0,
		SceneTypeDistribution: dist,
	}, nil
}

func (s *PostgresStore) CreateExportTask(ctx context.Context, task *model.ExportTask) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO export_tasks (id, scene_id, format, status, file_path, created_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		task.ID, task.SceneID, task.Format, task.Status, task.FilePath, task.CreatedAt, task.CompletedAt,
	)
	return err
}

func (s *PostgresStore) GetExportTask(ctx context.Context, id string) (*model.ExportTask, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, scene_id, format, status, file_path, created_at, completed_at FROM export_tasks WHERE id = $1`, id)
	var task model.ExportTask
	err := row.Scan(&task.ID, &task.SceneID, &task.Format, &task.Status, &task.FilePath, &task.CreatedAt, &task.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *PostgresStore) UpdateExportTask(ctx context.Context, task *model.ExportTask) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE export_tasks SET status=$2, file_path=$3, completed_at=$4 WHERE id=$1`,
		task.ID, task.Status, task.FilePath, task.CompletedAt,
	)
	return err
}

func (s *PostgresStore) CreateAlert(ctx context.Context, alert *model.Alert) error {
	details, _ := json.Marshal(alert.Details)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO alerts (id, scene_id, type, severity, message, details, resolved, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		alert.ID, alert.SceneID, alert.Type, alert.Severity, alert.Message, details, alert.Resolved, alert.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetAlerts(ctx context.Context, sceneID string) ([]*model.Alert, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, scene_id, type, severity, message, details, resolved, created_at FROM alerts WHERE scene_id = $1 ORDER BY created_at DESC`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*model.Alert
	for rows.Next() {
		var alert model.Alert
		var details []byte
		if err := rows.Scan(&alert.ID, &alert.SceneID, &alert.Type, &alert.Severity, &alert.Message, &details, &alert.Resolved, &alert.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(details, &alert.Details)
		alerts = append(alerts, &alert)
	}
	return alerts, nil
}

func (s *PostgresStore) GetAllAlerts(ctx context.Context) ([]*model.Alert, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, scene_id, type, severity, message, details, resolved, created_at FROM alerts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*model.Alert
	for rows.Next() {
		var alert model.Alert
		var details []byte
		if err := rows.Scan(&alert.ID, &alert.SceneID, &alert.Type, &alert.Severity, &alert.Message, &details, &alert.Resolved, &alert.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(details, &alert.Details)
		alerts = append(alerts, &alert)
	}
	return alerts, nil
}

func (s *PostgresStore) ResolveAlert(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE alerts SET resolved = TRUE WHERE id = $1`, id)
	return err
}

func (s *PostgresStore) UpdateSceneStatus(ctx context.Context, id string, status model.SceneStatus) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE scenes SET status=$2, updated_at=$3 WHERE id=$1`,
		id, status, time.Now())
	return err
}

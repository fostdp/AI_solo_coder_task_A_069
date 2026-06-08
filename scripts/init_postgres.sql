CREATE TABLE scenes (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    video_path VARCHAR(500),
    can_log_path VARCHAR(500),
    duration FLOAT,
    frame_count INT,
    frame_rate FLOAT,
    status VARCHAR(50) DEFAULT 'uploading',
    scene_type VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tags TEXT[]
);

CREATE TABLE export_tasks (
    id UUID PRIMARY KEY,
    scene_id UUID REFERENCES scenes(id),
    format VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    file_path VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    error_message TEXT
);

CREATE TABLE alerts (
    id UUID PRIMARY KEY,
    scene_id UUID REFERENCES scenes(id),
    alert_type VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    message TEXT,
    frame_index INT,
    is_resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

CREATE INDEX idx_scenes_status ON scenes(status);
CREATE INDEX idx_scenes_scene_type ON scenes(scene_type);
CREATE INDEX idx_scenes_created_at ON scenes(created_at);
CREATE INDEX idx_export_tasks_status ON export_tasks(status);
CREATE INDEX idx_alerts_scene_id ON alerts(scene_id);
CREATE INDEX idx_alerts_severity ON alerts(severity);

INSERT INTO scenes (id, name, description, video_path, can_log_path, duration, frame_count, frame_rate, status, scene_type, created_at, updated_at, tags) VALUES
('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Highway Merge Scenario', 'Vehicle merging onto highway at 120km/h with surrounding traffic', '/videos/highway_merge.mp4', '/can_logs/highway_merge.csv', 3.0, 30, 10.0, 'completed', 'highway', '2026-01-15 08:30:00', '2026-01-15 08:30:05', ARRAY['highway', 'merge', 'high-speed']),
('b2c3d4e5-f6a7-8901-bcde-f12345678901', 'Urban Intersection Left Turn', 'Left turn at urban intersection with pedestrian crossing', '/videos/urban_left_turn.mp4', '/can_logs/urban_left_turn.csv', 3.0, 30, 10.0, 'completed', 'urban', '2026-01-16 10:15:00', '2026-01-16 10:15:05', ARRAY['urban', 'intersection', 'pedestrian']),
('c3d4e5f6-a7b8-9012-cdef-123456789012', 'Parking Lot Navigation', 'Autonomous parking in crowded lot with dynamic obstacles', '/videos/parking_nav.mp4', '/can_logs/parking_nav.csv', 3.0, 30, 10.0, 'processing', 'parking', '2026-01-17 14:20:00', '2026-01-17 14:20:03', ARRAY['parking', 'low-speed', 'obstacle']),
('d4e5f6a7-b8c9-0123-defa-234567890123', 'Rainy Highway Emergency Brake', 'Emergency braking on wet highway at high speed', '/videos/rain_brake.mp4', '/can_logs/rain_brake.csv', 3.0, 30, 10.0, 'uploading', 'highway', '2026-01-18 09:45:00', '2026-01-18 09:45:00', ARRAY['highway', 'rain', 'emergency', 'braking']),
('e5f6a7b8-c9d0-1234-efab-345678901234', 'Night Urban Pedestrian Detection', 'Pedestrian detection at night with low visibility', '/videos/night_pedestrian.mp4', '/can_logs/night_pedestrian.csv', 3.0, 30, 10.0, 'failed', 'urban', '2026-01-19 22:10:00', '2026-01-19 22:10:08', ARRAY['urban', 'night', 'pedestrian', 'low-visibility']);

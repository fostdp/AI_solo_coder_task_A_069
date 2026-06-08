export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
  label: string;
  track_id: number;
  frame_index: number;
}

export interface Annotation {
  id: number;
  scene_id: number;
  frame_index: number;
  bounding_boxes: BoundingBox[];
  created_at: string;
  updated_at: string;
}

export interface CANSignal {
  timestamp: number;
  speed: number;
  steering_angle: number;
  throttle: number;
  brake: number;
}

export interface CANLog {
  id: number;
  scene_id: number;
  signals: CANSignal[];
  created_at: string;
}

export interface Scene {
  id: number;
  name: string;
  description: string;
  type: 'highway' | 'urban' | 'rural' | 'parking';
  status: 'uploaded' | 'processing' | 'ready' | 'error';
  duration: number;
  frame_count: number;
  file_path: string;
  created_at: string;
  updated_at: string;
}

export interface Alert {
  id: number;
  type: 'time_sync' | 'annotation_quality';
  scene_id: number;
  scene_name: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  created_at: string;
  resolved: boolean;
}

export interface ExportTask {
  id: number;
  scene_id: number;
  format: 'openscenario' | 'rosbag';
  status: 'pending' | 'processing' | 'completed' | 'failed';
  file_path: string;
  created_at: string;
}

export interface SceneStats {
  total_scenes: number;
  annotated_scenes: number;
  annotation_rate: number;
  alert_count: number;
  type_distribution: { type: string; count: number }[];
  scenes_per_month: { month: string; count: number }[];
}

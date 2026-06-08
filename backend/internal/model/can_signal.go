package model

type CANSignal struct {
	Timestamp     float64 `json:"timestamp" bson:"timestamp"`
	Speed         float64 `json:"speed" bson:"speed"`
	SteeringAngle float64 `json:"steering_angle" bson:"steering_angle"`
	Throttle      float64 `json:"throttle" bson:"throttle"`
	Brake         float64 `json:"brake" bson:"brake"`
	Gear          string  `json:"gear" bson:"gear"`
}

type CANLog struct {
	SceneID   string      `json:"scene_id" bson:"scene_id"`
	Signals   []CANSignal `json:"signals" bson:"signals"`
	StartTime float64     `json:"start_time" bson:"start_time"`
	EndTime   float64     `json:"end_time" bson:"end_time"`
}

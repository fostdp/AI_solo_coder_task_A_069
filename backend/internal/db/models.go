package db

import (
	"encoding/json"
	"time"
)

type Device struct {
	ID                   int
	DeviceCode           string
	DeviceName           string
	DeviceType           string
	Area                 string
	RatedPower           float64
	RatedCoolingCapacity float64
	SetpointTemp         float64
	CreatedAt            time.Time
}

type DeviceData struct {
	Time            time.Time
	DeviceID        int
	SupplyTemp      float64
	ReturnTemp      float64
	FlowRate        float64
	Power           float64
	Pressure        float64
	COP             float64
	CoolingCapacity float64
	Status          int
}

type PUERecord struct {
	Time        time.Time
	ITPower     float64
	CoolingPower float64
	TotalPower  float64
	PUEValue    float64
}

type CoolingAllocation struct {
	Time               time.Time
	Area               string
	OptimizationMethod string
	HeatLoad           float64
	AllocatedCooling   float64
	SetpointTemp       float64
	ActualTemp         float64
}

type Alert struct {
	ID          int
	Time        time.Time
	Level       int
	DeviceID    *int
	AlertType   string
	Message     string
	Value       float64
	Threshold   float64
	Acknowledged bool
	DingTalkSent bool
}

type OptimizationSuggestion struct {
	ID             int
	Time           time.Time
	SuggestionType string
	Content        json.RawMessage
	PUEValue       float64
	Applied        bool
}

type DeviceStatus struct {
	DeviceID        int
	DeviceCode      string
	DeviceName      string
	DeviceType      string
	COP             float64
	Status          string
	Power           float64
	CoolingCapacity float64
}

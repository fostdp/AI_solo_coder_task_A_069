package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"sim-scenario-platform/internal/model"
)

type CANParser struct{}

func NewCANParser() *CANParser {
	return &CANParser{}
}

func (p *CANParser) ParseCANLog(reader io.Reader, sceneID string) (*model.CANLog, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = 6

	_, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var signals []model.CANSignal
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		timestamp, _ := strconv.ParseFloat(strings.TrimSpace(record[0]), 64)
		speed, _ := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		steeringAngle, _ := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		throttle, _ := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)
		brake, _ := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		gear := strings.TrimSpace(record[5])

		signals = append(signals, model.CANSignal{
			Timestamp:     timestamp,
			Speed:         speed,
			SteeringAngle: steeringAngle,
			Throttle:      throttle,
			Brake:         brake,
			Gear:          gear,
		})
	}

	if len(signals) == 0 {
		return nil, fmt.Errorf("no signals found in CAN log")
	}

	return &model.CANLog{
		SceneID:   sceneID,
		Signals:   signals,
		StartTime: signals[0].Timestamp,
		EndTime:   signals[len(signals)-1].Timestamp,
	}, nil
}

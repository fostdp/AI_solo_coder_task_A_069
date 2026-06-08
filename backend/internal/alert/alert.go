package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"dc-cooling-optimizer/internal/db"
)

type DeviceThresholds struct {
	MinCOP             float64
	MaxSupplyTempDelta float64
	MaxPowerRatio      float64
}

type AlertManager struct {
	db                 *db.DB
	dingtalkWebhook    string
	thresholds         DeviceThresholds
	deviceViolationStart map[int]time.Time
	pueViolationStart  *time.Time
	stopCh             chan struct{}
}

func New(database *db.DB, dingtalkWebhook string) *AlertManager {
	return &AlertManager{
		db:              database,
		dingtalkWebhook: dingtalkWebhook,
		thresholds: DeviceThresholds{
			MinCOP:             3.0,
			MaxSupplyTempDelta: 5.0,
			MaxPowerRatio:      1.2,
		},
		deviceViolationStart: make(map[int]time.Time),
		stopCh:               make(chan struct{}),
	}
}

func (am *AlertManager) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-am.stopCh:
			return
		case <-ticker.C:
			am.checkDeviceAlerts(ctx)
			am.checkPUEAlert(ctx)
		}
	}
}

func (am *AlertManager) Stop() {
	close(am.stopCh)
}

func (am *AlertManager) checkDeviceAlerts(ctx context.Context) {
	devices, err := am.db.GetAllDevices(ctx)
	if err != nil {
		log.Printf("alert: failed to get devices: %v", err)
		return
	}

	for _, d := range devices {
		data, err := am.db.GetDeviceData24h(ctx, d.ID)
		if err != nil {
			log.Printf("alert: failed to get device data for device %d: %v", d.ID, err)
			continue
		}

		if len(data) == 0 {
			delete(am.deviceViolationStart, d.ID)
			continue
		}

		latest := data[len(data)-1]

		var violation string
		var value float64
		var threshold float64

		if latest.COP < am.thresholds.MinCOP {
			violation = "low_cop"
			value = latest.COP
			threshold = am.thresholds.MinCOP
		} else if latest.SupplyTemp-d.SetpointTemp > am.thresholds.MaxSupplyTempDelta {
			violation = "high_supply_temp"
			value = latest.SupplyTemp
			threshold = d.SetpointTemp + am.thresholds.MaxSupplyTempDelta
		} else if d.RatedPower > 0 && latest.Power/d.RatedPower > am.thresholds.MaxPowerRatio {
			violation = "high_power"
			value = latest.Power / d.RatedPower
			threshold = am.thresholds.MaxPowerRatio
		}

		if violation != "" {
			startTime, exists := am.deviceViolationStart[d.ID]
			if exists {
				if time.Since(startTime) >= 10*time.Minute {
					alert := &db.Alert{
						Time:      time.Now(),
						Level:     1,
						DeviceID:  &d.ID,
						AlertType: violation,
						Message:   fmt.Sprintf("设备 %s 参数超标持续10分钟", d.DeviceName),
						Value:     value,
						Threshold: threshold,
					}
					if err := am.db.InsertAlert(ctx, alert); err != nil {
						log.Printf("alert: failed to insert device alert: %v", err)
					} else {
						if err := am.sendDingTalk(alert); err != nil {
							log.Printf("alert: failed to send dingtalk: %v", err)
						}
					}
					delete(am.deviceViolationStart, d.ID)
				}
			} else {
				am.deviceViolationStart[d.ID] = time.Now()
			}
		} else {
			delete(am.deviceViolationStart, d.ID)
		}
	}
}

func (am *AlertManager) checkPUEAlert(ctx context.Context) {
	records, err := am.db.GetPUERecords(ctx, 1)
	if err != nil {
		log.Printf("alert: failed to get pue records: %v", err)
		return
	}

	if len(records) == 0 {
		return
	}

	latest := records[len(records)-1]

	if latest.PUEValue > 1.5 {
		if am.pueViolationStart != nil {
			if time.Since(*am.pueViolationStart) >= 30*time.Minute {
				alert := &db.Alert{
					Time:      time.Now(),
					Level:     2,
					DeviceID:  nil,
					AlertType: "high_pue",
					Message:   "PUE超过1.5持续30分钟",
					Value:     latest.PUEValue,
					Threshold: 1.5,
				}
				if err := am.db.InsertAlert(ctx, alert); err != nil {
					log.Printf("alert: failed to insert pue alert: %v", err)
				} else {
					if err := am.sendDingTalk(alert); err != nil {
						log.Printf("alert: failed to send dingtalk: %v", err)
					}
				}
				am.pueViolationStart = nil
			}
		} else {
			now := time.Now()
			am.pueViolationStart = &now
		}
	} else {
		am.pueViolationStart = nil
	}
}

func (am *AlertManager) sendDingTalk(alert *db.Alert) error {
	if am.dingtalkWebhook == "" {
		return nil
	}

	levelStr := "一级"
	if alert.Level == 2 {
		levelStr = "二级"
	}

	text := fmt.Sprintf("### 数据中心告警\n\n- **告警级别**: %s\n- **告警类型**: %s\n- **告警信息**: %s\n- **当前值**: %.2f\n- **阈值**: %.2f\n- **时间**: %s",
		levelStr, alert.AlertType, alert.Message, alert.Value, alert.Threshold, alert.Time.Format("2006-01-02 15:04:05"))

	msg := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "数据中心告警",
			"text":  text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal dingtalk message: %w", err)
	}

	resp, err := http.Post(am.dingtalkWebhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send dingtalk webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (am *AlertManager) CheckDeviceAlert(ctx context.Context, deviceID int, cop float64, supplyTemp float64, setpointTemp float64, power float64, ratedPower float64) *db.Alert {
	var violation string
	var value float64
	var threshold float64

	if cop < am.thresholds.MinCOP {
		violation = "low_cop"
		value = cop
		threshold = am.thresholds.MinCOP
	} else if delta := supplyTemp - setpointTemp; delta > am.thresholds.MaxSupplyTempDelta {
		violation = "high_supply_temp"
		value = supplyTemp
		threshold = setpointTemp + am.thresholds.MaxSupplyTempDelta
	} else if ratedPower > 0 && power/ratedPower > am.thresholds.MaxPowerRatio {
		violation = "high_power"
		value = power / ratedPower
		threshold = am.thresholds.MaxPowerRatio
	}

	if violation == "" {
		return nil
	}

	alert := &db.Alert{
		Time:      time.Now(),
		Level:     1,
		DeviceID:  &deviceID,
		AlertType: violation,
		Message:   fmt.Sprintf("设备参数超标: %s", violation),
		Value:     value,
		Threshold: threshold,
	}

	if err := am.db.InsertAlert(ctx, alert); err != nil {
		log.Printf("alert: failed to insert device alert: %v", err)
		return nil
	}

	return alert
}

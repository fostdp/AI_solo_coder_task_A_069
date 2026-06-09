package alarm_notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"dc-cooling-optimizer/internal/config"
	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/modbus_gateway"
	"dc-cooling-optimizer/internal/pue_calculator"
)

type AlertEvent struct {
	Alert      *db.Alert
	TriggeredAt time.Time
}

type Notifier struct {
	db                   *db.DB
	cfg                  *config.AlarmNotifierConfig
	deviceInCh           <-chan *modbus_gateway.DeviceDataEvent
	pueInCh              <-chan *pue_calculator.PUEEvent
	outCh                chan *AlertEvent
	stopCh               chan struct{}
	mu                   sync.Mutex
	deviceViolationStart map[string]time.Time
	pueViolationStart    *time.Time
	latestDeviceData     map[int]*db.DeviceData
	latestPUE            float64
}

func New(database *db.DB, cfg *config.AlarmNotifierConfig, deviceInCh <-chan *modbus_gateway.DeviceDataEvent, pueInCh <-chan *pue_calculator.PUEEvent) *Notifier {
	return &Notifier{
		db:                   database,
		cfg:                  cfg,
		deviceInCh:           deviceInCh,
		pueInCh:              pueInCh,
		outCh:                make(chan *AlertEvent, 64),
		stopCh:               make(chan struct{}),
		deviceViolationStart: make(map[string]time.Time),
		latestDeviceData:     make(map[int]*db.DeviceData),
	}
}

func (n *Notifier) Start(ctx context.Context) {
	go n.deviceDataListener(ctx)
	go n.pueListener(ctx)
	go n.checkLoop(ctx)
}

func (n *Notifier) deviceDataListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stopCh:
			return
		case evt := <-n.deviceInCh:
			if evt == nil || evt.Data == nil {
				continue
			}
			n.mu.Lock()
			n.latestDeviceData[evt.Data.DeviceID] = evt.Data
			n.mu.Unlock()
		}
	}
}

func (n *Notifier) pueListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stopCh:
			return
		case evt := <-n.pueInCh:
			if evt == nil {
				continue
			}
			n.mu.Lock()
			n.latestPUE = evt.PUE
			n.mu.Unlock()
		}
	}
}

func (n *Notifier) checkLoop(ctx context.Context) {
	ticker := time.NewTicker(n.cfg.CheckInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.checkDeviceAlerts(ctx)
			n.checkPUEAlert(ctx)
		}
	}
}

func (n *Notifier) Stop() {
	close(n.stopCh)
}

func (n *Notifier) Output() <-chan *AlertEvent {
	return n.outCh
}

type violationInfo struct {
	violationType string
	value         float64
	threshold     float64
}

func (n *Notifier) checkDeviceAlerts(ctx context.Context) {
	devices, err := n.db.GetAllDevices(ctx)
	if err != nil {
		log.Printf("alarm_notifier: failed to get devices: %v", err)
		return
	}

	for _, d := range devices {
		n.mu.Lock()
		data, ok := n.latestDeviceData[d.ID]
		n.mu.Unlock()

		if !ok || data == nil {
			continue
		}

		var violations []violationInfo
		activeKeys := make(map[string]bool)

		if data.COP < n.cfg.DeviceThresholds.MinCOP {
			key := fmt.Sprintf("%d:%s", d.ID, "low_cop")
			activeKeys[key] = true
			violations = append(violations, violationInfo{
				violationType: "low_cop",
				value:         data.COP,
				threshold:     n.cfg.DeviceThresholds.MinCOP,
			})
		}

		if data.SupplyTemp-d.SetpointTemp > n.cfg.DeviceThresholds.MaxSupplyTempDelta {
			key := fmt.Sprintf("%d:%s", d.ID, "high_supply_temp")
			activeKeys[key] = true
			violations = append(violations, violationInfo{
				violationType: "high_supply_temp",
				value:         data.SupplyTemp,
				threshold:     d.SetpointTemp + n.cfg.DeviceThresholds.MaxSupplyTempDelta,
			})
		}

		if d.RatedPower > 0 && data.Power/d.RatedPower > n.cfg.DeviceThresholds.MaxPowerRatio {
			key := fmt.Sprintf("%d:%s", d.ID, "high_power")
			activeKeys[key] = true
			violations = append(violations, violationInfo{
				violationType: "high_power",
				value:         data.Power / d.RatedPower,
				threshold:     n.cfg.DeviceThresholds.MaxPowerRatio,
			})
		}

		for _, v := range violations {
			key := fmt.Sprintf("%d:%s", d.ID, v.violationType)

			n.mu.Lock()
			startTime, exists := n.deviceViolationStart[key]
			if !exists {
				n.deviceViolationStart[key] = time.Now()
				n.mu.Unlock()
				continue
			}
			n.mu.Unlock()

			if time.Since(startTime) < n.cfg.Level1ViolationDuration() {
				continue
			}

			alert := &db.Alert{
				Time:      time.Now(),
				Level:     1,
				DeviceID:  &d.ID,
				AlertType: v.violationType,
				Message:   fmt.Sprintf("设备 %s 参数超标: %s", d.DeviceName, v.violationType),
				Value:     v.value,
				Threshold: v.threshold,
			}

			if err := n.db.InsertAlert(ctx, alert); err != nil {
				log.Printf("alarm_notifier: failed to insert device alert: %v", err)
			} else {
				select {
				case n.outCh <- &AlertEvent{Alert: alert, TriggeredAt: time.Now()}:
				default:
				}
				if err := n.sendDingTalk(alert); err != nil {
					log.Printf("alarm_notifier: failed to send dingtalk for alert %d: %v", alert.ID, err)
				}
			}

			n.mu.Lock()
			delete(n.deviceViolationStart, key)
			n.mu.Unlock()
		}

		allTypes := []string{"low_cop", "high_supply_temp", "high_power"}
		for _, vt := range allTypes {
			key := fmt.Sprintf("%d:%s", d.ID, vt)
			if !activeKeys[key] {
				n.mu.Lock()
				delete(n.deviceViolationStart, key)
				n.mu.Unlock()
			}
		}
	}
}

func (n *Notifier) checkPUEAlert(ctx context.Context) {
	n.mu.Lock()
	pue := n.latestPUE
	n.mu.Unlock()

	if pue > n.cfg.Level2PUEThreshold {
		n.mu.Lock()
		if n.pueViolationStart == nil {
			now := time.Now()
			n.pueViolationStart = &now
			n.mu.Unlock()
			return
		}
		startTime := *n.pueViolationStart
		n.mu.Unlock()

		if time.Since(startTime) < n.cfg.Level2ViolationDuration() {
			return
		}

		alert := &db.Alert{
			Time:      time.Now(),
			Level:     2,
			DeviceID:  nil,
			AlertType: "high_pue",
			Message:   fmt.Sprintf("PUE超过%.2f持续%s", n.cfg.Level2PUEThreshold, n.cfg.Level2ViolationDuration()),
			Value:     pue,
			Threshold: n.cfg.Level2PUEThreshold,
		}

		if err := n.db.InsertAlert(ctx, alert); err != nil {
			log.Printf("alarm_notifier: failed to insert pue alert: %v", err)
		} else {
			select {
			case n.outCh <- &AlertEvent{Alert: alert, TriggeredAt: time.Now()}:
			default:
			}
			if err := n.sendDingTalk(alert); err != nil {
				log.Printf("alarm_notifier: failed to send dingtalk for alert %d: %v", alert.ID, err)
			}
		}

		n.mu.Lock()
		n.pueViolationStart = nil
		n.mu.Unlock()
	} else {
		n.mu.Lock()
		n.pueViolationStart = nil
		n.mu.Unlock()
	}
}

func (n *Notifier) sendDingTalk(alert *db.Alert) error {
	if n.cfg.DingtalkWebhook == "" {
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

	var lastErr error
	for i := 0; i <= n.cfg.DingtalkRetryCount; i++ {
		if i > 0 {
			time.Sleep(n.cfg.DingtalkRetryDelay())
		}
		resp, err := http.Post(n.cfg.DingtalkWebhook, "application/json", bytes.NewReader(body))
		if err != nil {
			lastErr = fmt.Errorf("send dingtalk webhook: %w", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if alert.ID > 0 {
				if err := n.db.MarkDingTalkSent(context.Background(), alert.ID); err != nil {
					log.Printf("alarm_notifier: failed to mark dingtalk_sent for alert %d: %v", alert.ID, err)
				}
			}
			return nil
		}
		lastErr = fmt.Errorf("dingtalk webhook returned status %d", resp.StatusCode)
	}

	return lastErr
}

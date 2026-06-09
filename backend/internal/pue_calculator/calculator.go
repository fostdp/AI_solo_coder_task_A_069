package pue_calculator

import (
	"context"
	"log"
	"sync"
	"time"

	"dc-cooling-optimizer/internal/config"
	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/modbus_gateway"
)

type PUEEvent struct {
	PUE               float64
	ITPower           float64
	CoolingPower      float64
	DistributionLoss  float64
	OtherInfraPower   float64
	TotalFacilityPower float64
	CalculatedAt      time.Time
}

type Calculator struct {
	db              *db.DB
	cfg             *config.PUECalculatorConfig
	inCh            <-chan *modbus_gateway.DeviceDataEvent
	outCh           chan *PUEEvent
	stopCh          chan struct{}
	devicePowerCache map[int]float64
	mu              sync.Mutex
}

func New(database *db.DB, cfg *config.PUECalculatorConfig, inCh <-chan *modbus_gateway.DeviceDataEvent) *Calculator {
	return &Calculator{
		db:               database,
		cfg:              cfg,
		inCh:             inCh,
		outCh:            make(chan *PUEEvent, 64),
		stopCh:           make(chan struct{}),
		devicePowerCache: make(map[int]float64),
	}
}

func (c *Calculator) Start(ctx context.Context) {
	go c.inputListener(ctx)
	go c.calculatorLoop(ctx)
}

func (c *Calculator) Stop() {
	close(c.stopCh)
}

func (c *Calculator) Output() <-chan *PUEEvent {
	return c.outCh
}

func (c *Calculator) inputListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case evt := <-c.inCh:
			if evt == nil || evt.Data == nil {
				continue
			}
			c.mu.Lock()
			c.devicePowerCache[evt.Data.DeviceID] = evt.Data.Power
			c.mu.Unlock()
		}
	}
}

func (c *Calculator) calculatorLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.CalculationInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.calculate(ctx)
		}
	}
}

func (c *Calculator) calculate(ctx context.Context) {
	itPower := float64(c.cfg.ITPowerKW)
	if itPower <= 0 {
		log.Printf("pue_calculator: ITPower is %.2f, skipping calculation", itPower)
		return
	}

	c.mu.Lock()
	powerSnapshot := make(map[int]float64, len(c.devicePowerCache))
	for k, v := range c.devicePowerCache {
		powerSnapshot[k] = v
	}
	c.mu.Unlock()

	devices, err := c.db.GetAllDevices(ctx)
	if err != nil {
		log.Printf("pue_calculator: failed to get devices: %v", err)
		return
	}

	coolingTypes := map[string]bool{
		"chiller":       true,
		"cooling_tower": true,
		"precision_ac":  true,
		"cdu":           true,
	}

	var coolingPower float64
	for _, d := range devices {
		if !coolingTypes[d.DeviceType] {
			continue
		}
		if p, ok := powerSnapshot[d.ID]; ok {
			coolingPower += p
		}
	}

	distributionLoss := float64(c.cfg.DistributionLossKW)
	otherInfraPower := float64(c.cfg.OtherInfraPowerKW)
	totalFacilityPower := itPower + coolingPower + distributionLoss + otherInfraPower
	pue := totalFacilityPower / itPower

	now := time.Now()

	evt := &PUEEvent{
		PUE:               pue,
		ITPower:           itPower,
		CoolingPower:      coolingPower,
		DistributionLoss:  distributionLoss,
		OtherInfraPower:   otherInfraPower,
		TotalFacilityPower: totalFacilityPower,
		CalculatedAt:      now,
	}

	record := &db.PUERecord{
		Time:               now,
		ITPower:            itPower,
		CoolingPower:       coolingPower,
		DistributionLoss:   distributionLoss,
		OtherInfraPower:    otherInfraPower,
		TotalFacilityPower: totalFacilityPower,
		PUEValue:           pue,
	}

	if err := c.db.InsertPUERecord(ctx, record); err != nil {
		log.Printf("pue_calculator: failed to insert pue record: %v", err)
	}

	select {
	case c.outCh <- evt:
	default:
	}
}

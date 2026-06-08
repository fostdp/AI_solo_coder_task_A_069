package pue

import (
	"context"
	"log"
	"time"

	"dc-cooling-optimizer/internal/db"
)

type PUACalculator struct {
	db               *db.DB
	itPower          float64
	distributionLoss float64
	otherInfraPower  float64
	stopCh           chan struct{}
	OnPUEUpdate      func(pue float64, itPower, coolingPower, distributionLoss, otherInfraPower, totalFacilityPower float64)
	OnHighPUE        func(pue float64)
}

func New(database *db.DB, itPower float64, distributionLoss float64, otherInfraPower float64) *PUACalculator {
	return &PUACalculator{
		db:               database,
		itPower:          itPower,
		distributionLoss: distributionLoss,
		otherInfraPower:  otherInfraPower,
		stopCh:           make(chan struct{}),
	}
}

func (p *PUACalculator) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	p.calculate(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.calculate(ctx)
		}
	}
}

func (p *PUACalculator) calculate(ctx context.Context) {
	devices, err := p.db.GetAllDevices(ctx)
	if err != nil {
		log.Printf("pue: get all devices: %v", err)
		return
	}

	var coolingPower float64
	for _, device := range devices {
		switch device.DeviceType {
		case "chiller", "cooling_tower", "precision_ac", "cdu":
		default:
			continue
		}

		history, err := p.db.GetDeviceCOPHistory(ctx, device.ID, 1)
		if err != nil {
			log.Printf("pue: get device cop history device_id=%d: %v", device.ID, err)
			continue
		}

		if len(history) == 0 {
			continue
		}

		coolingPower += history[len(history)-1].Power
	}

	totalFacilityPower := p.itPower + coolingPower + p.distributionLoss + p.otherInfraPower
	pueValue := totalFacilityPower / p.itPower

	now := time.Now()
	rec := &db.PUERecord{
		Time:               now,
		ITPower:            p.itPower,
		CoolingPower:       coolingPower,
		DistributionLoss:   p.distributionLoss,
		OtherInfraPower:    p.otherInfraPower,
		TotalFacilityPower: totalFacilityPower,
		PUEValue:           pueValue,
	}

	if err := p.db.InsertPUERecord(ctx, rec); err != nil {
		log.Printf("pue: insert pue record: %v", err)
	}

	if p.OnPUEUpdate != nil {
		p.OnPUEUpdate(pueValue, p.itPower, coolingPower, p.distributionLoss, p.otherInfraPower, totalFacilityPower)
	}

	if pueValue > 1.4 && p.OnHighPUE != nil {
		p.OnHighPUE(pueValue)
	}
}

func (p *PUACalculator) Stop() {
	close(p.stopCh)
}

func (p *PUACalculator) SetITPower(power float64) {
	p.itPower = power
}

func (p *PUACalculator) SetDistributionLoss(loss float64) {
	p.distributionLoss = loss
}

func (p *PUACalculator) SetOtherInfraPower(power float64) {
	p.otherInfraPower = power
}

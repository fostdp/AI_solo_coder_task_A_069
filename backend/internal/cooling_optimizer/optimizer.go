package cooling_optimizer

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"dc-cooling-optimizer/internal/config"
	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/pue_calculator"
)

type AreaLoad struct {
	Area         string
	HeatLoad     float64
	SetpointTemp float64
	ActualTemp   float64
}

type AllocationResult struct {
	Area             string
	HeatLoad         float64
	AllocatedCooling float64
	SetpointTemp     float64
	ActualTemp       float64
	Method           string
}

type OptimizationResult struct {
	Allocations  []AllocationResult
	TotalCooling float64
	TotalPower   float64
	EstimatedPUE float64
	Suggestions  []string
	OptimizedAt  time.Time
}

type Optimizer struct {
	db      *db.DB
	cfg     *config.CoolingOptimizerConfig
	inCh    <-chan *pue_calculator.PUEEvent
	outCh   chan *OptimizationResult
	stopCh  chan struct{}
	lastPUE float64
	mu      sync.Mutex
}

func New(database *db.DB, cfg *config.CoolingOptimizerConfig, inCh <-chan *pue_calculator.PUEEvent) *Optimizer {
	return &Optimizer{
		db:     database,
		cfg:    cfg,
		inCh:   inCh,
		outCh:  make(chan *OptimizationResult, 32),
		stopCh: make(chan struct{}),
	}
}

func (o *Optimizer) Start(ctx context.Context) {
	go o.inputListener(ctx)
	go o.optimizerLoop(ctx)
}

func (o *Optimizer) Stop() {
	close(o.stopCh)
}

func (o *Optimizer) Output() <-chan *OptimizationResult {
	return o.outCh
}

func (o *Optimizer) inputListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case evt := <-o.inCh:
			if evt == nil {
				continue
			}
			o.mu.Lock()
			o.lastPUE = evt.PUE
			o.mu.Unlock()
		}
	}
}

func (o *Optimizer) optimizerLoop(ctx context.Context) {
	ticker := time.NewTicker(o.cfg.OptimizationInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			result, err := o.Optimize(ctx)
			if err != nil {
				log.Printf("cooling_optimizer: optimization failed: %v", err)
				continue
			}
			_ = result
		}
	}
}

func (o *Optimizer) Optimize(ctx context.Context) (*OptimizationResult, error) {
	devices, err := o.db.GetAllDevices(ctx)
	if err != nil {
		return nil, err
	}

	areaDevices := make(map[string][]db.Device)
	for _, d := range devices {
		areaDevices[d.Area] = append(areaDevices[d.Area], d)
	}

	type areaAccum struct {
		heatLoad     float64
		setpointTemp float64
		actualTemp   float64
		tempCount    int
	}

	areaAccums := make(map[string]*areaAccum)
	var chillers []db.Device
	chillerCOPs := make(map[int]float64)
	var totalChillerCOP float64
	var chillerCOPCount int

	historyHours := int(o.cfg.History().Hours())

	for area, devs := range areaDevices {
		acc := &areaAccum{}
		for _, d := range devs {
			if d.DeviceType == "chiller" {
				chillers = append(chillers, d)
				history, err := o.db.GetDeviceCOPHistory(ctx, d.ID, historyHours)
				if err != nil {
					log.Printf("cooling_optimizer: get COP history for chiller %d: %v", d.ID, err)
					continue
				}
				if len(history) > 0 {
					latest := history[len(history)-1]
					chillerCOPs[d.ID] = latest.COP
					totalChillerCOP += latest.COP
					chillerCOPCount++
				}
				continue
			}
			if d.DeviceType == "precision_ac" || d.DeviceType == "cdu" {
				history, err := o.db.GetDeviceCOPHistory(ctx, d.ID, historyHours)
				if err != nil {
					log.Printf("cooling_optimizer: get COP history for device %d: %v", d.ID, err)
					continue
				}
				if len(history) == 0 {
					continue
				}
				latest := history[len(history)-1]
				heatLoad := (latest.ReturnTemp - latest.SupplyTemp) * latest.FlowRate * 4.186 / 3600
				if heatLoad > 0 {
					acc.heatLoad += heatLoad
				}
				acc.actualTemp += latest.ReturnTemp
				acc.tempCount++
				acc.setpointTemp = d.SetpointTemp
			}
		}
		areaAccums[area] = acc
	}

	var areas []AreaLoad
	for area, acc := range areaAccums {
		al := AreaLoad{
			Area:         area,
			HeatLoad:     acc.heatLoad,
			SetpointTemp: acc.setpointTemp,
		}
		if acc.tempCount > 0 {
			al.ActualTemp = acc.actualTemp / float64(acc.tempCount)
		}
		areas = append(areas, al)
	}

	sort.Slice(areas, func(i, j int) bool {
		diffI := areas[i].ActualTemp - areas[i].SetpointTemp
		diffJ := areas[j].ActualTemp - areas[j].SetpointTemp
		return diffI > diffJ
	})

	var availableCooling float64
	for _, ch := range chillers {
		availableCooling += ch.RatedCoolingCapacity * o.cfg.ChillerCapacityUtilization
	}

	var allocations []AllocationResult
	var totalCooling float64
	remaining := availableCooling

	for _, area := range areas {
		var allocate float64
		if remaining > 0 {
			allocate = math.Min(area.HeatLoad*o.cfg.SafetyMargin, remaining)
		}
		remaining -= allocate
		totalCooling += allocate
		allocations = append(allocations, AllocationResult{
			Area:             area.Area,
			HeatLoad:         area.HeatLoad,
			AllocatedCooling: allocate,
			SetpointTemp:     area.SetpointTemp,
			ActualTemp:       area.ActualTemp,
			Method:           "greedy",
		})
	}

	var suggestions []string
	for _, a := range allocations {
		if a.ActualTemp < a.SetpointTemp-o.cfg.OvercoolThresholdDelta {
			suggestions = append(suggestions, "Area "+a.Area+": overcooled, consider reducing cooling")
		}
		if a.ActualTemp > a.SetpointTemp+o.cfg.UndercoolThresholdDelta {
			suggestions = append(suggestions, "Area "+a.Area+": undercooled, consider increasing cooling")
		}
	}
	for _, ch := range chillers {
		if cop, ok := chillerCOPs[ch.ID]; ok {
			if cop < o.cfg.LowCOPThreshold {
				suggestions = append(suggestions, "Chiller "+ch.DeviceName+": COP below threshold, consider maintenance")
			}
		}
	}

	var avgChillerCOP float64
	if chillerCOPCount > 0 {
		avgChillerCOP = totalChillerCOP / float64(chillerCOPCount)
	}

	var totalHeatLoad float64
	for _, a := range areas {
		totalHeatLoad += a.HeatLoad
	}

	var totalPower float64
	var estimatedPUE float64
	if avgChillerCOP > 0 && totalHeatLoad > 0 {
		coolingPower := totalCooling / avgChillerCOP
		totalPower = totalHeatLoad + coolingPower
		estimatedPUE = totalPower / totalHeatLoad
	}

	now := time.Now()
	for _, a := range allocations {
		alloc := &db.CoolingAllocation{
			Time:               now,
			Area:               a.Area,
			OptimizationMethod: a.Method,
			HeatLoad:           a.HeatLoad,
			AllocatedCooling:   a.AllocatedCooling,
			SetpointTemp:       a.SetpointTemp,
			ActualTemp:         a.ActualTemp,
		}
		if err := o.db.InsertCoolingAllocation(ctx, alloc); err != nil {
			log.Printf("cooling_optimizer: insert cooling allocation for area %s: %v", a.Area, err)
		}
	}

	contentData := map[string]interface{}{
		"allocations": allocations,
		"suggestions": suggestions,
	}
	content, _ := json.Marshal(contentData)

	sug := &db.OptimizationSuggestion{
		Time:           now,
		SuggestionType: "cooling_optimization",
		Content:        content,
		PUEValue:       estimatedPUE,
		Applied:        false,
	}
	if err := o.db.InsertOptimizationSuggestion(ctx, sug); err != nil {
		log.Printf("cooling_optimizer: insert optimization suggestion: %v", err)
	}

	result := &OptimizationResult{
		Allocations:  allocations,
		TotalCooling: totalCooling,
		TotalPower:   totalPower,
		EstimatedPUE: estimatedPUE,
		Suggestions:  suggestions,
		OptimizedAt:  now,
	}

	select {
	case o.outCh <- result:
	default:
	}

	return result, nil
}

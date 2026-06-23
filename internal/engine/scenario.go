package engine

import (
	"sync"
	"time"

	"park-device-simulator/internal/types"
)

// ==================== 场景引擎 ====================

type ScenarioEngine struct {
	mu          sync.RWMutex
	current     string
	overrides   map[string]any
	scenarioCfg map[string]map[string]any
}

func NewScenarioEngine() *ScenarioEngine {
	return &ScenarioEngine{
		current:     "normal_workday",
		overrides:   make(map[string]any),
		scenarioCfg: make(map[string]map[string]any),
	}
}

func (e *ScenarioEngine) RegisterScenario(name string, overrides map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.scenarioCfg[name] = overrides
}

func (e *ScenarioEngine) SetScenario(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.current = name
	if overrides, ok := e.scenarioCfg[name]; ok {
		e.overrides = overrides
	} else {
		e.overrides = make(map[string]any)
	}
}

func (e *ScenarioEngine) CurrentScenario() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current
}

func (e *ScenarioEngine) BuildContext(t time.Time) types.ScenarioContext {
	e.mu.RLock()
	defer e.mu.RUnlock()

	season := Season(t)
	isWorkday := IsWorkday(t)
	hour := t.Hour()

	baseTemp := SeasonalBaseTemp(season)
	amplitude := 6.0
	if v, ok := e.overrides["outdoor_temp_base"]; ok {
		if f, ok := v.(float64); ok {
			baseTemp = f
		}
	}
	if v, ok := e.overrides["outdoor_temp_amplitude"]; ok {
		if f, ok := v.(float64); ok {
			amplitude = f
		}
	}
	outdoorTemp := OutdoorTemperature(hour, baseTemp, amplitude)

	occupancy := OccupancyRate(hour, isWorkday)
	powerLoad := PowerLoadRate(hour, isWorkday)

	if e.current == "weekend" || e.current == "holiday" {
		occupancy *= 0.2
		powerLoad *= 0.3
	}
	if e.current == "summer_peak" {
		if v, ok := e.overrides["cooling_load_factor"]; ok {
			if f, ok := v.(float64); ok {
				powerLoad = Clamp(powerLoad*f, 0, 1)
			}
		}
	}

	return types.ScenarioContext{
		Timestamp:     t,
		Scenario:      e.current,
		OutdoorTemp:   outdoorTemp,
		OccupancyRate: occupancy,
		PowerLoadRate: powerLoad,
		IsWorkday:     isWorkday,
		Season:        season,
		ExtraParams:   e.overrides,
	}
}

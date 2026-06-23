package engine

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"park-device-simulator/internal/types"
)

// ==================== 场景引擎 ====================

type ScenarioEngine struct {
	mu          sync.RWMutex
	current     string
	previous    string
	overrides   map[string]any
	scenarioCfg map[string]map[string]any
	alarmEngine *AlarmEngine
}

func NewScenarioEngine() *ScenarioEngine {
	return &ScenarioEngine{
		current:     "normal_workday",
		overrides:   make(map[string]any),
		scenarioCfg: make(map[string]map[string]any),
	}
}

// SetAlarmEngine 注入告警引擎，用于场景切换时注入告警
func (e *ScenarioEngine) SetAlarmEngine(ae *AlarmEngine) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.alarmEngine = ae
}

func (e *ScenarioEngine) RegisterScenario(name string, overrides map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.scenarioCfg[name] = overrides
}

func (e *ScenarioEngine) SetScenario(name string) {
	e.mu.Lock()
	e.previous = e.current
	e.current = name
	if overrides, ok := e.scenarioCfg[name]; ok {
		e.overrides = overrides
	} else {
		e.overrides = make(map[string]any)
	}
	alarmEngine := e.alarmEngine
	e.mu.Unlock()

	// 突发场景注入告警
	if alarmEngine != nil {
		e.injectScenarioAlarms(name, alarmEngine)
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

// injectScenarioAlarms 场景切换时注入突发告警
func (e *ScenarioEngine) injectScenarioAlarms(scenario string, ae *AlarmEngine) {
	now := time.Now()
	var alarms []AlarmRecord

	switch scenario {
	case "fire_emergency":
		// 多个烟感触发
		for i := 1; i <= 5; i++ {
			alarms = append(alarms, AlarmRecord{
				DeviceID:  "smoke_detector-B001-" + padZero(i),
				Type:      "smoke_alarm",
				Level:     "critical",
				Message:   "烟感探测器报警",
				Timestamp: now,
				Value:     85.0 + rand.Float64()*15,
			})
		}
		// 温感探测器
		for i := 1; i <= 3; i++ {
			alarms = append(alarms, AlarmRecord{
				DeviceID:  "temp_detector-B001-" + padZero(i),
				Type:      "high_temp_alarm",
				Level:     "critical",
				Message:   "温感探测器超温报警",
				Timestamp: now,
				Value:     68.0 + rand.Float64()*10,
			})
		}
		// 手动报警按钮
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "manual_call_point-B001-001",
				Type:      "manual_alarm",
				Level:     "critical",
				Message:   "手动报警按钮触发",
				Timestamp: now,
			})
		// 消火栓
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "fire_hydrant-B001-001",
				Type:      "hydrant_open",
				Level:     "major",
				Message:   "消火栓被打开",
				Timestamp: now,
			})
		// 喷淋泵启动
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "sprinkler_pump-B001-001",
				Type:      "pump_started",
				Level:     "major",
				Message:   "喷淋泵已启动",
				Timestamp: now,
			})
		// 广播系统
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "emergency_broadcast-B001-001",
				Type:      "emergency_broadcast",
				Level:     "critical",
				Message:   "紧急广播已启动，请立即疏散",
				Timestamp: now,
			})
		log.Printf("[SCENARIO] 消防突发场景已触发，注入 %d 条告警", len(alarms))

	case "intrusion":
		// 红外对射报警
		for i := 1; i <= 4; i++ {
			alarms = append(alarms, AlarmRecord{
				DeviceID:  "infrared_beam-B001-" + padZero(i),
				Type:      "beam_alarm",
				Level:     "major",
				Message:   "周界红外对射报警",
				Timestamp: now,
			})
		}
		// 电子围栏
		for i := 1; i <= 2; i++ {
			alarms = append(alarms, AlarmRecord{
				DeviceID:  "electric_fence-B001-" + padZero(i),
				Type:      "fence_alarm",
				Level:     "major",
				Message:   "电子围栏触警",
				Timestamp: now,
			})
		}
		// 摄像头检测到入侵
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "video_analyzer-B001-001",
				Type:      "intrusion_detected",
				Level:     "major",
				Message:   "视频分析检测到入侵人员",
				Timestamp: now,
			})
		// 门禁异常
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "access_controller-B001-001",
				Type:      "forced_entry",
				Level:     "major",
				Message:   "门禁强制开启报警",
				Timestamp: now,
			})
		log.Printf("[SCENARIO] 安防入侵场景已触发，注入 %d 条告警", len(alarms))

	case "power_outage":
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "power_meter-B001-001",
				Type:      "power_outage",
				Level:     "critical",
				Message:   "市电中断，UPS 已接管",
				Timestamp: now,
			})
		alarms = append(alarms, AlarmRecord{
				DeviceID:  "elevator_controller-B001-001",
				Type:      "elevator_stopped",
				Level:     "major",
				Message:   "电梯停运",
				Timestamp: now,
			})
		log.Printf("[SCENARIO] 停电事件场景已触发，注入 %d 条告警", len(alarms))
	}

	for _, a := range alarms {
		ae.InjectAlarm(a)
	}
}

func padZero(n int) string {
	if n < 10 {
		return "00" + intToStr(n)
	}
	if n < 100 {
		return "0" + intToStr(n)
	}
	return intToStr(n)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [4]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

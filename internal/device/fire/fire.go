package fire

import (
	"time"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 烟感探测器 SmokeDetector ====================

func init() {
	device.RegisterDevice("smoke_detector", NewSmokeDetector)
}

type SmokeDetector struct {
	device.BaseDevice
	batteryLevel float64
	simAlarm     bool // 模拟报警状态
}

func NewSmokeDetector(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &SmokeDetector{
		BaseDevice:   device.NewBaseDevice(id, "smoke_detector", "fire", types.ProtocolMQTT, meta),
		batteryLevel: 100,
		simAlarm:     false,
	}
}

func (s *SmokeDetector) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 电池缓慢消耗
	s.batteryLevel -= 0.01
	if s.batteryLevel < 70 {
		s.batteryLevel = 70 + engine.RandFloat(0, 5)
	}

	// 偶尔模拟报警（极低概率）
	if engine.RandBool(0.002) {
		s.simAlarm = true
	}
	// 报警持续一段时间后恢复
	if s.simAlarm && engine.RandBool(0.3) {
		s.simAlarm = false
	}

	smokeDensity := engine.Clamp(engine.GaussNoise(0.5, 0.3), 0, 5)
	alarm := false
	if s.simAlarm {
		smokeDensity = engine.Clamp(50+engine.GaussNoise(0, 10), 30, 100)
		alarm = true
	}

	fault := engine.RandBool(0.005)

	return map[string]any{
		"smoke_density": smokeDensity,
		"alarm":         alarm,
		"fault":         fault,
		"battery_level": engine.Clamp(s.batteryLevel, 0, 100),
	}
}

func (s *SmokeDetector) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if alarm, ok := data["alarm"].(bool); ok && alarm {
		alarms = append(alarms, types.Alarm{
			DeviceID:  s.ID(),
			Type:      "smoke_alarm",
			Level:     "critical",
			Message:   "烟感探测器报警",
			Timestamp: time.Now(),
			Value:     data["smoke_density"],
		})
	}
	if fault, ok := data["fault"].(bool); ok && fault {
		alarms = append(alarms, types.Alarm{
			DeviceID:  s.ID(),
			Type:      "device_fault",
			Level:     "warning",
			Message:   "烟感探测器设备故障",
			Timestamp: time.Now(),
			Value:     fault,
		})
	}
	if bl, ok := data["battery_level"].(float64); ok && bl < 20 {
		alarms = append(alarms, types.Alarm{
			DeviceID:  s.ID(),
			Type:      "low_battery",
			Level:     "warning",
			Message:   "烟感探测器电池电量低",
			Timestamp: time.Now(),
			Value:     bl,
		})
	}
	return alarms
}

// ==================== 温感探测器 TempDetector ====================

func init() {
	device.RegisterDevice("temp_detector", NewTempDetector)
}

type TempDetector struct {
	device.BaseDevice
	prevTemp float64
	simAlarm bool
}

func NewTempDetector(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &TempDetector{
		BaseDevice: device.NewBaseDevice(id, "temp_detector", "fire", types.ProtocolMQTT, meta),
		prevTemp:   25.0,
		simAlarm:   false,
	}
}

func (t *TempDetector) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 温度与室外温度+室内修正相关
	targetTemp := ctx.OutdoorTemp + 2 // 室内略高于室外
	t.prevTemp = engine.TemperatureInertia(t.prevTemp, targetTemp, 0.95)
	t.prevTemp += engine.GaussNoise(0, 0.3)

	// 偶尔模拟报警
	if engine.RandBool(0.001) {
		t.simAlarm = true
	}
	if t.simAlarm && engine.RandBool(0.3) {
		t.simAlarm = false
	}

	riseRate := engine.Clamp(engine.GaussNoise(0.2, 0.15), 0, 5)
	alarm := false
	if t.simAlarm {
		t.prevTemp = engine.Clamp(t.prevTemp+engine.GaussNoise(5, 2), 50, 120)
		riseRate = engine.Clamp(engine.GaussNoise(8, 2), 5, 20)
		alarm = true
	}

	return map[string]any{
		"temperature": engine.Clamp(t.prevTemp, -20, 150),
		"rise_rate":   riseRate,
		"alarm":       alarm,
	}
}

func (t *TempDetector) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if alarm, ok := data["alarm"].(bool); ok && alarm {
		alarms = append(alarms, types.Alarm{
			DeviceID:  t.ID(),
			Type:      "temp_alarm",
			Level:     "critical",
			Message:   "温感探测器报警",
			Timestamp: time.Now(),
			Value:     data["temperature"],
		})
	}
	if rr, ok := data["rise_rate"].(float64); ok && rr > 5 {
		alarms = append(alarms, types.Alarm{
			DeviceID:  t.ID(),
			Type:      "rapid_temp_rise",
			Level:     "major",
			Message:   "温度上升速率过快",
			Timestamp: time.Now(),
			Value:     rr,
		})
	}
	return alarms
}

// ==================== 手动报警按钮 ManualCallPoint ====================

func init() {
	device.RegisterDevice("manual_call_point", NewManualCallPoint)
}

type ManualCallPoint struct {
	device.BaseDevice
}

func NewManualCallPoint(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &ManualCallPoint{
		BaseDevice: device.NewBaseDevice(id, "manual_call_point", "fire", types.ProtocolHTTP, meta),
	}
}

func (m *ManualCallPoint) GenerateData(ctx types.ScenarioContext) map[string]any {
	triggered := engine.RandBool(0.001) // 极低概率触发

	return map[string]any{
		"triggered": triggered,
	}
}

func (m *ManualCallPoint) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if triggered, ok := data["triggered"].(bool); ok && triggered {
		alarms = append(alarms, types.Alarm{
			DeviceID:  m.ID(),
			Type:      "manual_alarm",
			Level:     "critical",
			Message:   "手动报警按钮被触发",
			Timestamp: time.Now(),
			Value:     triggered,
		})
	}
	return alarms
}

// ==================== 消防栓 FireHydrant ====================

func init() {
	device.RegisterDevice("fire_hydrant", NewFireHydrant)
}

type FireHydrant struct {
	device.BaseDevice
}

func NewFireHydrant(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &FireHydrant{
		BaseDevice: device.NewBaseDevice(id, "fire_hydrant", "fire", types.ProtocolMQTT, meta),
	}
}

func (fh *FireHydrant) GenerateData(ctx types.ScenarioContext) map[string]any {
	pipePressure := engine.Clamp(0.45+engine.GaussNoise(0, 0.03), 0.3, 0.6)
	pressureAlarm := false
	if pipePressure < 0.3 || pipePressure > 0.6 {
		pressureAlarm = true
	}

	return map[string]any{
		"pipe_pressure":  pipePressure,
		"valve_status":   "closed",
		"pressure_alarm": pressureAlarm,
	}
}

func (fh *FireHydrant) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if pa, ok := data["pressure_alarm"].(bool); ok && pa {
		alarms = append(alarms, types.Alarm{
			DeviceID:  fh.ID(),
			Type:      "pressure_alarm",
			Level:     "major",
			Message:   "消防栓管压异常",
			Timestamp: time.Now(),
			Value:     data["pipe_pressure"],
		})
	}
	return alarms
}

// ==================== 喷淋泵 SprinklerPump ====================

func init() {
	device.RegisterDevice("sprinkler_pump", NewSprinklerPump)
}

type SprinklerPump struct {
	device.BaseDevice
}

func NewSprinklerPump(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &SprinklerPump{
		BaseDevice: device.NewBaseDevice(id, "sprinkler_pump", "fire", types.ProtocolMQTT, meta),
	}
}

func (sp *SprinklerPump) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 通常不运行
	running := engine.RandBool(0.01)
	pipePressure := engine.Clamp(0.4+engine.GaussNoise(0, 0.02), 0.3, 0.5)
	flowIndicator := false

	if running {
		pipePressure = engine.Clamp(0.8+engine.GaussNoise(0, 0.05), 0.6, 1.2)
		flowIndicator = true
	}

	return map[string]any{
		"running":        running,
		"pipe_pressure":  pipePressure,
		"flow_indicator": flowIndicator,
	}
}

func (sp *SprinklerPump) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 防火门 FireDoor ====================

func init() {
	device.RegisterDevice("fire_door", NewFireDoor)
}

type FireDoor struct {
	device.BaseDevice
	openCount int
}

func NewFireDoor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &FireDoor{
		BaseDevice: device.NewBaseDevice(id, "fire_door", "fire", types.ProtocolMQTT, meta),
		openCount:  0,
	}
}

func (fd *FireDoor) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 防火门通常关闭，偶尔被打开
	status := "closed"
	if engine.RandBool(0.05) {
		status = "open"
		fd.openCount++
	}

	fault := engine.RandBool(0.005)
	closerStatus := "normal"
	if fault {
		closerStatus = engine.RandChoice([]string{"damaged", "weak", "stuck"})
	}

	return map[string]any{
		"status":        status,
		"fault":         fault,
		"closer_status": closerStatus,
	}
}

func (fd *FireDoor) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if fault, ok := data["fault"].(bool); ok && fault {
		alarms = append(alarms, types.Alarm{
			DeviceID:  fd.ID(),
			Type:      "door_fault",
			Level:     "warning",
			Message:   "防火门故障",
			Timestamp: time.Now(),
			Value:     data["closer_status"],
		})
	}
	if status, ok := data["status"].(string); ok && status == "open" {
		alarms = append(alarms, types.Alarm{
			DeviceID:  fd.ID(),
			Type:      "door_open",
			Level:     "info",
			Message:   "防火门处于打开状态",
			Timestamp: time.Now(),
			Value:     status,
		})
	}
	return alarms
}

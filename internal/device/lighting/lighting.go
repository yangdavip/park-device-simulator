package lighting

import (
	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 回路控制器 LightingCircuit ====================

func init() {
	device.RegisterDevice("lighting_circuit", NewLightingCircuit)
}

type LightingCircuit struct {
	device.BaseDevice
}

func NewLightingCircuit(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &LightingCircuit{
		BaseDevice: device.NewBaseDevice(id, "lighting_circuit", "lighting", types.ProtocolMQTT, meta),
	}
}

func (lc *LightingCircuit) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	// 照明日程：7:00-20:00 开灯
	on := hour >= 7 && hour < 20
	brightness := 0
	if on {
		lux := engine.LuxLevel(hour)
		// 自然光不足时提高亮度
		if lux < 5000 {
			brightness = 80 + int(engine.GaussNoise(0, 5))
		} else if lux < 20000 {
			brightness = 50 + int(engine.GaussNoise(0, 5))
		} else {
			brightness = 20 + int(engine.GaussNoise(0, 5))
		}
	}

	current := engine.Clamp(0.5+float64(brightness)/100*5+engine.GaussNoise(0, 0.1), 0, 6)
	voltage := engine.Clamp(220+engine.GaussNoise(0, 2), 210, 230)

	return map[string]any{
		"on":          on,
		"brightness":  engine.Clamp(float64(brightness), 0, 100),
		"current":     current,
		"voltage":     voltage,
	}
}

func (lc *LightingCircuit) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 照度传感器 LuxSensor ====================

func init() {
	device.RegisterDevice("lux_sensor", NewLuxSensor)
}

type LuxSensor struct {
	device.BaseDevice
}

func NewLuxSensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &LuxSensor{
		BaseDevice: device.NewBaseDevice(id, "lux_sensor", "lighting", types.ProtocolMQTT, meta),
	}
}

func (ls *LuxSensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	// 室内照度 = 室外照度 × 透光率（约 10-20%）
	outdoorLux := engine.LuxLevel(hour)
	indoorLux := outdoorLux * 0.15 + engine.GaussNoise(0, 30)
	if indoorLux < 0 {
		indoorLux = 0
	}
	// 补充照明
	if hour >= 7 && hour < 20 {
		indoorLux += 200 // 室内灯光补充
	}

	return map[string]any{
		"lux": engine.Clamp(indoorLux, 0, 15000),
	}
}

func (ls *LuxSensor) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 灯具控制器 LampController ====================

func init() {
	device.RegisterDevice("lamp_controller", NewLampController)
}

type LampController struct {
	device.BaseDevice
	totalHours float64
}

func NewLampController(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &LampController{
		BaseDevice: device.NewBaseDevice(id, "lamp_controller", "lighting", types.ProtocolMQTT, meta),
		totalHours: 1000 + engine.RandFloat(0, 5000), // 初始已有使用时长
	}
}

func (lc *LampController) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	on := hour >= 7 && hour < 20
	brightness := 0.0
	if on {
		brightness = 80 + engine.GaussNoise(0, 5)
	}

	if on {
		lc.totalHours += 1.0 / 60.0 // 60s 间隔
	}

	// 故障概率随使用时长增加
	fault := engine.RandBool(0.0005 * (lc.totalHours / 5000))

	return map[string]any{
		"on":          on,
		"brightness":  engine.Clamp(brightness, 0, 100),
		"total_hours": lc.totalHours,
		"fault":       fault,
	}
}

func (lc *LampController) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if f, ok := data["fault"].(bool); ok && f {
		alarms = append(alarms, types.Alarm{
			DeviceID: lc.ID(), Type: "lamp_fault", Level: "warning",
			Message: "灯具故障",
		})
	}
	return alarms
}

// ==================== 场景面板 ScenePanel ====================

func init() {
	device.RegisterDevice("scene_panel", NewScenePanel)
}

type ScenePanel struct {
	device.BaseDevice
	currentScene string
}

func NewScenePanel(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &ScenePanel{
		BaseDevice:   device.NewBaseDevice(id, "scene_panel", "lighting", types.ProtocolMQTT, meta),
		currentScene: "normal",
	}
}

func (sp *ScenePanel) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	// 根据时间段自动切换场景
	switch {
	case hour >= 7 && hour < 9:
		sp.currentScene = "morning"
	case hour >= 9 && hour < 18:
		sp.currentScene = "work"
	case hour >= 18 && hour < 22:
		sp.currentScene = "evening"
	default:
		sp.currentScene = "night"
	}

	// 偶尔有按键事件
	keyEvent := ""
	if engine.RandBool(0.1) {
		keyEvent = engine.RandChoice([]string{"scene_1", "scene_2", "scene_3", "off"})
	}

	return map[string]any{
		"current_scene": sp.currentScene,
		"key_event":     keyEvent,
	}
}

func (sp *ScenePanel) CheckAlarms(data map[string]any) []types.Alarm { return nil }

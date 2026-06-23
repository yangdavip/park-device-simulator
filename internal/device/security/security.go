package security

import (
	"time"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 网络摄像机 IPCamera ====================

func init() {
	device.RegisterDevice("ip_camera", NewIPCamera)
}

type IPCamera struct {
	device.BaseDevice
}

func NewIPCamera(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &IPCamera{
		BaseDevice: device.NewBaseDevice(id, "ip_camera", "security", types.ProtocolMQTT, meta),
	}
}

func (c *IPCamera) GenerateData(ctx types.ScenarioContext) map[string]any {
	bitrate := 2048 + engine.GaussNoise(0, 100)
	motionDetect := engine.RandBool(0.05) // 低概率触发
	faceCaptureEvent := ""
	if engine.RandBool(0.02) {
		faceCaptureEvent = engine.RandChoice([]string{"face_captured", "face_matched", "stranger_alert"})
	}

	return map[string]any{
		"online":             true,
		"bitrate":            engine.Clamp(bitrate, 500, 8000),
		"resolution":         "1080p",
		"motion_detect":      motionDetect,
		"face_capture_event": faceCaptureEvent,
	}
}

func (c *IPCamera) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if online, ok := data["online"].(bool); ok && !online {
		alarms = append(alarms, types.Alarm{
			DeviceID:  c.ID(),
			Type:      "camera_offline",
			Level:     "major",
			Message:   "网络摄像机离线",
			Timestamp: time.Now(),
			Value:     online,
		})
	}
	return alarms
}

// ==================== 球机 PTZCamera ====================

func init() {
	device.RegisterDevice("ptz_camera", NewPTZCamera)
}

type PTZCamera struct {
	device.BaseDevice
	panAngle  float64
	tiltAngle float64
	zoom      float64
	preset    int
}

func NewPTZCamera(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &PTZCamera{
		BaseDevice: device.NewBaseDevice(id, "ptz_camera", "security", types.ProtocolMQTT, meta),
		panAngle:   0,
		tiltAngle:  45,
		zoom:       1,
		preset:     0,
	}
}

func (c *PTZCamera) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 缓慢变化
	c.panAngle += engine.GaussNoise(0, 2)
	c.tiltAngle += engine.GaussNoise(0, 1)
	c.zoom += engine.GaussNoise(0, 0.1)

	c.panAngle = engine.Clamp(c.panAngle, 0, 360)
	c.tiltAngle = engine.Clamp(c.tiltAngle, 0, 90)
	c.zoom = engine.Clamp(c.zoom, 1, 20)

	// 偶尔切换预置位
	if engine.RandBool(0.03) {
		c.preset = engine.RandInt(0, 8)
	}

	return map[string]any{
		"online":     true,
		"pan_angle":  c.panAngle,
		"tilt_angle": c.tiltAngle,
		"zoom":       c.zoom,
		"preset":     c.preset,
	}
}

func (c *PTZCamera) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 视频分析盒子 VideoAnalyzer ====================

func init() {
	device.RegisterDevice("video_analyzer", NewVideoAnalyzer)
}

type VideoAnalyzer struct {
	device.BaseDevice
	peopleCount  int
	vehicleCount int
}

func NewVideoAnalyzer(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &VideoAnalyzer{
		BaseDevice:   device.NewBaseDevice(id, "video_analyzer", "security", types.ProtocolMQTT, meta),
		peopleCount:  0,
		vehicleCount: 0,
	}
}

func (va *VideoAnalyzer) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 人数与入住率正相关
	basePeople := int(ctx.OccupancyRate * 200)
	va.peopleCount = basePeople + engine.RandInt(-10, 10)
	if va.peopleCount < 0 {
		va.peopleCount = 0
	}

	// 车辆数与入住率正相关（停车场视角）
	baseVehicles := int(ctx.OccupancyRate * 80)
	va.vehicleCount = baseVehicles + engine.RandInt(-5, 5)
	if va.vehicleCount < 0 {
		va.vehicleCount = 0
	}

	anomalyEvent := ""
	if engine.RandBool(0.01) {
		anomalyEvent = engine.RandChoice([]string{"loitering", "fall", "crowd"})
	}

	return map[string]any{
		"people_count":  va.peopleCount,
		"vehicle_count": va.vehicleCount,
		"anomaly_event": anomalyEvent,
	}
}

func (va *VideoAnalyzer) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if event, ok := data["anomaly_event"].(string); ok && event != "" {
		alarms = append(alarms, types.Alarm{
			DeviceID:  va.ID(),
			Type:      "anomaly_detected",
			Level:     "warning",
			Message:   "视频分析检测到异常事件: " + event,
			Timestamp: time.Now(),
			Value:     event,
		})
	}
	return alarms
}

// ==================== 周界红外对射 InfraredBeam ====================

func init() {
	device.RegisterDevice("infrared_beam", NewInfraredBeam)
}

type InfraredBeam struct {
	device.BaseDevice
}

func NewInfraredBeam(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &InfraredBeam{
		BaseDevice: device.NewBaseDevice(id, "infrared_beam", "security", types.ProtocolHTTP, meta),
	}
}

func (ib *InfraredBeam) GenerateData(ctx types.ScenarioContext) map[string]any {
	alarm := engine.RandBool(0.02) // 低概率触发
	signalStrength := engine.Clamp(90+engine.GaussNoise(0, 2), 85, 95)

	return map[string]any{
		"alarm":           alarm,
		"signal_strength": signalStrength,
	}
}

func (ib *InfraredBeam) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if alarm, ok := data["alarm"].(bool); ok && alarm {
		alarms = append(alarms, types.Alarm{
			DeviceID:  ib.ID(),
			Type:      "beam_alarm",
			Level:     "major",
			Message:   "周界红外对射报警",
			Timestamp: time.Now(),
			Value:     alarm,
		})
	}
	return alarms
}

// ==================== 电子围栏 ElectricFence ====================

func init() {
	device.RegisterDevice("electric_fence", NewElectricFence)
}

type ElectricFence struct {
	device.BaseDevice
}

func NewElectricFence(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &ElectricFence{
		BaseDevice: device.NewBaseDevice(id, "electric_fence", "security", types.ProtocolMQTT, meta),
	}
}

func (ef *ElectricFence) GenerateData(ctx types.ScenarioContext) map[string]any {
	voltage := engine.Clamp(9500+engine.GaussNoise(0, 200), 9000, 10000)
	alarm := engine.RandBool(0.02) // 低概率触发
	shortCircuit := engine.RandBool(0.005)

	return map[string]any{
		"alarm":         alarm,
		"voltage":       voltage,
		"short_circuit": shortCircuit,
	}
}

func (ef *ElectricFence) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if alarm, ok := data["alarm"].(bool); ok && alarm {
		alarms = append(alarms, types.Alarm{
			DeviceID:  ef.ID(),
			Type:      "fence_alarm",
			Level:     "major",
			Message:   "电子围栏报警",
			Timestamp: time.Now(),
			Value:     alarm,
		})
	}
	if sc, ok := data["short_circuit"].(bool); ok && sc {
		alarms = append(alarms, types.Alarm{
			DeviceID:  ef.ID(),
			Type:      "short_circuit",
			Level:     "critical",
			Message:   "电子围栏短路",
			Timestamp: time.Now(),
			Value:     sc,
		})
	}
	return alarms
}

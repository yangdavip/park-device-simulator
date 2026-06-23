package parking

import (
	"fmt"
	"time"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 车牌识别相机 LPRCamera ====================

func init() {
	device.RegisterDevice("lpr_camera", NewLPRCamera)
}

type LPRCamera struct {
	device.BaseDevice
}

func NewLPRCamera(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &LPRCamera{
		BaseDevice: device.NewBaseDevice(id, "lpr_camera", "parking", types.ProtocolHTTP, meta),
	}
}

// generatePlate 生成随机车牌号
func generatePlate() string {
	provinces := []string{"京", "沪", "粤", "浙", "苏", "川", "鲁", "豫", "鄂", "湘"}
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "J", "K"}
	plate := engine.RandChoice(provinces) + engine.RandChoice(letters)
	plate += fmt.Sprintf("%d%02d", engine.RandInt(1000, 9999), engine.RandInt(10, 99))
	return plate
}

func (c *LPRCamera) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 工作时间频率较高，非工作时间偶有车辆进出
	recognize := false
	direction := "in"
	passEvent := false

	hour := ctx.Timestamp.Hour()
	if ctx.IsWorkday && (hour >= 7 && hour <= 20) {
		recognize = engine.RandBool(0.7)
	} else {
		recognize = engine.RandBool(0.15)
	}

	if recognize {
		direction = engine.RandChoice([]string{"in", "out"})
		passEvent = engine.RandBool(0.85) // 85% 概率放行
	}

	return map[string]any{
		"plate_number":   generatePlate(),
		"direction":      direction,
		"recognize_time": ctx.Timestamp.Format(time.RFC3339),
		"pass_event":     passEvent,
	}
}

func (c *LPRCamera) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 地磁传感器 Geomagnetic ====================

func init() {
	device.RegisterDevice("geomagnetic", NewGeomagnetic)
}

type Geomagnetic struct {
	device.BaseDevice
	occupied bool
}

func NewGeomagnetic(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &Geomagnetic{
		BaseDevice: device.NewBaseDevice(id, "geomagnetic", "parking", types.ProtocolMQTT, meta),
		occupied:   engine.RandBool(0.5),
	}
}

func (g *Geomagnetic) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 40-70% 概率占用，与入住率正相关
	occupyProb := 0.4 + ctx.OccupancyRate*0.3
	g.occupied = engine.RandBool(occupyProb)

	// 信号强度 70-90 dB
	signal := engine.Clamp(engine.GaussNoise(80, 5), 70, 90)

	return map[string]any{
		"occupied":        g.occupied,
		"signal_strength": signal,
	}
}

func (g *Geomagnetic) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if sig, ok := data["signal_strength"].(float64); ok && sig < 72 {
		alarms = append(alarms, types.Alarm{
			DeviceID: g.ID(), Type: "low_signal", Level: "warning",
			Message: "地磁传感器信号强度低", Value: sig,
		})
	}
	return alarms
}

// ==================== 超声波车位探测器 UltrasonicSensor ====================

func init() {
	device.RegisterDevice("ultrasonic_sensor", NewUltrasonicSensor)
}

type UltrasonicSensor struct {
	device.BaseDevice
	occupied bool
}

func NewUltrasonicSensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &UltrasonicSensor{
		BaseDevice: device.NewBaseDevice(id, "ultrasonic_sensor", "parking", types.ProtocolMQTT, meta),
		occupied:   engine.RandBool(0.5),
	}
}

func (u *UltrasonicSensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	occupyProb := 0.4 + ctx.OccupancyRate*0.3
	u.occupied = engine.RandBool(occupyProb)

	var distance float64
	if u.occupied {
		// 占用时 30-80cm
		distance = engine.RandFloat(30, 80)
	} else {
		// 空闲时 150-250cm
		distance = engine.RandFloat(150, 250)
	}

	return map[string]any{
		"occupied": u.occupied,
		"distance": distance,
	}
}

func (u *UltrasonicSensor) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 车位引导屏 GuideScreen ====================

func init() {
	device.RegisterDevice("guide_screen", NewGuideScreen)
}

type GuideScreen struct {
	device.BaseDevice
	totalSpaces int
	remaining   int
}

func NewGuideScreen(id string, meta map[string]any, cfg map[string]any) device.Device {
	totalSpaces := 200
	if v, ok := cfg["total_spaces"]; ok {
		if f, ok := v.(float64); ok {
			totalSpaces = int(f)
		}
	}
	return &GuideScreen{
		BaseDevice:  device.NewBaseDevice(id, "guide_screen", "parking", types.ProtocolMQTT, meta),
		totalSpaces: totalSpaces,
		remaining:   totalSpaces,
	}
}

func (g *GuideScreen) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 根据入住率估算已占用车位
	occupyRate := 0.3 + ctx.OccupancyRate*0.5
	occupied := int(float64(g.totalSpaces) * occupyRate)
	g.remaining = g.totalSpaces - occupied

	displayContent := fmt.Sprintf("剩余车位: %d", g.remaining)
	if g.remaining < 10 {
		displayContent = "车位已满"
	}

	return map[string]any{
		"remaining":       g.remaining,
		"display_content": displayContent,
		"online":          true,
	}
}

func (g *GuideScreen) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 充电桩 ChargingPile ====================

func init() {
	device.RegisterDevice("charging_pile", NewChargingPile)
}

type ChargingPile struct {
	device.BaseDevice
	soc          float64 // 电池 SOC 0-100
	status       string  // idle / charging / fault
	energy       float64 // 累计充电量 kWh
	ratedPower   float64 // 额定功率 kW
	capacity     float64 // 电池容量 kWh
}

func NewChargingPile(id string, meta map[string]any, cfg map[string]any) device.Device {
	ratedPower := 120.0
	if v, ok := cfg["rated_power"]; ok {
		if f, ok := v.(float64); ok {
			ratedPower = f
		}
	}
	return &ChargingPile{
		BaseDevice: device.NewBaseDevice(id, "charging_pile", "parking", types.ProtocolMQTT, meta),
		soc:        0,
		status:     "idle",
		energy:     0,
		ratedPower: ratedPower,
	}
}

func (cp *ChargingPile) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 状态流转逻辑
	switch cp.status {
	case "idle":
		// 工作时间有较高概率开始充电
		hour := ctx.Timestamp.Hour()
		if (hour >= 8 && hour <= 20) && engine.RandBool(0.3) {
			cp.status = "charging"
			cp.soc = engine.RandFloat(10, 50) // 开始充电时的初始 SOC
		}
		// 极低概率故障
		if engine.RandBool(0.001) {
			cp.status = "fault"
		}
	case "charging":
		// 充电中：使用 BatterySOC 模拟充电过程（15s 间隔 = 0.25 min）
		cp.soc = engine.BatterySOC(cp.soc, cp.ratedPower, 100.0, 0.25)
		if cp.soc >= 100 {
			cp.status = "idle"
			cp.soc = 0
		}
		// 极低概率故障
		if engine.RandBool(0.002) {
			cp.status = "fault"
		}
	case "fault":
		// 故障后有概率恢复
		if engine.RandBool(0.1) {
			cp.status = "idle"
		}
	}

	var power, voltage, current, temperature float64

	if cp.status == "charging" {
		power = cp.ratedPower
		// 80% 后功率递减
		if cp.soc >= 80 {
			power = cp.ratedPower * 0.4
		}
		voltage = 220.0 + engine.GaussNoise(0, 1)
		current = power / voltage * 1000 // mA → A
		current = engine.Clamp(current, 0, 32)
		temperature = engine.Clamp(35+power/cp.ratedPower*15+engine.GaussNoise(0, 2), 25, 50)
		cp.energy += power * 15.0 / 3600.0 // 15s 间隔累计 kWh
	} else if cp.status == "idle" {
		power = 0
		voltage = 220.0 + engine.GaussNoise(0, 1)
		current = 0
		temperature = engine.Clamp(25+engine.GaussNoise(0, 1), 20, 35)
	} else {
		// fault
		power = 0
		voltage = 0
		current = 0
		temperature = engine.Clamp(25+engine.GaussNoise(0, 1), 20, 35)
	}

	return map[string]any{
		"charging_status": cp.status,
		"power":           power,
		"energy":          cp.energy,
		"voltage":         voltage,
		"current":         current,
		"temperature":     temperature,
		"soc":             cp.soc,
	}
}

func (cp *ChargingPile) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if status, ok := data["charging_status"].(string); ok && status == "fault" {
		alarms = append(alarms, types.Alarm{
			DeviceID: cp.ID(), Type: "charging_fault", Level: "major",
			Message: "充电桩故障",
		})
	}
	if temp, ok := data["temperature"].(float64); ok && temp > 48 {
		alarms = append(alarms, types.Alarm{
			DeviceID: cp.ID(), Type: "high_temperature", Level: "warning",
			Message: "充电桩温度过高", Value: temp,
		})
	}
	return alarms
}

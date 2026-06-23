package bas

import (
	"math"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== AHU 空调主机 ====================

func init() {
	device.RegisterDevice("ahu", NewAHU)
}

type AHU struct {
	device.BaseDevice
	prevSupplyTemp  float64
	prevReturnTemp  float64
	setTemp         float64
}

func NewAHU(id string, meta map[string]any, cfg map[string]any) device.Device {
	setTemp := 24.0
	if v, ok := cfg["set_temp"]; ok {
		if f, ok := v.(float64); ok {
			setTemp = f
		}
	}
	return &AHU{
		BaseDevice:      device.NewBaseDevice(id, "ahu", "bas", types.ProtocolMQTT, meta),
		prevSupplyTemp:  18.0,
		prevReturnTemp:  24.0,
		setTemp:         setTemp,
	}
}

func (a *AHU) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 送风温度：空调运行时 16-20°C，停机时接近回风温度
	targetSupply := 18.0
	if ctx.OutdoorTemp > 28 {
		targetSupply = 16.0 // 夏天更冷
	} else if ctx.OutdoorTemp < 10 {
		targetSupply = 28.0 // 冬天供暖
	}
	a.prevSupplyTemp = engine.TemperatureInertia(a.prevSupplyTemp, targetSupply, 0.92)
	a.prevSupplyTemp += engine.GaussNoise(0, 0.2)

	// 回风温度：受室内温度和设定温度影响
	targetReturn := a.setTemp + (ctx.OutdoorTemp-a.setTemp)*0.1
	a.prevReturnTemp = engine.TemperatureInertia(a.prevReturnTemp, targetReturn, 0.95)
	a.prevReturnTemp += engine.GaussNoise(0, 0.3)

	running := true
	if ctx.OccupancyRate < 0.1 && !ctx.IsWorkday {
		running = false
	}

	fanSpeed := 3
	if ctx.OccupancyRate > 0.8 {
		fanSpeed = 5
	} else if ctx.OccupancyRate > 0.5 {
		fanSpeed = 4
	} else if ctx.OccupancyRate < 0.2 {
		fanSpeed = 2
	}

	return map[string]any{
		"supply_temp": engine.Clamp(a.prevSupplyTemp, 5, 40),
		"return_temp": engine.Clamp(a.prevReturnTemp, 10, 40),
		"running":     running,
		"set_temp":    a.setTemp,
		"fan_speed":   fanSpeed,
	}
}

func (a *AHU) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if rt, ok := data["return_temp"].(float64); ok && rt > 30 {
		alarms = append(alarms, types.Alarm{
			DeviceID: a.ID(),
			Type:     "high_return_temp",
			Level:    "warning",
			Message:  "回风温度过高",
			Value:    rt,
		})
	}
	return alarms
}

// ==================== FCU 风机盘管 ====================

func init() {
	device.RegisterDevice("fcu", NewFCU)
}

type FCU struct {
	device.BaseDevice
	prevRoomTemp float64
	setTemp      float64
}

func NewFCU(id string, meta map[string]any, cfg map[string]any) device.Device {
	setTemp := 24.0
	if v, ok := cfg["set_temp"]; ok {
		if f, ok := v.(float64); ok {
			setTemp = f
		}
	}
	return &FCU{
		BaseDevice:   device.NewBaseDevice(id, "fcu", "bas", types.ProtocolMQTT, meta),
		prevRoomTemp: 24.0,
		setTemp:      setTemp,
	}
}

func (f *FCU) GenerateData(ctx types.ScenarioContext) map[string]any {
	targetTemp := f.setTemp + (ctx.OutdoorTemp-f.setTemp)*0.08
	f.prevRoomTemp = engine.TemperatureInertia(f.prevRoomTemp, targetTemp, 0.93)

	running := ctx.OccupancyRate > 0.05
	valveOpen := 0.0
	fanSpeed := 0

	if running {
		diff := math.Abs(f.prevRoomTemp - f.setTemp)
		valveOpen = engine.Clamp(30+diff*20+engine.GaussNoise(0, 5), 0, 100)
		if diff > 2 {
			fanSpeed = 3
		} else if diff > 1 {
			fanSpeed = 2
		} else {
			fanSpeed = 1
		}
	}

	return map[string]any{
		"room_temp":  engine.Clamp(f.prevRoomTemp, 10, 40),
		"valve_open": engine.Clamp(valveOpen, 0, 100),
		"fan_speed":  fanSpeed,
		"running":    running,
	}
}

func (f *FCU) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if rt, ok := data["room_temp"].(float64); ok && rt > 32 {
		alarms = append(alarms, types.Alarm{
			DeviceID: f.ID(),
			Type:     "high_room_temp",
			Level:    "warning",
			Message:  "房间温度过高",
			Value:    rt,
		})
	}
	return alarms
}

// ==================== 新风机组 FAU ====================

func init() {
	device.RegisterDevice("fau", NewFAU)
}

type FAU struct {
	device.BaseDevice
	prevSupplyTemp float64
}

func NewFAU(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &FAU{
		BaseDevice:     device.NewBaseDevice(id, "fau", "bas", types.ProtocolMQTT, meta),
		prevSupplyTemp: 20.0,
	}
}

func (f *FAU) GenerateData(ctx types.ScenarioContext) map[string]any {
	targetSupply := 20.0 + (ctx.OutdoorTemp-20)*0.3
	f.prevSupplyTemp = engine.TemperatureInertia(f.prevSupplyTemp, targetSupply, 0.9)

	freshTemp := ctx.OutdoorTemp + engine.GaussNoise(0, 0.5)
	var freshHumidity float64
	if ctx.Season == "summer" {
		freshHumidity = 70 + engine.GaussNoise(0, 5)
	} else if ctx.Season == "winter" {
		freshHumidity = 35 + engine.GaussNoise(0, 5)
	} else {
		freshHumidity = 55 + engine.GaussNoise(0, 5)
	}

	filterPressure := engine.Clamp(80+engine.GaussNoise(0, 5), 50, 200)

	return map[string]any{
		"fresh_temp":      engine.Clamp(freshTemp, -20, 50),
		"fresh_humidity":  engine.Clamp(freshHumidity, 10, 100),
		"supply_temp":     engine.Clamp(f.prevSupplyTemp, 5, 40),
		"filter_pressure": filterPressure,
		"running":         ctx.OccupancyRate > 0.1,
	}
}

func (f *FAU) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if fp, ok := data["filter_pressure"].(float64); ok && fp > 150 {
		alarms = append(alarms, types.Alarm{
			DeviceID: f.ID(),
			Type:     "filter_clogged",
			Level:    "warning",
			Message:  "滤网压差过大，需清洗或更换",
			Value:    fp,
		})
	}
	return alarms
}

// ==================== 冷水机组 Chiller ====================

func init() {
	device.RegisterDevice("chiller", NewChiller)
}

type Chiller struct {
	device.BaseDevice
	ratedPower float64
}

func NewChiller(id string, meta map[string]any, cfg map[string]any) device.Device {
	ratedPower := 500.0
	if v, ok := cfg["rated_power"]; ok {
		if f, ok := v.(float64); ok {
			ratedPower = f
		}
	}
	return &Chiller{
		BaseDevice:  device.NewBaseDevice(id, "chiller", "bas", types.ProtocolModbus, meta),
		ratedPower:  ratedPower,
	}
}

func (c *Chiller) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 冷冻水供水 7°C，回水 12°C
	chwSupply := 7.0 + engine.GaussNoise(0, 0.3)
	chwReturn := 12.0 + engine.GaussNoise(0, 0.5)

	// 冷却水供水 32°C，回水 37°C
	cwSupply := 32.0 + (ctx.OutdoorTemp-25)*0.3 + engine.GaussNoise(0, 0.5)
	cwReturn := cwSupply + 5 + engine.GaussNoise(0, 0.3)

	// 负载率与室外温度和入住率相关
	loadRate := engine.Clamp(0.3+ctx.PowerLoadRate*0.6+(ctx.OutdoorTemp-25)*0.02, 0.1, 1.0)
	power := c.ratedPower * loadRate + c.ratedPower*0.05 // 空载损耗

	return map[string]any{
		"chw_supply_temp": engine.Clamp(chwSupply, 3, 15),
		"chw_return_temp": engine.Clamp(chwReturn, 8, 20),
		"cw_supply_temp":  engine.Clamp(cwSupply, 20, 45),
		"cw_return_temp":  engine.Clamp(cwReturn, 25, 50),
		"power":           engine.Clamp(power, 0, c.ratedPower*1.1),
		"running":         true,
		"load_rate":       engine.Clamp(loadRate*100, 0, 100),
	}
}

func (c *Chiller) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if t, ok := data["chw_supply_temp"].(float64); ok && t > 12 {
		alarms = append(alarms, types.Alarm{
			DeviceID: c.ID(),
			Type:     "high_chw_supply_temp",
			Level:    "warning",
			Message:  "冷冻水供水温度过高",
			Value:    t,
		})
	}
	return alarms
}

// ==================== 冷却塔 CoolingTower ====================

func init() {
	device.RegisterDevice("cooling_tower", NewCoolingTower)
}

type CoolingTower struct {
	device.BaseDevice
}

func NewCoolingTower(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &CoolingTower{
		BaseDevice: device.NewBaseDevice(id, "cooling_tower", "bas", types.ProtocolMQTT, meta),
	}
}

func (ct *CoolingTower) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 出水温度接近室外湿球温度（约比室外温度低 5-8°C）
	outletTemp := ctx.OutdoorTemp - 6 + engine.GaussNoise(0, 0.5)
	fanRPM := 0
	running := ctx.OccupancyRate > 0.1
	if running {
		fanRPM = int(engine.Clamp(800+ctx.PowerLoadRate*1200+engine.GaussNoise(0, 50), 0, 2000))
	}

	return map[string]any{
		"outlet_temp": engine.Clamp(outletTemp, 15, 40),
		"fan_rpm":     fanRPM,
		"running":     running,
	}
}

func (ct *CoolingTower) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 水泵 Pump ====================

func init() {
	device.RegisterDevice("pump", NewPump)
}

type Pump struct {
	device.BaseDevice
	ratedPower  float64
	ratedFlow   float64
}

func NewPump(id string, meta map[string]any, cfg map[string]any) device.Device {
	ratedPower := 22.0
	ratedFlow := 100.0
	if v, ok := cfg["rated_power"]; ok {
		if f, ok := v.(float64); ok {
			ratedPower = f
		}
	}
	if v, ok := cfg["rated_flow"]; ok {
		if f, ok := v.(float64); ok {
			ratedFlow = f
		}
	}
	return &Pump{
		BaseDevice:  device.NewBaseDevice(id, "pump", "bas", types.ProtocolMQTT, meta),
		ratedPower:  ratedPower,
		ratedFlow:   ratedFlow,
	}
}

func (p *Pump) GenerateData(ctx types.ScenarioContext) map[string]any {
	running := ctx.OccupancyRate > 0.1
	flow := 0.0
	power := 0.0
	freq := 0.0
	head := 0.0

	if running {
		loadRate := engine.Clamp(ctx.PowerLoadRate+engine.GaussNoise(0, 0.05), 0.3, 1.0)
		freq = 30 + loadRate*20 // 30-50 Hz
		flow = p.ratedFlow * loadRate
		power = p.ratedPower * loadRate * 0.9 + p.ratedPower * 0.1
		head = 30 + loadRate*10 + engine.GaussNoise(0, 1)
	}

	return map[string]any{
		"flow":  engine.Clamp(flow, 0, p.ratedFlow*1.2),
		"head":  engine.Clamp(head, 0, 50),
		"power": engine.Clamp(power, 0, p.ratedPower*1.2),
		"running": running,
		"freq":  engine.Clamp(freq, 0, 50),
	}
}

func (p *Pump) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 给排水水箱 WaterTank ====================

func init() {
	device.RegisterDevice("water_tank", NewWaterTank)
}

type WaterTank struct {
	device.BaseDevice
	level float64
}

func NewWaterTank(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &WaterTank{
		BaseDevice: device.NewBaseDevice(id, "water_tank", "bas", types.ProtocolMQTT, meta),
		level:      2.0,
	}
}

func (w *WaterTank) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 水位在 0.5-3.5m 之间波动
	delta := engine.GaussNoise(0, 0.2)
	// 白天用水多，水位下降趋势
	if ctx.OccupancyRate > 0.5 {
		delta -= 0.1
	}
	w.level += delta
	w.level = engine.Clamp(w.level, 0.3, 3.8)

	highAlarm := w.level > 3.5
	lowAlarm := w.level < 0.5

	return map[string]any{
		"level":      w.level,
		"temp":       engine.Clamp(ctx.OutdoorTemp-2+engine.GaussNoise(0, 1), 5, 40),
		"high_alarm": highAlarm,
		"low_alarm":  lowAlarm,
	}
}

func (w *WaterTank) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if hi, ok := data["high_alarm"].(bool); ok && hi {
		alarms = append(alarms, types.Alarm{
			DeviceID: w.ID(), Type: "high_level", Level: "major", Message: "水箱高液位报警",
		})
	}
	if lo, ok := data["low_alarm"].(bool); ok && lo {
		alarms = append(alarms, types.Alarm{
			DeviceID: w.ID(), Type: "low_level", Level: "major", Message: "水箱低液位报警",
		})
	}
	return alarms
}

// ==================== 送排风机 VentFan ====================

func init() {
	device.RegisterDevice("vent_fan", NewVentFan)
}

type VentFan struct {
	device.BaseDevice
}

func NewVentFan(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &VentFan{
		BaseDevice: device.NewBaseDevice(id, "vent_fan", "bas", types.ProtocolMQTT, meta),
	}
}

func (v *VentFan) GenerateData(ctx types.ScenarioContext) map[string]any {
	running := ctx.OccupancyRate > 0.1
	flow := 0.0
	pressure := 0.0
	freq := 0.0

	if running {
		loadRate := engine.Clamp(ctx.PowerLoadRate+engine.GaussNoise(0, 0.05), 0.3, 1.0)
		freq = 30 + loadRate*20
		flow = 5000 * loadRate
		pressure = 200 + loadRate*150 + engine.GaussNoise(0, 10)
	}

	return map[string]any{
		"flow":      engine.Clamp(flow, 0, 6000),
		"pressure":  engine.Clamp(pressure, 0, 500),
		"running":   running,
		"freq":      engine.Clamp(freq, 0, 50),
	}
}

func (v *VentFan) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 热交换器 HeatExchanger ====================

func init() {
	device.RegisterDevice("heat_exchanger", NewHeatExchanger)
}

type HeatExchanger struct {
	device.BaseDevice
}

func NewHeatExchanger(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &HeatExchanger{
		BaseDevice: device.NewBaseDevice(id, "heat_exchanger", "bas", types.ProtocolMQTT, meta),
	}
}

func (h *HeatExchanger) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 一次侧：高温热水 60/50°C
	priSupply := 60 + engine.GaussNoise(0, 1)
	priReturn := 50 + engine.GaussNoise(0, 1)
	// 二次侧：供暖水 45/38°C
	secSupply := 45 + engine.GaussNoise(0, 0.5)
	secReturn := 38 + engine.GaussNoise(0, 0.5)
	flow := 20 + ctx.PowerLoadRate*30 + engine.GaussNoise(0, 2)

	return map[string]any{
		"pri_supply_temp": engine.Clamp(priSupply, 40, 80),
		"pri_return_temp": engine.Clamp(priReturn, 30, 70),
		"sec_supply_temp": engine.Clamp(secSupply, 30, 60),
		"sec_return_temp": engine.Clamp(secReturn, 20, 50),
		"flow":            engine.Clamp(flow, 0, 60),
	}
}

func (h *HeatExchanger) CheckAlarms(data map[string]any) []types.Alarm { return nil }

package energy

import (
	"math"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 电力仪表 PowerMeter ====================

func init() {
	device.RegisterDevice("power_meter", NewPowerMeter)
}

type PowerMeter struct {
	device.BaseDevice
	ratedPower  float64
	ratedVoltage float64
	ratedCurrent float64
	totalEnergy  float64 // 累计电度 kWh
}

func NewPowerMeter(id string, meta map[string]any, cfg map[string]any) device.Device {
	ratedPower := 100.0
	ratedVoltage := 220.0
	ratedCurrent := 200.0
	if v, ok := cfg["rated_power"]; ok {
		if f, ok := v.(float64); ok {
			ratedPower = f
		}
	}
	return &PowerMeter{
		BaseDevice:   device.NewBaseDevice(id, "power_meter", "energy", types.ProtocolModbus, meta),
		ratedPower:   ratedPower,
		ratedVoltage: ratedVoltage,
		ratedCurrent: ratedCurrent,
		totalEnergy:  0,
	}
}

func (pm *PowerMeter) GenerateData(ctx types.ScenarioContext) map[string]any {
	loadRate := engine.Clamp(ctx.PowerLoadRate+engine.GaussNoise(0, 0.03), 0.05, 1.0)

	// 三相电压（380V 系统，相电压 220V）
	va, vb, vc := engine.ThreePhaseVoltage(pm.ratedVoltage)
	// 三相电流
	ia, ib, ic := engine.ThreePhaseCurrent(pm.ratedCurrent, loadRate)

	// 有功功率 = √3 × U × I × cos(φ) (三相)
	pf := engine.PowerFactor(loadRate)
	totalCurrent := (ia + ib + ic) / 3
	activePower := math.Sqrt(3) * 380 * totalCurrent * pf / 1000 // kW
	reactivePower := activePower * math.Sqrt(1-pf*pf) / pf        // kVar

	// 累计电度（按 60s 间隔，1/60 小时）
	pm.totalEnergy += activePower / 60.0

	freq := 50.0 + engine.GaussNoise(0, 0.05)

	return map[string]any{
		"voltage_a":     engine.Clamp(va, 180, 250),
		"voltage_b":     engine.Clamp(vb, 180, 250),
		"voltage_c":     engine.Clamp(vc, 180, 250),
		"current_a":     engine.Clamp(ia, 0, pm.ratedCurrent*1.1),
		"current_b":     engine.Clamp(ib, 0, pm.ratedCurrent*1.1),
		"current_c":     engine.Clamp(ic, 0, pm.ratedCurrent*1.1),
		"active_power":  engine.Clamp(activePower, 0, pm.ratedPower*1.2),
		"reactive_power": engine.Clamp(reactivePower, 0, pm.ratedPower),
		"power_factor":  engine.Clamp(pf, 0.5, 0.99),
		"energy":        pm.totalEnergy,
		"frequency":     engine.Clamp(freq, 49.5, 50.5),
	}
}

func (pm *PowerMeter) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if v, ok := data["voltage_a"].(float64); ok && (v < 198 || v > 242) {
		alarms = append(alarms, types.Alarm{
			DeviceID: pm.ID(), Type: "voltage_abnormal", Level: "warning",
			Message: "A相电压异常", Value: v,
		})
	}
	return alarms
}

// ==================== 水表 WaterMeter ====================

func init() {
	device.RegisterDevice("water_meter", NewWaterMeter)
}

type WaterMeter struct {
	device.BaseDevice
	totalFlow float64
}

func NewWaterMeter(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &WaterMeter{
		BaseDevice: device.NewBaseDevice(id, "water_meter", "energy", types.ProtocolMQTT, meta),
		totalFlow:  0,
	}
}

func (wm *WaterMeter) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 瞬时流量与入住率正相关
	instantFlow := engine.Clamp(5+ctx.OccupancyRate*30+engine.GaussNoise(0, 2), 0.5, 50)
	// 累计流量（120s 间隔）
	wm.totalFlow += instantFlow * 2.0 / 60.0 // m³

	return map[string]any{
		"instant_flow": instantFlow,
		"total_flow":   wm.totalFlow,
		"pressure":     engine.Clamp(0.3+engine.GaussNoise(0, 0.02), 0.2, 0.5),
	}
}

func (wm *WaterMeter) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 燃气表 GasMeter ====================

func init() {
	device.RegisterDevice("gas_meter", NewGasMeter)
}

type GasMeter struct {
	device.BaseDevice
	totalFlow float64
}

func NewGasMeter(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &GasMeter{
		BaseDevice: device.NewBaseDevice(id, "gas_meter", "energy", types.ProtocolMQTT, meta),
		totalFlow:  0,
	}
}

func (gm *GasMeter) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 餐厅时段用气量大（11-13点，17-19点）
	hour := ctx.Timestamp.Hour()
	baseFlow := 0.5
	if (hour >= 11 && hour < 13) || (hour >= 17 && hour < 19) {
		baseFlow = 5 + ctx.OccupancyRate*15
	}
	instantFlow := engine.Clamp(baseFlow+engine.GaussNoise(0, 0.5), 0, 25)
	gm.totalFlow += instantFlow * 2.0 / 60.0

	// 偶尔模拟泄漏（极低概率）
	leak := engine.RandBool(0.001)

	return map[string]any{
		"instant_flow": instantFlow,
		"total_flow":   gm.totalFlow,
		"leak_alarm":   leak,
	}
}

func (gm *GasMeter) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if leak, ok := data["leak_alarm"].(bool); ok && leak {
		alarms = append(alarms, types.Alarm{
			DeviceID: gm.ID(), Type: "gas_leak", Level: "critical",
			Message: "燃气泄漏报警",
		})
	}
	return alarms
}

// ==================== 冷热量表 HeatMeter ====================

func init() {
	device.RegisterDevice("heat_meter", NewHeatMeter)
}

type HeatMeter struct {
	device.BaseDevice
	totalEnergy float64
}

func NewHeatMeter(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &HeatMeter{
		BaseDevice: device.NewBaseDevice(id, "heat_meter", "energy", types.ProtocolMQTT, meta),
		totalEnergy: 0,
	}
}

func (hm *HeatMeter) GenerateData(ctx types.ScenarioContext) map[string]any {
	supplyTemp := 45.0 + engine.GaussNoise(0, 1)
	returnTemp := 38.0 + engine.GaussNoise(0, 1)
	flow := engine.Clamp(15+ctx.PowerLoadRate*25+engine.GaussNoise(0, 2), 0, 50)

	// 瞬时冷热量 Q = c × m × ΔT (c=4.186 kJ/(kg·°C), 水密度 1000kg/m³)
	deltaT := supplyTemp - returnTemp
	instantPower := 4.186 * flow * 1000 / 3600 * deltaT // kW
	hm.totalEnergy += instantPower / 60.0                // kWh

	return map[string]any{
		"supply_temp":    engine.Clamp(supplyTemp, 30, 60),
		"return_temp":    engine.Clamp(returnTemp, 20, 50),
		"instant_power":  engine.Clamp(instantPower, 0, 500),
		"total_energy":   hm.totalEnergy,
		"flow":           flow,
	}
}

func (hm *HeatMeter) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 光伏逆变器 PVInverter ====================

func init() {
	device.RegisterDevice("pv_inverter", NewPVInverter)
}

type PVInverter struct {
	device.BaseDevice
	ratedPower  float64
	dailyEnergy float64
	lastDay     int
}

func NewPVInverter(id string, meta map[string]any, cfg map[string]any) device.Device {
	ratedPower := 50.0
	if v, ok := cfg["rated_power"]; ok {
		if f, ok := v.(float64); ok {
			ratedPower = f
		}
	}
	return &PVInverter{
		BaseDevice:  device.NewBaseDevice(id, "pv_inverter", "energy", types.ProtocolMQTT, meta),
		ratedPower:  ratedPower,
		dailyEnergy: 0,
		lastDay:     0,
	}
}

func (pv *PVInverter) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	day := ctx.Timestamp.YearDay()

	// 新的一天重置日发电量
	if day != pv.lastDay {
		pv.dailyEnergy = 0
		pv.lastDay = day
	}

	// 仅白天发电（6:00-18:00）
	outputPower := 0.0
	if hour >= 6 && hour < 18 {
		lux := engine.LuxLevel(hour)
		// 发电功率与光照强度成正比
		outputPower = pv.ratedPower * (lux / 50000.0)
		outputPower = engine.Clamp(outputPower+engine.GaussNoise(0, 0.5), 0, pv.ratedPower)
	}

	pv.dailyEnergy += outputPower * 15.0 / 3600.0 // 15s 间隔

	return map[string]any{
		"output_power":   engine.Clamp(outputPower, 0, pv.ratedPower),
		"daily_energy":   pv.dailyEnergy,
		"total_energy":   pv.dailyEnergy * 30, // 模拟累计
		"dc_voltage":     engine.Clamp(600+outputPower*2+engine.GaussNoise(0, 5), 0, 1000),
		"ac_voltage":     engine.Clamp(220+engine.GaussNoise(0, 1), 210, 230),
		"temperature":    engine.Clamp(35+outputPower/pv.ratedPower*15+engine.GaussNoise(0, 2), 20, 70),
	}
}

func (pv *PVInverter) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if t, ok := data["temperature"].(float64); ok && t > 65 {
		alarms = append(alarms, types.Alarm{
			DeviceID: pv.ID(), Type: "inverter_overheat", Level: "major",
			Message: "逆变器温度过高", Value: t,
		})
	}
	return alarms
}

// ==================== 储能电池 BatteryStorage ====================

func init() {
	device.RegisterDevice("battery_storage", NewBatteryStorage)
}

type BatteryStorage struct {
	device.BaseDevice
	soc    float64 // 0-100
	capacity float64 // kWh
}

func NewBatteryStorage(id string, meta map[string]any, cfg map[string]any) device.Device {
	capacity := 200.0
	if v, ok := cfg["capacity"]; ok {
		if f, ok := v.(float64); ok {
			capacity = f
		}
	}
	return &BatteryStorage{
		BaseDevice: device.NewBaseDevice(id, "battery_storage", "energy", types.ProtocolMQTT, meta),
		soc:        60.0,
		capacity:   capacity,
	}
}

func (bs *BatteryStorage) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	power := 0.0 // 正=充电，负=放电

	// 策略：夜间（0-6）充电，白天高峰（9-18）放电
	if hour >= 0 && hour < 6 {
		// 充电
		power = 30.0
		if bs.soc >= 95 {
			power = 5.0
		}
	} else if hour >= 9 && hour < 18 {
		// 放电
		power = -25.0 * ctx.PowerLoadRate
		if bs.soc <= 15 {
			power = 0 // 低电量保护
		}
	} else {
		power = engine.GaussNoise(0, 2) // 待机微波动
	}

	// 更新 SOC
	deltaSOC := power * 30.0 / 3600.0 / bs.capacity * 100 // 30s 间隔
	bs.soc = engine.Clamp(bs.soc+deltaSOC, 5, 100)

	cellVoltage := engine.Clamp(3.2+bs.soc/100*0.4+engine.GaussNoise(0, 0.02), 2.8, 4.2)
	temp := engine.Clamp(25+math.Abs(power)/bs.capacity*50+engine.GaussNoise(0, 1), 15, 55)
	soh := 98.5 - engine.GaussNoise(0, 0.1) // 缓慢衰减

	return map[string]any{
		"soc":         bs.soc,
		"power":       power,
		"voltage":     cellVoltage * 16, // 16 串
		"temperature": temp,
		"soh":         engine.Clamp(soh, 80, 100),
	}
}

func (bs *BatteryStorage) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if soc, ok := data["soc"].(float64); ok && soc < 10 {
		alarms = append(alarms, types.Alarm{
			DeviceID: bs.ID(), Type: "low_soc", Level: "critical",
			Message: "电池电量极低", Value: soc,
		})
	}
	if t, ok := data["temperature"].(float64); ok && t > 50 {
		alarms = append(alarms, types.Alarm{
			DeviceID: bs.ID(), Type: "battery_overheat", Level: "major",
			Message: "电池温度过高", Value: t,
		})
	}
	return alarms
}

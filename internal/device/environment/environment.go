package environment

import (
	"math"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 温湿度传感器 TempHumiditySensor ====================

func init() {
	device.RegisterDevice("temp_humidity_sensor", NewTempHumiditySensor)
}

type TempHumiditySensor struct {
	device.BaseDevice
	prevTemp float64
	setTemp  float64
}

func NewTempHumiditySensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	setTemp := 24.0
	if v, ok := cfg["set_temp"]; ok {
		if f, ok := v.(float64); ok {
			setTemp = f
		}
	}
	return &TempHumiditySensor{
		BaseDevice: device.NewBaseDevice(id, "temp_humidity_sensor", "environment", types.ProtocolMQTT, meta),
		prevTemp:   24.0,
		setTemp:    setTemp,
	}
}

func (s *TempHumiditySensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 室内温度使用 IndoorTemperature 模型
	temp := engine.IndoorTemperature(ctx.OutdoorTemp, s.setTemp, 0.85)
	s.prevTemp = engine.TemperatureInertia(s.prevTemp, temp, 0.9)

	// 湿度 40-70%RH，与季节相关
	humidity := 55.0
	switch ctx.Season {
	case "summer":
		humidity = 65 + engine.GaussNoise(0, 3)
	case "winter":
		humidity = 45 + engine.GaussNoise(0, 3)
	default:
		humidity = 55 + engine.GaussNoise(0, 3)
	}

	return map[string]any{
		"temperature": engine.Clamp(s.prevTemp, 10, 40),
		"humidity":    engine.Clamp(humidity, 40, 70),
	}
}

func (s *TempHumiditySensor) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if t, ok := data["temperature"].(float64); ok && t > 35 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "high_temperature", Level: "warning",
			Message: "环境温度过高", Value: t,
		})
	}
	if h, ok := data["humidity"].(float64); ok && h > 70 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "high_humidity", Level: "info",
			Message: "环境湿度偏高", Value: h,
		})
	}
	return alarms
}

// ==================== PM2.5传感器 PM25Sensor ====================

func init() {
	device.RegisterDevice("pm25_sensor", NewPM25Sensor)
}

type PM25Sensor struct {
	device.BaseDevice
}

func NewPM25Sensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &PM25Sensor{
		BaseDevice: device.NewBaseDevice(id, "pm25_sensor", "environment", types.ProtocolMQTT, meta),
	}
}

func (s *PM25Sensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	pm25 := engine.PM25Level(ctx.Season, ctx.OutdoorTemp)
	// PM10 通常为 PM2.5 的 1.5 倍
	pm10 := pm25 * 1.5 + engine.GaussNoise(0, 3)

	return map[string]any{
		"pm25": engine.Clamp(pm25, 0, 500),
		"pm10": engine.Clamp(pm10, 0, 600),
	}
}

func (s *PM25Sensor) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if pm25, ok := data["pm25"].(float64); ok && pm25 > 150 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "pm25_exceed", Level: "warning",
			Message: "PM2.5浓度超标", Value: pm25,
		})
	}
	return alarms
}

// ==================== CO2传感器 CO2Sensor ====================

func init() {
	device.RegisterDevice("co2_sensor", NewCO2Sensor)
}

type CO2Sensor struct {
	device.BaseDevice
}

func NewCO2Sensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &CO2Sensor{
		BaseDevice: device.NewBaseDevice(id, "co2_sensor", "environment", types.ProtocolMQTT, meta),
	}
}

func (s *CO2Sensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	co2 := engine.CO2Level(ctx.OccupancyRate)

	return map[string]any{
		"co2": co2,
	}
}

func (s *CO2Sensor) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if co2, ok := data["co2"].(float64); ok && co2 > 1000 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "high_co2", Level: "warning",
			Message: "CO2浓度过高，建议通风", Value: co2,
		})
	}
	return alarms
}

// ==================== 噪声传感器 NoiseSensor ====================

func init() {
	device.RegisterDevice("noise_sensor", NewNoiseSensor)
}

type NoiseSensor struct {
	device.BaseDevice
}

func NewNoiseSensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &NoiseSensor{
		BaseDevice: device.NewBaseDevice(id, "noise_sensor", "environment", types.ProtocolMQTT, meta),
	}
}

func (s *NoiseSensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()
	noise := engine.NoiseLevel(ctx.OccupancyRate, hour)

	return map[string]any{
		"noise": noise,
	}
}

func (s *NoiseSensor) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if n, ok := data["noise"].(float64); ok && n > 70 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "high_noise", Level: "warning",
			Message: "噪声超标", Value: n,
		})
	}
	return alarms
}

// ==================== 气体传感器 GasSensor ====================

func init() {
	device.RegisterDevice("gas_sensor", NewGasSensor)
}

type GasSensor struct {
	device.BaseDevice
}

func NewGasSensor(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &GasSensor{
		BaseDevice: device.NewBaseDevice(id, "gas_sensor", "environment", types.ProtocolMQTT, meta),
	}
}

func (s *GasSensor) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 甲醛 0.02-0.08 mg/m³，新装修或高温时偏高
	formaldehyde := 0.04 + engine.GaussNoise(0, 0.01)
	if ctx.Season == "summer" {
		formaldehyde += 0.02 // 夏季高温挥发多
	}

	// TVOC 0.3-0.6 mg/m³
	tvoc := 0.4 + engine.GaussNoise(0, 0.05)
	if ctx.Season == "summer" {
		tvoc += 0.05
	}

	return map[string]any{
		"formaldehyde": engine.Clamp(formaldehyde, 0.02, 0.08),
		"tvoc":         engine.Clamp(tvoc, 0.3, 0.6),
	}
}

func (s *GasSensor) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if fa, ok := data["formaldehyde"].(float64); ok && fa > 0.08 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "formaldehyde_exceed", Level: "warning",
			Message: "甲醛浓度超标", Value: fa,
		})
	}
	if tv, ok := data["tvoc"].(float64); ok && tv > 0.6 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "tvoc_exceed", Level: "warning",
			Message: "TVOC浓度超标", Value: tv,
		})
	}
	return alarms
}

// ==================== 气象站 WeatherStation ====================

func init() {
	device.RegisterDevice("weather_station", NewWeatherStation)
}

type WeatherStation struct {
	device.BaseDevice
}

func NewWeatherStation(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &WeatherStation{
		BaseDevice: device.NewBaseDevice(id, "weather_station", "environment", types.ProtocolMQTT, meta),
	}
}

func (s *WeatherStation) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	// 室外温度来自场景上下文
	temperature := ctx.OutdoorTemp + engine.GaussNoise(0, 0.5)

	// 湿度 40-80%，夏季偏高
	humidity := 60.0
	switch ctx.Season {
	case "summer":
		humidity = 75 + engine.GaussNoise(0, 5)
	case "winter":
		humidity = 45 + engine.GaussNoise(0, 5)
	default:
		humidity = 60 + engine.GaussNoise(0, 5)
	}

	// 风速 0-15 m/s
	windSpeed := engine.Clamp(engine.GaussNoise(3, 2), 0, 15)

	// 风向 0-360°
	windDirection := float64(engine.RandInt(0, 359))

	// 气压 1000-1020 hPa
	pressure := 1010 + engine.GaussNoise(0, 3)

	// 降雨量 0-5mm，夏季概率较高
	rainfall := 0.0
	if ctx.Season == "summer" && engine.RandBool(0.2) {
		rainfall = engine.RandFloat(0.5, 5)
	} else if engine.RandBool(0.05) {
		rainfall = engine.RandFloat(0.1, 2)
	}

	// 照度
	lux := engine.LuxLevel(hour)

	return map[string]any{
		"temperature":   engine.Clamp(temperature, -20, 50),
		"humidity":      engine.Clamp(humidity, 40, 80),
		"wind_speed":    windSpeed,
		"wind_direction": math.Mod(windDirection, 360),
		"pressure":      engine.Clamp(pressure, 1000, 1020),
		"rainfall":      engine.Clamp(rainfall, 0, 5),
		"lux":           lux,
	}
}

func (s *WeatherStation) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if ws, ok := data["wind_speed"].(float64); ok && ws > 12 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "high_wind", Level: "warning",
			Message: "风速过大", Value: ws,
		})
	}
	if rf, ok := data["rainfall"].(float64); ok && rf > 4 {
		alarms = append(alarms, types.Alarm{
			DeviceID: s.ID(), Type: "heavy_rain", Level: "warning",
			Message: "降雨量过大", Value: rf,
		})
	}
	return alarms
}

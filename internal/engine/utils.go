package engine

import (
	"math"
	"math/rand"
	"time"
)

// ==================== 数学工具 ====================

// GaussNoise 高斯噪声，mean=0
func GaussNoise(mean, stddev float64) float64 {
	return mean + rand.NormFloat64()*stddev
}

// Clamp 限制范围
func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Lerp 线性插值
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// TemperatureInertia 温度惯性模型：新值 = 旧值 + (目标值 - 旧值) * 系数 + 噪声
func TemperatureInertia(current, target, inertia float64) float64 {
	return current + (target-current)*inertia
}

// ==================== 时间规律模型 ====================

// OutdoorTemperature 室外温度（正弦曲线）
// base: 季节基准温度，amplitude: 日振幅
func OutdoorTemperature(hour int, base, amplitude float64) float64 {
	// 最低 5:00，最高 14:00
	phase := float64(hour-14) * math.Pi / 12
	return base + amplitude*math.Cos(phase)
}

// IndoorTemperature 室内温度（受室外和设定温度影响）
func IndoorTemperature(outdoor, setTemp, insulationFactor float64) float64 {
	return setTemp + (outdoor-setTemp)*(1 - insulationFactor)
}

// OccupancyRate 入住率（与工作时间正相关）
func OccupancyRate(hour int, isWorkday bool) float64 {
	if !isWorkday {
		return 0.05 + rand.Float64()*0.1
	}
	switch {
	case hour >= 8 && hour < 9: // 上班高峰
		return 0.3 + rand.Float64()*0.3
	case hour >= 9 && hour < 12: // 上午工作
		return 0.7 + rand.Float64()*0.2
	case hour >= 12 && hour < 14: // 午休
		return 0.3 + rand.Float64()*0.2
	case hour >= 14 && hour < 18: // 下午工作
		return 0.7 + rand.Float64()*0.2
	case hour >= 18 && hour < 20: // 下班
		return 0.3 + rand.Float64()*0.3
	default:
		return 0.05 + rand.Float64()*0.1
	}
}

// PowerLoadRate 用电负载率
func PowerLoadRate(hour int, isWorkday bool) float64 {
	if !isWorkday {
		return 0.1 + rand.Float64()*0.1
	}
	switch {
	case hour >= 8 && hour < 12:
		return 0.6 + rand.Float64()*0.2
	case hour >= 12 && hour < 14:
		return 0.3 + rand.Float64()*0.1
	case hour >= 14 && hour < 18:
		return 0.7 + rand.Float64()*0.2
	case hour >= 18 && hour < 21:
		return 0.5 + rand.Float64()*0.2
	default:
		return 0.1 + rand.Float64()*0.1
	}
}

// LuxLevel 光照强度（lux）
func LuxLevel(hour int) float64 {
	if hour < 6 || hour > 18 {
		return 0
	}
	// 正午 120000 lux，日出日落 0
	dayHours := float64(hour - 6)
	totalDayHours := 12.0
	sunAngle := dayHours / totalDayHours * math.Pi
	lux := 120000 * math.Sin(sunAngle)
	if lux < 0 {
		lux = 0
	}
	return lux
}

// CO2Level CO2 浓度（ppm），与入住率正相关
func CO2Level(occupancyRate float64) float64 {
	// 室外 400ppm，室内随入住率升高
	base := 400.0
	increase := occupancyRate * 800 // 满员时约 1200ppm
	return base + increase + GaussNoise(0, 20)
}

// PM25Level PM2.5 浓度（μg/m³），与季节和天气相关
func PM25Level(season string, outdoorTemp float64) float64 {
	base := 30.0
	switch season {
	case "winter":
		base = 80 // 冬季逆温，PM2.5 高
	case "summer":
		base = 20 // 夏季扩散好
	}
	// 低温高湿时 PM2.5 易升高
	if outdoorTemp < 10 {
		base *= 1.5
	}
	return base + GaussNoise(0, 10)
}

// NoiseLevel 噪声水平（dB）
func NoiseLevel(occupancyRate float64, hour int) float64 {
	// 背景噪声 35dB
	base := 35.0
	// 加上人员活动噪声
	activity := occupancyRate * 30
	// 夜间施工噪声
	if hour >= 22 || hour < 6 {
		activity *= 0.2
	}
	return base + activity + GaussNoise(0, 3)
}

// ==================== 物理约束 ====================

// ThreePhaseVoltage 三相电压，返回三相电压值（带不平衡度）
func ThreePhaseVoltage(ratedV float64) (float64, float64, float64) {
	unbalance := 0.02 // 2% 不平衡度
	va := ratedV + GaussNoise(0, ratedV*unbalance)
	vb := ratedV + GaussNoise(0, ratedV*unbalance)
	vc := ratedV + GaussNoise(0, ratedV*unbalance)
	return va, vb, vc
}

// ThreePhaseCurrent 三相电流（考虑负载率和不平衡）
func ThreePhaseCurrent(ratedI float64, loadRate float64) (float64, float64, float64) {
	baseI := ratedI * loadRate
	ia := baseI + GaussNoise(0, baseI*0.05)
	ib := baseI + GaussNoise(0, baseI*0.05)
	ic := baseI + GaussNoise(0, baseI*0.05)
	return ia, ib, ic
}

// PowerFactor 功率因数（与负载率相关）
func PowerFactor(loadRate float64) float64 {
	// 轻载时功率因数低，满载时高
	pf := 0.75 + loadRate*0.2 + GaussNoise(0, 0.02)
	return Clamp(pf, 0.5, 0.99)
}

// BatterySOC 电池充电 SOC 模拟（简单库仑计数）
// current: 电流（+充电，-放电），capacity: 额定容量（Ah），dt: 时间间隔（小时）
func BatterySOC(soc, current, capacity, dt float64) float64 {
	// ΔSOC = I × dt / capacity × 100
	delta := current * dt / capacity * 100
	newSOC := soc + delta
	return Clamp(newSOC, 0, 100)
}

// ==================== 随机工具 ====================

// RandInt 随机整数 [min, max]
func RandInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

// RandFloat 随机浮点数 [min, max)
func RandFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// RandBool 随机布尔值，p 为 true 的概率
func RandBool(p float64) bool {
	return rand.Float64() < p
}

// RandChoice 从切片中随机选一个
func RandChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}

// RandString 随机字符串（用于 ID 生成）
func RandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// ==================== 时间工具 ====================

// Season 获取季节（简化：3-5 春，6-8 夏，9-11 秋，12-2 冬）
func Season(t time.Time) string {
	m := t.Month()
	switch {
	case m >= 3 && m <= 5:
		return "spring"
	case m >= 6 && m <= 8:
		return "summer"
	case m >= 9 && m <= 11:
		return "autumn"
	default:
		return "winter"
	}
}

// IsWorkday 判断是否工作日（简化：周一到周五）
func IsWorkday(t time.Time) bool {
	d := t.Weekday()
	return d >= time.Monday && d <= time.Friday
}

// SeasonalBaseTemp 季节基准温度（摄氏度）
func SeasonalBaseTemp(season string) float64 {
	switch season {
	case "spring":
		return 20
	case "summer":
		return 28
	case "autumn":
		return 18
	case "winter":
		return 8
	default:
		return 20
	}
}

// WeekdayFactor 星期因子（影响设备行为）
func WeekdayFactor(t time.Time) float64 {
	if !IsWorkday(t) {
		return 0.2 // 周末降低为 20%
	}
	return 1.0
}

// ==================== 场景覆盖工具 ====================

// OverrideValue 根据场景覆写值
func OverrideValue(base float64, overrides map[string]any, key string) float64 {
	if v, ok := overrides[key].(float64); ok {
		return v
	}
	return base
}

// TriggerScenario 触发瞬时场景（修改设备状态）
// 返回需要修改的设备状态 map
func TriggerScenario(scenario string, devices []string) map[string]map[string]any {
	result := make(map[string]map[string]any)
	switch scenario {
	case "fire_emergency":
		for _, id := range devices {
			result[id] = map[string]any{
				"alarm": true,
			}
		}
	case "power_outage":
		for _, id := range devices {
			result[id] = map[string]any{
				"power": 0,
			}
		}
	}
	return result
}

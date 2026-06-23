package broadcast

import (
	"time"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 广播终端 BroadcastTerminal ====================

func init() {
	device.RegisterDevice("broadcast_terminal", NewBroadcastTerminal)
}

type BroadcastTerminal struct {
	device.BaseDevice
	volume       int    // 30-60 dB
	playing      bool
	currentProgram string // 背景音乐 / 新闻 / 通知 / 无
}

func NewBroadcastTerminal(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &BroadcastTerminal{
		BaseDevice:    device.NewBaseDevice(id, "broadcast_terminal", "broadcast", types.ProtocolMQTT, meta),
		volume:        45,
		playing:       false,
		currentProgram: "无",
	}
}

func (b *BroadcastTerminal) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	// 在线状态：工作时间在线，非工作时间离线概率高
	online := true
	if hour < 7 || hour >= 22 {
		online = engine.RandBool(0.3)
	}

	if !online {
		b.playing = false
		b.currentProgram = "无"
		return map[string]any{
			"online":          false,
			"volume":          0,
			"playing":         false,
			"current_program": "无",
		}
	}

	// 播放内容根据时间段决定
	if !b.playing || engine.RandBool(0.2) {
		switch {
		case hour >= 8 && hour < 9:
			// 早间新闻
			b.currentProgram = "新闻"
			b.playing = engine.RandBool(0.8)
		case hour >= 12 && hour < 13:
			// 午间新闻
			b.currentProgram = "新闻"
			b.playing = engine.RandBool(0.7)
		case hour >= 18 && hour < 19:
			// 晚间新闻
			b.currentProgram = "新闻"
			b.playing = engine.RandBool(0.7)
		case (hour >= 9 && hour < 12) || (hour >= 14 && hour < 18):
			// 工作时间背景音乐
			b.currentProgram = "背景音乐"
			b.playing = engine.RandBool(0.6)
		case hour >= 7 && hour < 8:
			// 早间通知
			b.currentProgram = "通知"
			b.playing = engine.RandBool(0.3)
		default:
			b.currentProgram = "无"
			b.playing = false
		}
	}

	// 音量 30-60 dB
	if b.playing {
		b.volume = engine.RandInt(35, 55)
	} else {
		b.volume = 0
	}

	return map[string]any{
		"online":          online,
		"volume":          b.volume,
		"playing":         b.playing,
		"current_program": b.currentProgram,
	}
}

func (b *BroadcastTerminal) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 紧急广播终端 EmergencyBroadcast ====================

func init() {
	device.RegisterDevice("emergency_broadcast", NewEmergencyBroadcast)
}

type EmergencyBroadcast struct {
	device.BaseDevice
	lastTestWeek int // 上次测试的周数
}

func NewEmergencyBroadcast(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &EmergencyBroadcast{
		BaseDevice:   device.NewBaseDevice(id, "emergency_broadcast", "broadcast", types.ProtocolHTTP, meta),
		lastTestWeek: -1,
	}
}

// isoWeek 获取 ISO 周数
func isoWeek(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func (e *EmergencyBroadcast) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 在线状态：基本保持在线
	online := engine.RandBool(0.99)

	// 每周一次测试（周三 10:00 左右）
	testStatus := false
	currentWeek := isoWeek(ctx.Timestamp)
	hour := ctx.Timestamp.Hour()
	weekday := ctx.Timestamp.Weekday()

	if weekday == time.Wednesday && hour >= 10 && hour < 11 && currentWeek != e.lastTestWeek {
		if engine.RandBool(0.3) { // 在该小时窗口内约 30% 概率触发
			testStatus = true
			e.lastTestWeek = currentWeek
		}
	}

	// 极低概率触发紧急广播
	triggered := engine.RandBool(0.0001)

	return map[string]any{
		"online":      online,
		"test_status": testStatus,
		"triggered":   triggered,
	}
}

func (e *EmergencyBroadcast) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if triggered, ok := data["triggered"].(bool); ok && triggered {
		alarms = append(alarms, types.Alarm{
			DeviceID: e.ID(), Type: "emergency_triggered", Level: "critical",
			Message: "紧急广播已触发",
		})
	}
	if online, ok := data["online"].(bool); ok && !online {
		alarms = append(alarms, types.Alarm{
			DeviceID: e.ID(), Type: "device_offline", Level: "major",
			Message: "紧急广播终端离线",
		})
	}
	return alarms
}

package access

import (
	"fmt"
	"time"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 门禁控制器 AccessController ====================

func init() {
	device.RegisterDevice("access_controller", NewAccessController)
}

type AccessController struct {
	device.BaseDevice
	passCount int
}

func NewAccessController(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &AccessController{
		BaseDevice: device.NewBaseDevice(id, "access_controller", "access", types.ProtocolHTTP, meta),
		passCount:  0,
	}
}

func (ac *AccessController) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	// 工作时间高频刷卡
	cardEvent := map[string]any{}
	if ctx.IsWorkday && hour >= 8 && hour < 18 && engine.RandBool(0.7) {
		cardNo := fmt.Sprintf("CARD%06d", engine.RandInt(1, 9999))
		personName := engine.RandChoice([]string{"张三", "李四", "王五", "赵六", "陈七", "周八"})
		result := "allow"
		if engine.RandBool(0.03) {
			result = "deny"
		}
		cardEvent = map[string]any{
			"card_no":     cardNo,
			"person_name": personName,
			"result":      result,
		}
		if result == "allow" {
			ac.passCount++
		}
	}

	doorStatus := "closed"
	openDuration := 0.0
	if len(cardEvent) > 0 {
		if r, ok := cardEvent["result"].(string); ok && r == "allow" {
			doorStatus = "open"
			openDuration = engine.Clamp(3+engine.GaussNoise(0, 1), 1, 10)
		}
	}

	illegalEntry := engine.RandBool(0.005)

	return map[string]any{
		"door_status":   doorStatus,
		"card_event":    cardEvent,
		"open_duration": openDuration,
		"illegal_entry": illegalEntry,
		"pass_count":    ac.passCount,
	}
}

func (ac *AccessController) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if ie, ok := data["illegal_entry"].(bool); ok && ie {
		alarms = append(alarms, types.Alarm{
			DeviceID:  ac.ID(),
			Type:      "illegal_entry",
			Level:     "critical",
			Message:   "门禁检测到非法闯入",
			Timestamp: time.Now(),
			Value:     ie,
		})
	}
	if event, ok := data["card_event"].(map[string]any); ok {
		if r, ok := event["result"].(string); ok && r == "deny" {
			alarms = append(alarms, types.Alarm{
				DeviceID:  ac.ID(),
				Type:      "card_denied",
				Level:     "info",
				Message:   "门禁刷卡拒绝",
				Timestamp: time.Now(),
				Value:     event,
			})
		}
	}
	return alarms
}

// ==================== 人脸识别终端 FaceTerminal ====================

func init() {
	device.RegisterDevice("face_terminal", NewFaceTerminal)
}

type FaceTerminal struct {
	device.BaseDevice
}

func NewFaceTerminal(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &FaceTerminal{
		BaseDevice: device.NewBaseDevice(id, "face_terminal", "access", types.ProtocolHTTP, meta),
	}
}

func (ft *FaceTerminal) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	recognizeResult := ""
	bodyTemp := 0.0
	recordID := ""

	// 工作时间高频识别
	if ctx.IsWorkday && hour >= 8 && hour < 18 && engine.RandBool(0.6) {
		if engine.RandBool(0.95) {
			recognizeResult = "allow"
		} else {
			recognizeResult = "deny"
		}
		bodyTemp = engine.Clamp(36.5+engine.GaussNoise(0, 0.2), 36.0, 37.5)
		recordID = fmt.Sprintf("FR%d%04d", ctx.Timestamp.Unix(), engine.RandInt(0, 9999))
	} else if !ctx.IsWorkday && hour >= 9 && hour < 17 && engine.RandBool(0.2) {
		recognizeResult = "allow"
		bodyTemp = engine.Clamp(36.5+engine.GaussNoise(0, 0.2), 36.0, 37.5)
		recordID = fmt.Sprintf("FR%d%04d", ctx.Timestamp.Unix(), engine.RandInt(0, 9999))
	}

	// 体温异常偶尔出现
	if bodyTemp > 0 && engine.RandBool(0.005) {
		bodyTemp = engine.Clamp(37.8+engine.GaussNoise(0, 0.2), 37.3, 39.0)
		recognizeResult = "deny"
	}

	return map[string]any{
		"recognize_result": recognizeResult,
		"body_temp":        bodyTemp,
		"record_id":        recordID,
		"online":           true,
	}
}

func (ft *FaceTerminal) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if temp, ok := data["body_temp"].(float64); ok && temp >= 37.3 {
		alarms = append(alarms, types.Alarm{
			DeviceID:  ft.ID(),
			Type:      "high_body_temp",
			Level:     "warning",
			Message:   "人脸识别终端检测到体温异常",
			Timestamp: time.Now(),
			Value:     temp,
		})
	}
	return alarms
}

// ==================== 访客机 VisitorKiosk ====================

func init() {
	device.RegisterDevice("visitor_kiosk", NewVisitorKiosk)
}

type VisitorKiosk struct {
	device.BaseDevice
}

func NewVisitorKiosk(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &VisitorKiosk{
		BaseDevice: device.NewBaseDevice(id, "visitor_kiosk", "access", types.ProtocolHTTP, meta),
	}
}

func (vk *VisitorKiosk) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	registerEvent := false
	visitorType := ""
	visitee := ""
	authPeriod := ""

	// 工作日白天有访客登记
	if ctx.IsWorkday && hour >= 9 && hour < 17 && engine.RandBool(0.3) {
		registerEvent = true
		visitorType = engine.RandChoice([]string{"临时", "预约"})
		visitee = engine.RandChoice([]string{"张三", "李四", "王五", "赵六", "陈七"})
		authPeriod = engine.RandChoice([]string{"2小时", "4小时", "半天", "全天"})
	}

	return map[string]any{
		"register_event": registerEvent,
		"visitor_type":   visitorType,
		"visitee":        visitee,
		"auth_period":    authPeriod,
	}
}

func (vk *VisitorKiosk) CheckAlarms(data map[string]any) []types.Alarm { return nil }

// ==================== 通道闸机 Turnstile ====================

func init() {
	device.RegisterDevice("turnstile", NewTurnstile)
}

type Turnstile struct {
	device.BaseDevice
	passCount int
}

func NewTurnstile(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &Turnstile{
		BaseDevice: device.NewBaseDevice(id, "turnstile", "access", types.ProtocolHTTP, meta),
		passCount:  0,
	}
}

func (ts *Turnstile) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	direction := ""
	alarm := false

	// 工作时间高频通行
	if ctx.IsWorkday && hour >= 8 && hour < 18 && engine.RandBool(0.5) {
		direction = engine.RandChoice([]string{"in", "out"})
		ts.passCount++
	} else if !ctx.IsWorkday && hour >= 10 && hour < 20 && engine.RandBool(0.15) {
		direction = engine.RandChoice([]string{"in", "out"})
		ts.passCount++
	}

	if direction == "" {
		direction = "idle"
	}

	alarm = engine.RandBool(0.01) // 低概率告警

	return map[string]any{
		"direction":  direction,
		"pass_count": ts.passCount,
		"alarm":      alarm,
		"running":    true,
	}
}

func (ts *Turnstile) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if alarm, ok := data["alarm"].(bool); ok && alarm {
		alarms = append(alarms, types.Alarm{
			DeviceID:  ts.ID(),
			Type:      "turnstile_alarm",
			Level:     "warning",
			Message:   "闸机异常报警",
			Timestamp: time.Now(),
			Value:     alarm,
		})
	}
	return alarms
}

package elevator

import (
	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
)

// ==================== 电梯控制器 ElevatorController ====================

func init() {
	device.RegisterDevice("elevator_controller", NewElevatorController)
}

type ElevatorController struct {
	device.BaseDevice
	currentFloor int
	direction    string // up / down / idle
	status       string // running / idle / fault
	doorStatus   string // open / closed / opening / closing
	targetFloor  int
	totalFloors  int
	doorTimer    int // 门状态计时器（模拟开关门过程）
}

func NewElevatorController(id string, meta map[string]any, cfg map[string]any) device.Device {
	totalFloors := 20
	if v, ok := cfg["total_floors"]; ok {
		if f, ok := v.(float64); ok {
			totalFloors = int(f)
		}
	}
	return &ElevatorController{
		BaseDevice:   device.NewBaseDevice(id, "elevator_controller", "elevator", types.ProtocolMQTT, meta),
		currentFloor: 1,
		direction:    "idle",
		status:       "idle",
		doorStatus:   "closed",
		targetFloor:  1,
		totalFloors:  totalFloors,
		doorTimer:    0,
	}
}

func (e *ElevatorController) GenerateData(ctx types.ScenarioContext) map[string]any {
	// 极低概率故障
	if e.status != "fault" && engine.RandBool(0.001) {
		e.status = "fault"
		e.direction = "idle"
		e.doorStatus = "closed"
	}

	if e.status == "fault" {
		// 故障后有概率恢复
		if engine.RandBool(0.05) {
			e.status = "idle"
			e.doorStatus = "closed"
		} else {
			return map[string]any{
				"current_floor": e.currentFloor,
				"direction":     "idle",
				"status":        "fault",
				"door_status":   "closed",
				"load":          0,
				"fault_code":    engine.RandChoice([]int{101, 102, 103, 201, 202}),
			}
		}
	}

	// 电梯运行逻辑
	switch e.status {
	case "idle":
		// 门关闭状态下，根据入住率决定是否接到新任务
		if e.doorStatus == "closed" {
			if engine.RandBool(ctx.OccupancyRate * 0.4) {
				// 生成新目标楼层
				e.targetFloor = engine.RandInt(1, e.totalFloors)
				if e.targetFloor != e.currentFloor {
					e.status = "running"
					if e.targetFloor > e.currentFloor {
						e.direction = "up"
					} else {
						e.direction = "down"
					}
				}
			}
		}

	case "running":
		// 每次上报（5s）移动一层
		if e.direction == "up" {
			e.currentFloor++
		} else if e.direction == "down" {
			e.currentFloor--
		}

		// 到达目标楼层
		if e.currentFloor == e.targetFloor {
			e.status = "idle"
			e.direction = "idle"
			e.doorStatus = "opening"
			e.doorTimer = 1 // 开始开门
		}
	}

	// 门状态机
	switch e.doorStatus {
	case "opening":
		e.doorTimer++
		if e.doorTimer >= 2 { // 开门过程约 2 个周期（10s）
			e.doorStatus = "open"
			e.doorTimer = 0
		}
	case "open":
		e.doorTimer++
		if e.doorTimer >= 3 { // 开门停留约 3 个周期（15s）
			e.doorStatus = "closing"
			e.doorTimer = 0
		}
	case "closing":
		e.doorTimer++
		if e.doorTimer >= 2 { // 关门过程约 2 个周期（10s）
			e.doorStatus = "closed"
			e.doorTimer = 0
		}
	}

	// 负载 0-1000kg，与入住率和运行状态相关
	load := 0.0
	if e.status == "running" || e.doorStatus == "open" {
		load = ctx.OccupancyRate * 600 + engine.GaussNoise(0, 50)
	}
	load = engine.Clamp(load, 0, 1000)

	faultCode := 0
	if e.status == "fault" {
		faultCode = engine.RandChoice([]int{101, 102, 103, 201, 202})
	}

	return map[string]any{
		"current_floor": e.currentFloor,
		"direction":     e.direction,
		"status":        e.status,
		"door_status":   e.doorStatus,
		"load":          load,
		"fault_code":    faultCode,
	}
}

func (e *ElevatorController) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if status, ok := data["status"].(string); ok && status == "fault" {
		alarms = append(alarms, types.Alarm{
			DeviceID: e.ID(), Type: "elevator_fault", Level: "critical",
			Message: "电梯故障",
		})
	}
	if load, ok := data["load"].(float64); ok && load > 900 {
		alarms = append(alarms, types.Alarm{
			DeviceID: e.ID(), Type: "overload", Level: "major",
			Message: "电梯超载报警", Value: load,
		})
	}
	return alarms
}

// ==================== 扶梯控制器 EscalatorController ====================

func init() {
	device.RegisterDevice("escalator_controller", NewEscalatorController)
}

type EscalatorController struct {
	device.BaseDevice
	running   bool
	direction string // up / down
}

func NewEscalatorController(id string, meta map[string]any, cfg map[string]any) device.Device {
	return &EscalatorController{
		BaseDevice: device.NewBaseDevice(id, "escalator_controller", "elevator", types.ProtocolMQTT, meta),
		running:    true,
		direction:  "up",
	}
}

func (e *EscalatorController) GenerateData(ctx types.ScenarioContext) map[string]any {
	hour := ctx.Timestamp.Hour()

	// 运行时间 6:00-23:00，非工作时间停机
	if hour >= 6 && hour < 23 {
		// 工作时间运行，偶尔停机维护
		e.running = engine.RandBool(0.98)
	} else {
		e.running = false
	}

	// 方向：工作日早晚高峰切换
	if e.running {
		if ctx.IsWorkday {
			if hour >= 7 && hour < 10 {
				e.direction = "up" // 早高峰上行
			} else if hour >= 17 && hour < 20 {
				e.direction = "down" // 晚高峰下行
			} else {
				// 随机方向
				if engine.RandBool(0.5) {
					e.direction = "up"
				} else {
					e.direction = "down"
				}
			}
		} else {
			// 周末随机
			e.direction = engine.RandChoice([]string{"up", "down"})
		}
	}

	// 极低概率故障
	faultCode := 0
	if e.running && engine.RandBool(0.001) {
		e.running = false
		faultCode = engine.RandChoice([]int{301, 302, 303})
	}

	speed := 0.0
	if e.running {
		speed = 0.5 + engine.GaussNoise(0, 0.02)
	}

	return map[string]any{
		"running":    e.running,
		"direction":  e.direction,
		"speed":      engine.Clamp(speed, 0, 1),
		"fault_code": faultCode,
	}
}

func (e *EscalatorController) CheckAlarms(data map[string]any) []types.Alarm {
	var alarms []types.Alarm
	if fc, ok := data["fault_code"].(int); ok && fc != 0 {
		alarms = append(alarms, types.Alarm{
			DeviceID: e.ID(), Type: "escalator_fault", Level: "major",
			Message: "扶梯故障", Value: fc,
		})
	}
	return alarms
}

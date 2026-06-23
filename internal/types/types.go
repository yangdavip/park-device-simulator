package types

import "time"

// ProtocolType 协议类型
type ProtocolType string

const (
	ProtocolMQTT   ProtocolType = "mqtt"
	ProtocolHTTP   ProtocolType = "http"
	ProtocolModbus ProtocolType = "modbus"
	ProtocolOPCUA  ProtocolType = "opcua"
)

// DeviceStatus 设备状态
type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "online"
	StatusOffline DeviceStatus = "offline"
	StatusError   DeviceStatus = "error"
)

// SystemType 系统类型
type SystemType string

const (
	SysBAS         SystemType = "bas"
	SysLighting    SystemType = "lighting"
	SysSecurity    SystemType = "security"
	SysAccess      SystemType = "access"
	SysFire        SystemType = "fire"
	SysParking     SystemType = "parking"
	SysEnergy      SystemType = "energy"
	SysEnvironment SystemType = "environment"
	SysElevator    SystemType = "elevator"
	SysBroadcast   SystemType = "broadcast"
)

// ScenarioContext 场景上下文
type ScenarioContext struct {
	Timestamp     time.Time
	Scenario      string
	OutdoorTemp   float64
	OccupancyRate float64
	PowerLoadRate float64
	IsWorkday     bool
	Season        string
	ExtraParams   map[string]any
}

// Alarm 告警
type Alarm struct {
	DeviceID  string    `json:"device_id"`
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"ts"`
	Value     any       `json:"value,omitempty"`
}

// ReportData 上报数据
type ReportData struct {
	DeviceID   string         `json:"device_id"`
	DeviceType string         `json:"device_type"`
	System     string         `json:"system"`
	Protocol   string         `json:"protocol"`
	Timestamp  time.Time      `json:"ts"`
	Data       map[string]any `json:"data"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

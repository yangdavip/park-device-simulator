package config

import "time"

// ==================== 园区配置 ====================

type ParkConfig struct {
	Park     ParkInfo         `yaml:"park"`
	Buildings []BuildingConfig `yaml:"buildings"`
}

type ParkInfo struct {
	ID       string     `yaml:"id"`
	Name     string     `yaml:"name"`
	Location Location   `yaml:"location"`
	Timezone string     `yaml:"timezone"`
}

type Location struct {
	City      string  `yaml:"city"`
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
}

type BuildingConfig struct {
	ID     string         `yaml:"id"`
	Name   string         `yaml:"name"`
	Floors int            `yaml:"floors"`
	Systems []SystemConfig `yaml:"systems"`
}

type SystemConfig struct {
	Type    string         `yaml:"type"`
	Enabled bool           `yaml:"enabled"`
	Devices []DeviceDef    `yaml:"devices"`
}

type DeviceDef struct {
	Type       string         `yaml:"type"`
	Count      int            `yaml:"count"`
	FloorRange []int          `yaml:"floor_range"`
	Location   string         `yaml:"location"`
	Naming     string         `yaml:"naming"`
	Protocol   string         `yaml:"protocol"`
	Config     map[string]any  `yaml:"config"`
}

// ==================== 协议配置 ====================

type ProtocolConfig struct {
	MQTT   MQTTConfig   `yaml:"mqtt"`
	HTTP    HTTPConfig    `yaml:"http"`
	Modbus  ModbusConfig  `yaml:"modbus"`
	Opcua   OpcuaConfig   `yaml:"opcua"`
}

type MQTTConfig struct {
	Broker  MQTTBroker  `yaml:"broker"`
	Client  MQTTClient  `yaml:"client"`
}

type MQTTBroker struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Embedded bool   `yaml:"embedded"`
}

type MQTTClient struct {
	Keepalive int    `yaml:"keepalive"`
	QOS       int    `yaml:"qos"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type HTTPConfig struct {
	Server      HTTPServer  `yaml:"server"`
	CallbackURL string      `yaml:"callback_url"`
	Timeout     int         `yaml:"timeout"`
}

type HTTPServer struct {
	Port int `yaml:"port"`
}

type ModbusConfig struct {
	Servers []ModbusServer `yaml:"servers"`
}

type ModbusServer struct {
	Name      string  `yaml:"name"`
	Port      int     `yaml:"port"`
	SlaveIDs  []byte  `yaml:"slave_ids"`
}

type OpcuaConfig struct {
	Server OpcuaServer `yaml:"server"`
}

type OpcuaServer struct {
	Enable    bool   `yaml:"enable"`
	Port      int    `yaml:"port"`
	Endpoint  string `yaml:"endpoint"`
	Security  string `yaml:"security"`
}

// ==================== 场景配置 ====================

type ScenarioConfig struct {
	Scenarios []ScenarioDef `yaml:"scenarios"`
}

type ScenarioDef struct {
	Name        string          `yaml:"name"`
	Description string         `yaml:"description"`
	Type        string         `yaml:"type"`        // "schedule" or "instant"
	Schedule    *ScheduleDef   `yaml:"schedule"`    // 仅 schedule 类型
	Overrides   map[string]any `yaml:"overrides"`   // 仅 schedule 类型
	Trigger     *TriggerDef    `yaml:"trigger"`      // 仅 instant 类型
	Sequence    []SequenceStep `yaml:"sequence"`     // 仅 instant 类型
}

type ScheduleDef struct {
	Type      string   `yaml:"type"`      // "weekday" | "date_range" | "always"
	Start     string   `yaml:"start"`      // 当 type=date_range 时
	End       string   `yaml:"end"`        // 当 type=date_range 时
	Weekdays  []string `yaml:"weekdays"`   // 当 type=weekday 时，如 ["Mon","Tue"]
	StartTime string   `yaml:"start_time"`  // "HH:MM"
	EndTime   string   `yaml:"end_time"`   // "HH:MM"
}

type TriggerDef struct {
	Manual    bool     `yaml:"manual"`
	CronExpr  string   `yaml:"cron_expr"`
	OnAlarm   string   `yaml:"on_alarm"`
}

type SequenceStep struct {
	Time     string          `yaml:"time"`     // 相对于触发时间的偏移，如 "5s" "1m"
	Actions  []ActionDef     `yaml:"actions"`
}

type ActionDef struct {
	DeviceType string         `yaml:"device_type"`
	Target     string         `yaml:"target"`     // 设备ID或通配符
	State      map[string]any  `yaml:"state"`
}

// ==================== 时间因子配置 ====================

// TimeFactorConfig 时间段 → 因子值映射
type TimeFactorConfig struct {
	Occupancy  map[string]float64 `yaml:"occupancy"`
	PowerLoad  map[string]float64 `yaml:"power_load"`
}

// ParseTimeRange 解析 "8-12" 格式的时间范围
func ParseTimeRange(s string) (int, int) {
	// 简单实现，支持 "8-12" "0-6" 格式
	// 实际解析在 scenario.go 中处理
	return 0, 0
}

// ==================== 上报间隔解析 ====================

// ParseInterval 解析 "30s" "1m" "5m" 格式
func ParseInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

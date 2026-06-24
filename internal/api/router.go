package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
	mqttAdapter "park-device-simulator/internal/protocol/mqtt"
	httpAdapter "park-device-simulator/internal/protocol/http"
	modbusAdapter "park-device-simulator/internal/protocol/modbus"
	opcuaAdapter "park-device-simulator/internal/protocol/opcua"
)

// ==================== API Server ====================

type Server struct {
	mu         sync.RWMutex
	scheduler  *engine.Scheduler
	scenario   *engine.ScenarioEngine
	alarm      *engine.AlarmEngine
	mqtt       *mqttAdapter.Adapter
	http       *httpAdapter.Adapter
	modbus     *modbusAdapter.Adapter
	opcua      *opcuaAdapter.Adapter
	scenarios  []string
}

func NewServer(
	scheduler *engine.Scheduler,
	scenario *engine.ScenarioEngine,
	alarm *engine.AlarmEngine,
	mqtt *mqttAdapter.Adapter,
	http *httpAdapter.Adapter,
	modbus *modbusAdapter.Adapter,
	opcua *opcuaAdapter.Adapter,
	scenarios []string,
) *Server {
	return &Server{
		scheduler: scheduler,
		scenario:  scenario,
		alarm:     alarm,
		mqtt:      mqtt,
		http:      http,
		modbus:    modbus,
		opcua:     opcua,
		scenarios: scenarios,
	}
}

func (s *Server) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.GET("/stats", s.getStats)
		api.GET("/devices", s.getDevices)
		api.GET("/devices/:id", s.getDevice)
		api.GET("/devices/:id/data", s.getDeviceData)
		api.GET("/scenarios", s.getScenarios)
		api.POST("/scenarios/:name/activate", s.activateScenario)
		api.GET("/alarms", s.getAlarms)
		api.PUT("/alarms/:id/ack", s.ackAlarm)
		api.GET("/protocols/status", s.getProtocolStatus)
		api.GET("/protocols/info", s.getProtocolInfo)
	}
}

// ==================== Handlers ====================

func (s *Server) getStats(c *gin.Context) {
	devices := s.scheduler.Devices()
	total := len(devices)
	online := 0
	systemStats := make(map[string]int)

	for _, d := range devices {
		if d.Status() == types.StatusOnline {
			online++
		}
		systemStats[d.System()]++
	}

	c.JSON(http.StatusOK, gin.H{
		"total_devices":   total,
		"online_devices":  online,
		"offline_devices": total - online,
		"online_rate":     float64(online) / float64(max(total, 1)) * 100,
		"system_stats":    systemStats,
		"current_scenario": s.scenario.CurrentScenario(),
		"registered_types": device.RegisteredTypes(),
	})
}

func (s *Server) getDevices(c *gin.Context) {
	system := c.Query("system")
	status := c.Query("status")
	limit := 100
	if l := c.Query("limit"); l != "" {
		// 简单解析
	}

	devices := s.scheduler.Devices()
	result := make([]map[string]any, 0, len(devices))

	for _, d := range devices {
		if system != "" && d.System() != system {
			continue
		}
		if status != "" && string(d.Status()) != status {
			continue
		}

		item := map[string]any{
			"id":       d.ID(),
			"type":     d.Type(),
			"system":   d.System(),
			"protocol": string(d.Protocol()),
			"status":   string(d.Status()),
			"last_data": d.LastData(),
		}
		if meta := d.Meta(); meta != nil {
			item["metadata"] = meta
		}
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(result),
		"devices": result,
	})
}

func (s *Server) getDevice(c *gin.Context) {
	id := c.Param("id")
	for _, d := range s.scheduler.Devices() {
		if d.ID() == id {
			c.JSON(http.StatusOK, gin.H{
				"id":       d.ID(),
				"type":     d.Type(),
				"system":   d.System(),
				"protocol": string(d.Protocol()),
				"status":   string(d.Status()),
				"metadata": d.Meta(),
			})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
}

func (s *Server) getDeviceData(c *gin.Context) {
	id := c.Param("id")
	for _, d := range s.scheduler.Devices() {
		if d.ID() == id {
			c.JSON(http.StatusOK, gin.H{
				"device_id":   d.ID(),
				"device_type": d.Type(),
				"status":      string(d.Status()),
				"last_data":   d.LastData(),
			})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
}

func (s *Server) getScenarios(c *gin.Context) {
	scenarioMeta := map[string]map[string]string{
		"normal_workday": {"description": "正常工作日", "type": "schedule"},
		"weekend":       {"description": "周末", "type": "schedule"},
		"holiday":       {"description": "节假日", "type": "schedule"},
		"summer_peak":   {"description": "夏季高温", "type": "schedule"},
		"winter_peak":   {"description": "冬季严寒", "type": "schedule"},
		"fire_emergency": {"description": "消防突发", "type": "instant"},
		"power_outage":  {"description": "停电事件", "type": "instant"},
		"intrusion":     {"description": "安防入侵", "type": "instant"},
	}
	scenarios := make([]map[string]any, 0, len(s.scenarios))
	current := s.scenario.CurrentScenario()
	for _, name := range s.scenarios {
		meta := scenarioMeta[name]
		entry := map[string]any{
			"name":   name,
			"active": name == current,
		}
		if meta != nil {
			entry["description"] = meta["description"]
			entry["type"] = meta["type"]
		}
		scenarios = append(scenarios, entry)
	}
	c.JSON(http.StatusOK, gin.H{
		"current":   current,
		"scenarios": scenarios,
	})
}

func (s *Server) activateScenario(c *gin.Context) {
	name := c.Param("name")
	s.scenario.SetScenario(name)
	c.JSON(http.StatusOK, gin.H{
		"message":     "scenario activated",
		"scenario":    name,
	})
}

func (s *Server) getAlarms(c *gin.Context) {
	alarms := s.alarm.Alarms(100)
	c.JSON(http.StatusOK, gin.H{
		"total":  len(alarms),
		"alarms": alarms,
	})
}

func (s *Server) ackAlarm(c *gin.Context) {
	s.alarm.ClearAlarms()
	c.JSON(http.StatusOK, gin.H{"message": "alarms cleared"})
}

func (s *Server) getProtocolInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"mqtt": gin.H{
			"broker_host":    s.mqtt.BrokerHost(),
			"broker_port":    s.mqtt.BrokerPort(),
			"topic_pattern":  "park/{park_id}/{system}/{device_type}/{device_id}/properties",
			"alarm_topic":    "park/{park_id}/{system}/{device_type}/{device_id}/alarms",
			"qos":            1,
			"payload_format": "json",
			"direction":      "simulator → broker → platform",
			"note":           "平台订阅 topic 通配符 park/#/properties 即可接收所有设备数据",
		},
		"http": gin.H{
			"callback_url":   s.http.CallbackURL(),
			"method":         "POST",
			"path_pattern":   "/api/v1/devices/{device_id}/events",
			"payload_format": "json",
			"direction":      "simulator → platform",
			"note":           "平台需提供 callback 服务接收 POST 请求",
		},
		"modbus": gin.H{
			"servers": []gin.H{
				{"name": "power_meters", "port": 502, "slave_ids": []int{1, 2, 3, 4, 5}, "device_type": "power_meter"},
				{"name": "chillers", "port": 503, "slave_ids": []int{1, 2}, "device_type": "chiller"},
			},
			"register_type":  "holding register (功能码 03)",
			"data_format":    "float32 big-endian, 2 registers per value",
			"direction":      "platform → simulator (platform as master)",
			"register_map": []gin.H{
				{"device_type": "power_meter", "addr": "40001", "points": "voltage_a, voltage_b, voltage_c, current_a, current_b, current_c, active_power, power_factor, energy"},
				{"device_type": "chiller", "addr": "40001", "points": "running_status, supply_temp, return_temp, power, fault_code"},
			},
			"note":           "寄存器地址按设备类型连续分配，每 2 个寄存器为一个 float32 值",
		},
		"opcua": gin.H{
			"endpoint":       "opc.tcp://localhost:4840",
			"namespace":      "ParkDevices (NS=2)",
			"security":       "none",
			"node_pattern":   "NS=2:{system}/{device_id}",
			"value_type":     "string (JSON)",
			"direction":      "platform → simulator (platform as client)",
			"note":           "每个设备一个 Variable 节点，值为包含所有数据点的 JSON 字符串",
		},
	})
}

func (s *Server) getProtocolStatus(c *gin.Context) {
	status := map[string]any{
		"mqtt":   s.mqtt.Status(),
		"http":   s.http.Status(),
		"modbus": s.modbus.Status(),
		"opcua":  s.opcua.Status(),
	}
	c.JSON(http.StatusOK, status)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

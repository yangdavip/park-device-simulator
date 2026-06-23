package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"park-device-simulator/internal/api"
	"park-device-simulator/internal/config"
	"park-device-simulator/internal/device"
	"park-device-simulator/internal/engine"
	"park-device-simulator/internal/types"
	mqttAdapter "park-device-simulator/internal/protocol/mqtt"
	httpAdapter "park-device-simulator/internal/protocol/http"
	modbusAdapter "park-device-simulator/internal/protocol/modbus"
	opcuaAdapter "park-device-simulator/internal/protocol/opcua"

	// 引入设备包触发 init 注册
	_ "park-device-simulator/internal/device/bas"
	_ "park-device-simulator/internal/device/energy"
	_ "park-device-simulator/internal/device/lighting"
	_ "park-device-simulator/internal/device/security"
	_ "park-device-simulator/internal/device/access"
	_ "park-device-simulator/internal/device/fire"
	_ "park-device-simulator/internal/device/parking"
	_ "park-device-simulator/internal/device/environment"
	_ "park-device-simulator/internal/device/elevator"
	_ "park-device-simulator/internal/device/broadcast"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	configDir := flag.String("config", "configs", "配置文件目录")
	port := flag.Int("port", 8090, "API 服务端口")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] 智慧园区设备模拟器启动中...")

	// 1. 加载配置
	parkCfg, protoCfg, scenarioCfg, err := config.LoadAll(*configDir)
	if err != nil {
		log.Printf("[WARN] 配置加载失败，使用默认配置: %v", err)
		parkCfg = defaultParkConfig()
		protoCfg = defaultProtocolConfig()
		scenarioCfg = defaultScenarioConfig()
	}

	// 2. 初始化引擎
	scenarioEngine := engine.NewScenarioEngine()
	scenarioNames := make([]string, 0, len(scenarioCfg.Scenarios))
	for _, sc := range scenarioCfg.Scenarios {
		scenarioEngine.RegisterScenario(sc.Name, sc.Overrides)
		scenarioNames = append(scenarioNames, sc.Name)
	}

	alarmEngine := engine.NewAlarmEngine()
	registerDefaultAlarmRules(alarmEngine)

	// 3. 初始化协议适配器
	mqttAd := mqttAdapter.NewAdapter(protoCfg.MQTT, parkCfg.Park.ID)
	httpAd := httpAdapter.NewAdapter(protoCfg.HTTP)
	modbusAd := modbusAdapter.NewAdapter(protoCfg.Modbus)
	opcuaAd := opcuaAdapter.NewAdapter(protoCfg.Opcua)

	// 4. 上报回调
	reportFunc := func(d device.Device, data map[string]any) {
		switch d.Protocol() {
		case "mqtt":
			mqttAd.Report(d, data)
		case "http":
			httpAd.Report(d, data)
		case "modbus":
			// Modbus 设备写入寄存器
			if d.Type() == "power_meter" {
				modbusAd.WritePowerMeterData("power_meters", 1, data)
			} else if d.Type() == "chiller" {
				modbusAd.WriteChillerData("chillers", 1, data)
			}
		case "opcua":
			opcuaAd.Report(d, data)
		}
	}

	// 5. 初始化调度引擎
	scheduler := engine.NewScheduler(scenarioEngine, alarmEngine, reportFunc)

	// 6. 根据配置创建设备实例
	deviceCount := createDevicesFromConfig(scheduler, parkCfg)
	log.Printf("[INFO] 已创建 %d 个设备实例", deviceCount)

	// 7. 启动调度
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	// 7.5 启动 OPC UA server
	if protoCfg.Opcua.Server.Enable {
		if err := opcuaAd.Start(ctx); err != nil {
			log.Printf("[WARN] OPC UA server 启动失败: %v", err)
		}
		defer opcuaAd.Close()
	}

	// 8. 启动 API 服务
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	apiServer := api.NewServer(scheduler, scenarioEngine, alarmEngine, mqttAd, httpAd, modbusAd, opcuaAd, scenarioNames)
	apiServer.SetupRoutes(r)

	// 静态文件
	r.StaticFS("/web", http.Dir("./web/dist"))
	r.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/" || c.Request.URL.Path == "/index.html" {
			c.File("./web/dist/index.html")
			return
		}
		c.JSON(404, gin.H{"error": "not found"})
	})

	apiAddr := fmt.Sprintf(":%d", *port)
	go func() {
		log.Printf("[INFO] API 服务启动在 %s", apiAddr)
		if err := r.Run(apiAddr); err != nil {
			log.Fatalf("[FATAL] API 服务启动失败: %v", err)
		}
	}()

	log.Println("[INFO] 智慧园区设备模拟器启动完成")
	log.Println("[INFO] Web UI: http://localhost:" + fmt.Sprint(*port) + "/web/index.html")
	log.Println("[INFO] 注册设备类型:", device.RegisteredTypes())

	// 9. 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("[INFO] 正在关闭...")
	scheduler.Stop()
	mqttAd.Close()
	cancel()
	log.Println("[INFO] 已关闭")
}

// ==================== 默认配置 ====================

func defaultParkConfig() *config.ParkConfig {
	return &config.ParkConfig{
		Park: config.ParkInfo{
			ID:       "P001",
			Name:     "智慧园区演示项目",
			Location: config.Location{City: "深圳", Latitude: 22.5431, Longitude: 114.0579},
			Timezone: "Asia/Shanghai",
		},
		Buildings: []config.BuildingConfig{
			{
				ID: "B001", Name: "A栋办公楼", Floors: 20,
				Systems: []config.SystemConfig{
					{Type: "bas", Enabled: true, Devices: []config.DeviceDef{
						{Type: "ahu", Count: 10, Protocol: "mqtt", Config: map[string]any{"report_interval": "30s"}},
						{Type: "fcu", Count: 20, Protocol: "mqtt"},
						{Type: "fau", Count: 5, Protocol: "mqtt"},
						{Type: "chiller", Count: 2, Protocol: "modbus"},
						{Type: "cooling_tower", Count: 2, Protocol: "mqtt"},
						{Type: "pump", Count: 4, Protocol: "mqtt"},
						{Type: "water_tank", Count: 2, Protocol: "mqtt"},
						{Type: "vent_fan", Count: 6, Protocol: "mqtt"},
						{Type: "heat_exchanger", Count: 1, Protocol: "mqtt"},
					}},
					{Type: "lighting", Enabled: true, Devices: []config.DeviceDef{
						{Type: "lighting_circuit", Count: 10, Protocol: "mqtt"},
						{Type: "lux_sensor", Count: 5, Protocol: "mqtt"},
						{Type: "lamp_controller", Count: 20, Protocol: "mqtt"},
						{Type: "scene_panel", Count: 5, Protocol: "mqtt"},
					}},
					{Type: "energy", Enabled: true, Devices: []config.DeviceDef{
						{Type: "power_meter", Count: 5, Protocol: "modbus"},
						{Type: "water_meter", Count: 2, Protocol: "mqtt"},
						{Type: "gas_meter", Count: 1, Protocol: "mqtt"},
						{Type: "heat_meter", Count: 1, Protocol: "mqtt"},
						{Type: "pv_inverter", Count: 1, Protocol: "mqtt"},
						{Type: "battery_storage", Count: 1, Protocol: "mqtt"},
					}},
					{Type: "security", Enabled: true, Devices: []config.DeviceDef{
						{Type: "ip_camera", Count: 10, Protocol: "mqtt"},
						{Type: "ptz_camera", Count: 4, Protocol: "mqtt"},
						{Type: "video_analyzer", Count: 2, Protocol: "mqtt"},
						{Type: "infrared_beam", Count: 6, Protocol: "http"},
						{Type: "electric_fence", Count: 4, Protocol: "mqtt"},
					}},
					{Type: "access", Enabled: true, Devices: []config.DeviceDef{
						{Type: "access_controller", Count: 8, Protocol: "http"},
						{Type: "face_terminal", Count: 6, Protocol: "http"},
						{Type: "visitor_kiosk", Count: 2, Protocol: "http"},
						{Type: "turnstile", Count: 4, Protocol: "http"},
					}},
					{Type: "fire", Enabled: true, Devices: []config.DeviceDef{
						{Type: "smoke_detector", Count: 20, Protocol: "mqtt"},
						{Type: "temp_detector", Count: 10, Protocol: "mqtt"},
						{Type: "manual_call_point", Count: 5, Protocol: "http"},
						{Type: "fire_hydrant", Count: 6, Protocol: "mqtt"},
						{Type: "sprinkler_pump", Count: 2, Protocol: "mqtt"},
						{Type: "fire_door", Count: 8, Protocol: "mqtt"},
					}},
					{Type: "parking", Enabled: true, Devices: []config.DeviceDef{
						{Type: "lpr_camera", Count: 2, Protocol: "http"},
						{Type: "geomagnetic", Count: 20, Protocol: "mqtt"},
						{Type: "ultrasonic_sensor", Count: 30, Protocol: "mqtt"},
						{Type: "guide_screen", Count: 3, Protocol: "mqtt"},
						{Type: "charging_pile", Count: 4, Protocol: "mqtt"},
					}},
					{Type: "environment", Enabled: true, Devices: []config.DeviceDef{
						{Type: "temp_humidity_sensor", Count: 10, Protocol: "mqtt"},
						{Type: "pm25_sensor", Count: 5, Protocol: "mqtt"},
						{Type: "co2_sensor", Count: 5, Protocol: "mqtt"},
						{Type: "noise_sensor", Count: 3, Protocol: "mqtt"},
						{Type: "gas_sensor", Count: 2, Protocol: "mqtt"},
						{Type: "weather_station", Count: 1, Protocol: "mqtt"},
					}},
					{Type: "elevator", Enabled: true, Devices: []config.DeviceDef{
						{Type: "elevator_controller", Count: 4, Protocol: "mqtt"},
						{Type: "escalator_controller", Count: 2, Protocol: "mqtt"},
					}},
					{Type: "broadcast", Enabled: true, Devices: []config.DeviceDef{
						{Type: "broadcast_terminal", Count: 10, Protocol: "mqtt"},
						{Type: "emergency_broadcast", Count: 2, Protocol: "http"},
					}},
				},
			},
		},
	}
}

func defaultProtocolConfig() *config.ProtocolConfig {
	return &config.ProtocolConfig{
		MQTT: config.MQTTConfig{
			Broker: config.MQTTBroker{Host: "localhost", Port: 1883, Embedded: false},
			Client: config.MQTTClient{Keepalive: 60, QOS: 1, Username: "simulator", Password: "sim123"},
		},
		HTTP: config.HTTPConfig{
			Server:      config.HTTPServer{Port: 8090},
			CallbackURL: "http://localhost:8080/api/v1/devices/events",
			Timeout:     10,
		},
		Modbus: config.ModbusConfig{
			Servers: []config.ModbusServer{
				{Name: "power_meters", Port: 502, SlaveIDs: []byte{1, 2, 3, 4, 5}},
				{Name: "chillers", Port: 503, SlaveIDs: []byte{1, 2}},
			},
		},
	}
}

func defaultScenarioConfig() *config.ScenarioConfig {
	return &config.ScenarioConfig{
		Scenarios: []config.ScenarioDef{
			{Name: "normal_workday", Description: "正常工作日", Type: "schedule"},
			{Name: "weekend", Description: "周末", Type: "schedule"},
			{Name: "holiday", Description: "节假日", Type: "schedule"},
			{Name: "summer_peak", Description: "夏季高温", Type: "schedule",
				Overrides: map[string]any{"outdoor_temp_base": 35, "cooling_load_factor": 1.3}},
			{Name: "winter_peak", Description: "冬季严寒", Type: "schedule",
				Overrides: map[string]any{"outdoor_temp_base": 2}},
			{Name: "fire_emergency", Description: "消防突发事件", Type: "instant", Trigger: &config.TriggerDef{Manual: true}},
			{Name: "power_outage", Description: "停电事件", Type: "instant", Trigger: &config.TriggerDef{Manual: true}},
			{Name: "intrusion", Description: "安防入侵", Type: "instant", Trigger: &config.TriggerDef{Manual: true}},
		},
	}
}

// ==================== 设备创建 ====================

func createDevicesFromConfig(scheduler *engine.Scheduler, cfg *config.ParkConfig) int {
	count := 0
	for _, building := range cfg.Buildings {
		for _, sys := range building.Systems {
			if !sys.Enabled {
				continue
			}
			for _, devDef := range sys.Devices {
				for i := 0; i < devDef.Count; i++ {
					// 生成设备 ID
					deviceID := generateDeviceID(devDef, i, building.ID)

					meta := map[string]any{
						"building": building.ID,
						"floor":    (i % building.Floors) + 1,
					}
					if devDef.Location != "" {
						meta["location"] = devDef.Location
					}

					// 协议类型
					_ = types.ProtocolType(devDef.Protocol) // 协议类型在工厂函数中已指定

					// 创建设备
					d := device.CreateDevice(devDef.Type, deviceID, meta, devDef.Config)
					if d == nil {
						log.Printf("[WARN] 未知设备类型: %s", devDef.Type)
						continue
					}
					// 覆盖设备协议（以配置文件为准）
					if devDef.Protocol != "" {
						d.SetProtocol(types.ProtocolType(devDef.Protocol))
					}
					scheduler.AddDevice(d)
					count++
				}
			}
		}
	}
	return count
}

func generateDeviceID(def config.DeviceDef, index int, buildingID string) string {
	// 简单命名规则：TYPE-BUILDING-SEQ
	return fmt.Sprintf("%s-%s-%03d", def.Type, buildingID, index+1)
}

// ==================== 默认告警规则 ====================

func registerDefaultAlarmRules(alarm *engine.AlarmEngine) {
	// 高温告警
	alarm.AddRule(engine.AlarmRule{
		Name:       "high_temp",
		DeviceType: "ahu",
		Condition: func(data map[string]any) bool {
			if v, ok := data["return_temp"].(float64); ok {
				return v > 30
			}
			return false
		},
		SustainSeconds: 120,
		Level:          "warning",
		Message:        "回风温度过高",
	})

	// 烟雾报警
	alarm.AddRule(engine.AlarmRule{
		Name:       "smoke_alarm",
		DeviceType: "smoke_detector",
		Condition: func(data map[string]any) bool {
			if v, ok := data["smoke_density"].(float64); ok {
				return v > 50
			}
			return false
		},
		SustainSeconds: 5,
		Level:          "critical",
		Message:        "烟雾浓度超标",
	})

	// 电压异常
	alarm.AddRule(engine.AlarmRule{
		Name:       "voltage_abnormal",
		DeviceType: "power_meter",
		Condition: func(data map[string]any) bool {
			if v, ok := data["voltage_a"].(float64); ok {
				return v < 198 || v > 242
			}
			return false
		},
		SustainSeconds: 10,
		Level:          "warning",
		Message:        "电压异常",
	})

	// 燃气泄漏
	alarm.AddRule(engine.AlarmRule{
		Name:       "gas_leak",
		DeviceType: "gas_meter",
		Condition: func(data map[string]any) bool {
			if v, ok := data["leak_alarm"].(bool); ok {
				return v
			}
			return false
		},
		SustainSeconds: 0,
		Level:          "critical",
		Message:        "燃气泄漏报警",
	})
}

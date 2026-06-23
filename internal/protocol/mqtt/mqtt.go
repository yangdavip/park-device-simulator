package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"park-device-simulator/internal/config"
	"park-device-simulator/internal/device"
	"park-device-simulator/internal/types"
)

// ==================== MQTT 适配器 ====================

type Adapter struct {
	mu           sync.RWMutex
	clients      map[string]mqtt.Client // deviceID → client
	brokerCfg    config.MQTTConfig
	parkID       string
	brokerDown   bool // broker 不可用标记
	connectFails int64
}

func NewAdapter(cfg config.MQTTConfig, parkID string) *Adapter {
	return &Adapter{
		clients:   make(map[string]mqtt.Client),
		brokerCfg: cfg,
		parkID:    parkID,
	}
}

// Report 通过 MQTT 上报设备数据
func (a *Adapter) Report(d device.Device, data map[string]any) {
	deviceID := d.ID()

	a.mu.RLock()
	if a.brokerDown {
		a.mu.RUnlock()
		a.mu.Lock()
		a.connectFails++
		a.mu.Unlock()
		return
	}
	a.mu.RUnlock()

	topic := fmt.Sprintf("park/%s/%s/%s/%s/properties", a.parkID, d.System(), d.Type(), deviceID)

	payload := map[string]any{
		"device_id":   deviceID,
		"device_type": d.Type(),
		"system":      d.System(),
		"protocol":    "mqtt",
		"ts":          time.Now().Format(time.RFC3339),
		"data":        data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ERROR] MQTT JSON 编码失败 %s: %v", deviceID, err)
		return
	}

	client := a.getOrCreateClient(deviceID, d.System(), d.Type())
	if client == nil {
		return
	}

	qos := byte(a.brokerCfg.Client.QOS)
	token := client.Publish(topic, qos, false, jsonData)
	if token.WaitTimeout(5 * time.Second) && token.Error() != nil {
		log.Printf("[ERROR] MQTT 发布失败 %s: %v", deviceID, token.Error())
	}
}

// ReportAlarm 上报告警
func (a *Adapter) ReportAlarm(d device.Device, alarm types.Alarm) {
	topic := fmt.Sprintf("park/%s/%s/%s/%s/alarms", a.parkID, d.System(), d.Type(), d.ID())

	payload, _ := json.Marshal(map[string]any{
		"device_id": d.ID(),
		"ts":        time.Now().Format(time.RFC3339),
		"alarm":     alarm,
	})

	client := a.getOrCreateClient(d.ID(), d.System(), d.Type())
	if client != nil {
		client.Publish(topic, byte(a.brokerCfg.Client.QOS), false, payload)
	}
}

// getOrCreateClient 获取或创建 MQTT 客户端
func (a *Adapter) getOrCreateClient(deviceID, system, deviceType string) mqtt.Client {
	a.mu.RLock()
	client, ok := a.clients[deviceID]
	a.mu.RUnlock()
	if ok && client.IsConnected() {
		return client
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// 再次检查
	if client, ok := a.clients[deviceID]; ok && client.IsConnected() {
		return client
	}

	// 创建新客户端
	opts := mqtt.NewClientOptions()
	broker := fmt.Sprintf("tcp://%s:%d", a.brokerCfg.Broker.Host, a.brokerCfg.Broker.Port)
	opts.AddBroker(broker)
	opts.SetClientID(fmt.Sprintf("sim-%s-%d", deviceID, time.Now().UnixNano()%100000))
	opts.SetKeepAlive(time.Duration(a.brokerCfg.Client.Keepalive) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("[DEBUG] MQTT 客户端连接成功: %s → %s", deviceID, broker)
	})
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[WARN] MQTT 连接断开: %s: %v", deviceID, err)
	})

	if a.brokerCfg.Client.Username != "" {
		opts.SetUsername(a.brokerCfg.Client.Username)
		opts.SetPassword(a.brokerCfg.Client.Password)
	}

	client = mqtt.NewClient(opts)
	token := client.Connect()
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		if !a.brokerDown {
			log.Printf("[ERROR] MQTT broker 不可达，静默后续上报: %v", token.Error())
		}
		a.brokerDown = true
		a.connectFails++
		return nil
	}

	a.clients[deviceID] = client
	return client
}

// Close 关闭所有 MQTT 客户端
func (a *Adapter) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, client := range a.clients {
		client.Disconnect(500)
		log.Printf("[DEBUG] MQTT 客户端断开: %s", id)
	}
	a.clients = make(map[string]mqtt.Client)
}

// Status 返回连接状态
func (a *Adapter) Status() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return map[string]any{
		"protocol":      "mqtt",
		"broker":        fmt.Sprintf("%s:%d", a.brokerCfg.Broker.Host, a.brokerCfg.Broker.Port),
		"clients":       len(a.clients),
		"connected":     0, // broker 不通时无已连接客户端
		"connect_fails": a.connectFails,
		"broker_down":   a.brokerDown,
	}
}

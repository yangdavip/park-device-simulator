package opcua

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/server"

	"park-device-simulator/internal/config"
	"park-device-simulator/internal/device"
)

// ==================== OPC UA 适配器 ====================

type Adapter struct {
	cfg      config.OpcuaConfig
	mu       sync.RWMutex
	srv      *server.Server
	ns       *server.NodeNameSpace
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	nodeMap  map[string]*server.Node // deviceID -> node
	deviceCount int
}

func NewAdapter(cfg config.OpcuaConfig) *Adapter {
	return &Adapter{
		cfg:     cfg,
		nodeMap: make(map[string]*server.Node),
	}
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	port := a.cfg.Server.Port
	if port == 0 {
		port = 4840
	}

	endpoint := fmt.Sprintf("opc.tcp://localhost:%d", port)

	srv := server.New(
		server.EndPoint("localhost", port),
		server.ServerName("ParkDeviceSimulator"),
		server.ManufacturerName("ParkSim"),
		server.ProductName("Park Device Simulator"),
		server.SoftwareVersion("1.0.0"),
	)

	ns := server.NewNodeNameSpace(srv, "ParkDevices")

	a.srv = srv
	a.ns = ns
	a.ctx, a.cancel = context.WithCancel(ctx)

	go func() {
		if err := srv.Start(a.ctx); err != nil {
			log.Printf("[ERROR] OPC UA server 启动失败: %v", err)
		}
	}()

	a.running = true
	log.Printf("[INFO] OPC UA server 启动: %s, endpoint: %s", endpoint, endpoint)

	// 等待 server 就绪
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (a *Adapter) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return
	}

	if a.cancel != nil {
		a.cancel()
	}
	if a.srv != nil {
		a.srv.Close()
	}
	a.running = false
	log.Println("[INFO] OPC UA server 已停止")
}

// RegisterDevice 注册设备节点到 OPC UA 地址空间
func (a *Adapter) RegisterDevice(d device.Device) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ns == nil {
		return
	}

	deviceID := d.ID()
	deviceType := d.Type()
	system := d.System()

	// 创建文件夹节点（按系统分组）
	folderName := system
	folderNodeID := ua.NewStringNodeID(a.ns.ID(), folderName)

	// 检查文件夹是否已存在
	folder := a.ns.Node(folderNodeID)
	if folder == nil {
		folder = server.NewFolderNode(folderNodeID, folderName)
		a.ns.AddNode(folder)
	}

	// 创建设备变量节点
	nodeName := deviceID
	varNodeID := ua.NewStringNodeID(a.ns.ID(), nodeName)

	// 初始值
	initialValue := fmt.Sprintf("{'device':'%s','type':'%s','system':'%s','status':'online'}", deviceID, deviceType, system)

	node := server.NewVariableNode(varNodeID, nodeName, initialValue)
	a.ns.AddNode(node)

	a.nodeMap[deviceID] = node
	a.deviceCount++
}

// Report 上报设备数据（更新 OPC UA 节点值）
func (a *Adapter) Report(d device.Device, data map[string]any) {
	a.mu.RLock()
	node, ok := a.nodeMap[d.ID()]
	a.mu.RUnlock()

	if !ok || node == nil {
		// 动态注册
		a.RegisterDevice(d)
		a.mu.RLock()
		node = a.nodeMap[d.ID()]
		a.mu.RUnlock()
		if node == nil {
			return
		}
	}

	// 更新节点值为 JSON 字符串
	jsonStr := dataToJSON(data)
	node.SetAttribute(ua.AttributeIDValue, server.DataValueFromValue(jsonStr))
}

func (a *Adapter) Close() {
	a.Stop()
}

func (a *Adapter) Status() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()

	port := a.cfg.Server.Port
	if port == 0 {
		port = 4840
	}

	return map[string]any{
		"protocol":     "opcua",
		"endpoint":     fmt.Sprintf("opc.tcp://localhost:%d", port),
		"security":     a.cfg.Server.Security,
		"running":      a.running,
		"nodes":        a.deviceCount,
		"namespaces":   2, // 0=标准, 1=ParkDevices
	}
}

func dataToJSON(data map[string]any) string {
	if len(data) == 0 {
		return "{}"
	}
	s := "{"
	first := true
	for k, v := range data {
		if !first {
			s += ","
		}
		s += fmt.Sprintf("\"%s\":%v", k, v)
		first = false
	}
	s += "}"
	return s
}

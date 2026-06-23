package device

import (
	"sync"

	"park-device-simulator/internal/types"
)

// ==================== 设备接口 ====================

// Device 设备接口，所有设备类型必须实现
type Device interface {
	ID() string
	Type() string
	System() string
	Protocol() types.ProtocolType
	SetProtocol(p types.ProtocolType)
	Status() types.DeviceStatus

	Start() error
	Stop() error

	GenerateData(ctx types.ScenarioContext) map[string]any
	CheckAlarms(data map[string]any) []types.Alarm
	LastData() map[string]any
	SetLastData(data map[string]any)
	Meta() map[string]any
}

// ==================== 设备基类 ====================

// BaseDevice 设备基类
type BaseDevice struct {
	mu       sync.RWMutex
	id       string
	dtype    string
	system   string
	protocol types.ProtocolType
	status   types.DeviceStatus
	lastData map[string]any
	meta     map[string]any
}

func NewBaseDevice(id, dtype, system string, protocol types.ProtocolType, meta map[string]any) BaseDevice {
	return BaseDevice{
		id:       id,
		dtype:    dtype,
		system:   system,
		protocol: protocol,
		status:   types.StatusOnline,
		meta:     meta,
	}
}

func (b *BaseDevice) ID() string              { return b.id }
func (b *BaseDevice) Type() string            { return b.dtype }
func (b *BaseDevice) System() string          { return b.system }
func (b *BaseDevice) Protocol() types.ProtocolType { return b.protocol }

func (b *BaseDevice) SetProtocol(p types.ProtocolType) {
	b.mu.Lock()
	b.protocol = p
	b.mu.Unlock()
}
func (b *BaseDevice) Status() types.DeviceStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

func (b *BaseDevice) SetStatus(s types.DeviceStatus) {
	b.mu.Lock()
	b.status = s
	b.mu.Unlock()
}

func (b *BaseDevice) LastData() map[string]any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastData
}

func (b *BaseDevice) SetLastData(d map[string]any) {
	b.mu.Lock()
	b.lastData = d
	b.mu.Unlock()
}

func (b *BaseDevice) Meta() map[string]any { return b.meta }

func (b *BaseDevice) Start() error { b.SetStatus(types.StatusOnline); return nil }
func (b *BaseDevice) Stop() error  { b.SetStatus(types.StatusOffline); return nil }

// ==================== 设备注册中心 ====================

// DeviceFactory 设备工厂函数
type DeviceFactory func(id string, meta map[string]any, cfg map[string]any) Device

var (
	registryMu sync.RWMutex
	registry   = make(map[string]DeviceFactory)
)

func RegisterDevice(deviceType string, factory DeviceFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[deviceType] = factory
}

func CreateDevice(deviceType, id string, meta map[string]any, cfg map[string]any) Device {
	registryMu.RLock()
	factory, ok := registry[deviceType]
	registryMu.RUnlock()
	if !ok {
		return nil
	}
	return factory(id, meta, cfg)
}

func RegisteredTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}

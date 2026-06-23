package modbus

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"park-device-simulator/internal/config"
)

// ==================== Modbus TCP Slave 适配器 ====================

// HoldingRegister 保持寄存器
type HoldingRegister struct {
	Address uint16
	Value   uint16
}

// ModbusSlave Modbus 从站
type ModbusSlave struct {
	slaveID   byte
	registers map[uint16]uint16 // 寄存器地址 → 值
	mu        sync.RWMutex
}

func NewModbusSlave(slaveID byte) *ModbusSlave {
	return &ModbusSlave{
		slaveID:   slaveID,
		registers: make(map[uint16]uint16),
	}
}

func (s *ModbusSlave) SetRegister(addr uint16, value uint16) {
	s.mu.Lock()
	s.registers[addr] = value
	s.mu.Unlock()
}

func (s *ModbusSlave) SetFloat(addr uint16, value float64, scale float64) {
	// 将 float 值缩放后存为两个 16 位寄存器（大端序）
	scaled := uint32(value * scale)
	hi := uint16((scaled >> 16) & 0xFFFF)
	lo := uint16(scaled & 0xFFFF)
	s.mu.Lock()
	s.registers[addr] = hi
	s.registers[addr+1] = lo
	s.mu.Unlock()
}

func (s *ModbusSlave) GetRegister(addr uint16) uint16 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registers[addr]
}

// ==================== Modbus 适配器 ====================

type Adapter struct {
	mu     sync.RWMutex
	servers map[string]*ModbusSlave // slaveKey → slave
	cfg     config.ModbusConfig
}

func NewAdapter(cfg config.ModbusConfig) *Adapter {
	return &Adapter{
		servers: make(map[string]*ModbusSlave),
		cfg:     cfg,
	}
}

// GetOrCreateSlave 获取或创建从站
func (a *Adapter) GetOrCreateSlave(serverName string, slaveID byte) *ModbusSlave {
	key := fmt.Sprintf("%s_%d", serverName, slaveID)
	a.mu.RLock()
	slave, ok := a.servers[key]
	a.mu.RUnlock()
	if ok {
		return slave
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if slave, ok := a.servers[key]; ok {
		return slave
	}
	slave = NewModbusSlave(slaveID)
	a.servers[key] = slave
	log.Printf("[INFO] Modbus 从站创建: %s (slave_id=%d)", key, slaveID)
	return slave
}

// WritePowerMeterData 将电力仪表数据写入 Modbus 寄存器
func (a *Adapter) WritePowerMeterData(serverName string, slaveID byte, data map[string]any) {
	slave := a.GetOrCreateSlave(serverName, slaveID)

	// 寄存器映射（保持寄存器，从 40001 开始，实际地址偏移 0）
	if v, ok := data["voltage_a"].(float64); ok {
		slave.SetFloat(0, v, 10) // ×10
	}
	if v, ok := data["voltage_b"].(float64); ok {
		slave.SetFloat(2, v, 10)
	}
	if v, ok := data["voltage_c"].(float64); ok {
		slave.SetFloat(4, v, 10)
	}
	if v, ok := data["current_a"].(float64); ok {
		slave.SetFloat(6, v, 100) // ×100
	}
	if v, ok := data["current_b"].(float64); ok {
		slave.SetFloat(8, v, 100)
	}
	if v, ok := data["current_c"].(float64); ok {
		slave.SetFloat(10, v, 100)
	}
	if v, ok := data["active_power"].(float64); ok {
		slave.SetFloat(12, v, 10)
	}
	if v, ok := data["power_factor"].(float64); ok {
		slave.SetFloat(14, v, 1000) // ×1000
	}
	if v, ok := data["energy"].(float64); ok {
		slave.SetFloat(16, v, 1)
	}
}

// WriteChillerData 将冷水机组数据写入 Modbus 寄存器
func (a *Adapter) WriteChillerData(serverName string, slaveID byte, data map[string]any) {
	slave := a.GetOrCreateSlave(serverName, slaveID)

	if v, ok := data["chw_supply_temp"].(float64); ok {
		slave.SetFloat(0, v, 10)
	}
	if v, ok := data["chw_return_temp"].(float64); ok {
		slave.SetFloat(2, v, 10)
	}
	if v, ok := data["cw_supply_temp"].(float64); ok {
		slave.SetFloat(4, v, 10)
	}
	if v, ok := data["cw_return_temp"].(float64); ok {
		slave.SetFloat(6, v, 10)
	}
	if v, ok := data["power"].(float64); ok {
		slave.SetFloat(8, v, 10)
	}
	if v, ok := data["load_rate"].(float64); ok {
		slave.SetFloat(10, v, 1)
	}
	if running, ok := data["running"].(bool); ok {
		val := uint16(0)
		if running {
			val = 1
		}
		slave.SetRegister(12, val)
	}
}

// Status 返回状态
func (a *Adapter) Status() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return map[string]any{
		"protocol":   "modbus",
		"slaves":     len(a.servers),
		"servers":    a.cfg.Servers,
	}
}

// Slaves 返回所有从站信息
func (a *Adapter) Slaves() map[string]*ModbusSlave {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]*ModbusSlave, len(a.servers))
	for k, v := range a.servers {
		result[k] = v
	}
	return result
}

// 辅助：读取 32 位值
func readUint32(slave *ModbusSlave, addr uint16) uint32 {
	hi := uint32(slave.GetRegister(addr))
	lo := uint32(slave.GetRegister(addr + 1))
	return binary.BigEndian.Uint32([]byte{
		byte(hi >> 8), byte(hi),
		byte(lo >> 8), byte(lo),
	})
}

package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"park-device-simulator/internal/device"
	"park-device-simulator/internal/types"
)

// ==================== 调度引擎 ====================

type Scheduler struct {
	mu          sync.RWMutex
	devices     []device.Device
	scenario    *ScenarioEngine
	alarmEngine *AlarmEngine
	reportFunc  func(device.Device, map[string]any)
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
}

func NewScheduler(scenario *ScenarioEngine, alarm *AlarmEngine, reportFunc func(device.Device, map[string]any)) *Scheduler {
	return &Scheduler{
		scenario:    scenario,
		alarmEngine: alarm,
		reportFunc:  reportFunc,
	}
}

func (s *Scheduler) AddDevice(d device.Device) {
	s.mu.Lock()
	s.devices = append(s.devices, d)
	s.mu.Unlock()
}

// AddDeviceDynamic 运行时动态添加设备并立即启动上报 goroutine
func (s *Scheduler) AddDeviceDynamic(d device.Device) error {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if !running {
		return fmt.Errorf("scheduler not running")
	}

	if err := d.Start(); err != nil {
		return fmt.Errorf("device start failed: %w", err)
	}

	s.mu.Lock()
	s.devices = append(s.devices, d)
	s.mu.Unlock()

	interval := s.getReportInterval(d)
	s.wg.Add(1)
	go s.runDevice(context.Background(), d, interval)

	log.Printf("[INFO] 动态添加设备: %s (%s/%s)", d.ID(), d.System(), d.Type())
	return nil
}

// RemoveDevice 运行时移除设备
func (s *Scheduler) RemoveDevice(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, d := range s.devices {
		if d.ID() == id {
			d.Stop()
			s.devices = append(s.devices[:i], s.devices[i+1:]...)
			log.Printf("[INFO] 动态移除设备: %s", id)
			return true
		}
	}
	return false
}

func (s *Scheduler) Devices() []device.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]device.Device, len(s.devices))
	copy(result, s.devices)
	return result
}

func (s *Scheduler) DeviceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.devices)
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	for _, d := range s.Devices() {
		if err := d.Start(); err != nil {
			log.Printf("[WARN] 设备 %s 启动失败: %v", d.ID(), err)
			continue
		}
		interval := s.getReportInterval(d)
		s.wg.Add(1)
		go s.runDevice(ctx, d, interval)
	}

	log.Printf("[INFO] 调度引擎启动，共 %d 个设备", s.DeviceCount())
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	s.wg.Wait()

	for _, d := range s.Devices() {
		d.Stop()
	}
	log.Printf("[INFO] 调度引擎停止")
}

func (s *Scheduler) runDevice(ctx context.Context, d device.Device, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次立即上报
	s.tickDevice(d)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickDevice(d)
		}
	}
}

func (s *Scheduler) tickDevice(d device.Device) {
	if d.Status() != types.StatusOnline {
		return
	}

	ctx := s.scenario.BuildContext(time.Now())
	data := d.GenerateData(ctx)
	d.SetLastData(data)

	alarms := d.CheckAlarms(data)
	for _, a := range alarms {
		log.Printf("[ALARM] %s | %s | %s | %s", a.Level, d.ID(), a.Type, a.Message)
	}

	if s.reportFunc != nil {
		s.reportFunc(d, data)
	}
}

func (s *Scheduler) getReportInterval(d device.Device) time.Duration {
	switch d.Type() {
	case "elevator_controller":
		return 5 * time.Second
	case "ultrasonic_sensor", "charging_pile", "pv_inverter":
		return 15 * time.Second
	case "ahu", "fau", "pump", "ip_camera", "ptz_camera", "electric_fence",
		"lighting_circuit", "infrared_beam", "access_controller", "face_terminal",
		"turnstile", "fire_door", "guide_screen", "geomagnetic",
		"broadcast_terminal", "battery_storage", "escalator_controller":
		return 30 * time.Second
	case "fcu", "chiller", "cooling_tower", "vent_fan", "heat_exchanger",
		"lamp_controller", "lux_sensor", "video_analyzer",
		"smoke_detector", "temp_detector", "sprinkler_pump",
		"power_meter", "heat_meter",
		"temp_humidity_sensor", "pm25_sensor", "co2_sensor",
		"noise_sensor", "weather_station":
		return 60 * time.Second
	case "water_tank", "fire_hydrant", "water_meter", "gas_meter", "gas_sensor":
		return 120 * time.Second
	default:
		return 30 * time.Second
	}
}

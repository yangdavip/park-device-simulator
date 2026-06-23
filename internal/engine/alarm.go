package engine

import (
	"sync"
	"time"
)

// ==================== 告警引擎 ====================

type AlarmRule struct {
	Name           string
	DeviceType     string
	Condition      func(data map[string]any) bool
	SustainSeconds int
	Level          string
	Message        string
}

type AlarmEngine struct {
	mu         sync.RWMutex
	rules      []AlarmRule
	violations map[string]time.Time
	alarms     []AlarmRecord
}

type AlarmRecord struct {
	DeviceID  string    `json:"device_id"`
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"ts"`
	Value     any       `json:"value,omitempty"`
}

func NewAlarmEngine() *AlarmEngine {
	return &AlarmEngine{
		violations: make(map[string]time.Time),
	}
}

func (e *AlarmEngine) AddRule(rule AlarmRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

func (e *AlarmEngine) Check(deviceID, deviceType string, data map[string]any) []AlarmRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	var triggered []AlarmRecord
	now := time.Now()

	for _, rule := range e.rules {
		if rule.DeviceType != "" && rule.DeviceType != deviceType {
			continue
		}

		key := deviceID + ":" + rule.Name
		conditionMet := rule.Condition(data)

		if conditionMet {
			startTime, exists := e.violations[key]
			if !exists {
				e.violations[key] = now
				startTime = now
			}

			if rule.SustainSeconds == 0 || now.Sub(startTime).Seconds() >= float64(rule.SustainSeconds) {
				triggered = append(triggered, AlarmRecord{
					DeviceID:  deviceID,
					Type:      rule.Name,
					Level:     rule.Level,
					Message:   rule.Message,
					Timestamp: now,
				})
			}
		} else {
			delete(e.violations, key)
		}
	}

	e.alarms = append(e.alarms, triggered...)
	if len(e.alarms) > 1000 {
		e.alarms = e.alarms[len(e.alarms)-1000:]
	}

	return triggered
}

func (e *AlarmEngine) Alarms(limit int) []AlarmRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if limit <= 0 || limit > len(e.alarms) {
		limit = len(e.alarms)
	}
	result := make([]AlarmRecord, limit)
	copy(result, e.alarms[len(e.alarms)-limit:])
	return result
}

func (e *AlarmEngine) ClearAlarms() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.alarms = nil
	e.violations = make(map[string]time.Time)
}

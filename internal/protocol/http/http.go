package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"park-device-simulator/internal/config"
	"park-device-simulator/internal/device"
)

// ==================== HTTP 适配器 ====================

type Adapter struct {
	mu          sync.RWMutex
	callbackURL string
	timeout     time.Duration
	client      *http.Client
	sentCount   int64
	failCount   int64
	connected   bool // callback server 是否可达
}

func NewAdapter(cfg config.HTTPConfig) *Adapter {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Adapter{
		callbackURL: cfg.CallbackURL,
		timeout:     timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Report 通过 HTTP POST 上报设备事件
func (a *Adapter) Report(d device.Device, data map[string]any) {
	if a.callbackURL == "" {
		return
	}

	payload := map[string]any{
		"device_id":   d.ID(),
		"device_type": d.Type(),
		"system":      d.System(),
		"protocol":    "http",
		"ts":          time.Now().Format(time.RFC3339),
		"data":        data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ERROR] HTTP JSON 编码失败 %s: %v", d.ID(), err)
		return
	}

	url := fmt.Sprintf("%s/api/v1/devices/%s/events", a.callbackURL, d.ID())

	go func() {
		resp, err := a.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			a.mu.Lock()
			a.failCount++
			wasConnected := a.connected
			a.connected = false
			a.mu.Unlock()
			if wasConnected {
				log.Printf("[WARN] HTTP 上报失败 %s: %v", d.ID(), err)
			}
			return
		}
		defer resp.Body.Close()

		a.mu.Lock()
		a.sentCount++
		a.connected = true
		a.mu.Unlock()

		if resp.StatusCode >= 400 {
			log.Printf("[WARN] HTTP 上报 %s 返回 %d", d.ID(), resp.StatusCode)
		}
	}()
}

// SetCallbackURL 动态修改回调地址
func (a *Adapter) SetCallbackURL(url string) {
	a.mu.Lock()
	a.callbackURL = url
	a.mu.Unlock()
}

// Status 返回状态
func (a *Adapter) Status() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return map[string]any{
		"protocol":    "http",
		"callback":    a.callbackURL,
		"sent_count":  a.sentCount,
		"fail_count":  a.failCount,
		"connected":   a.connected,
	}
}

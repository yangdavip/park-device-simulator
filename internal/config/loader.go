package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadParkConfig 加载园区拓扑配置
func LoadParkConfig(path string) (*ParkConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg ParkConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadProtocolConfig 加载协议配置
func LoadProtocolConfig(path string) (*ProtocolConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg ProtocolConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadScenarioConfig 加载场景配置
func LoadScenarioConfig(path string) (*ScenarioConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg ScenarioConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadAll 加载所有配置
func LoadAll(configDir string) (*ParkConfig, *ProtocolConfig, *ScenarioConfig, error) {
	parkCfg, err := LoadParkConfig(filepath.Join(configDir, "park.yaml"))
	if err != nil {
		return nil, nil, nil, err
	}
	protoCfg, err := LoadProtocolConfig(filepath.Join(configDir, "protocols.yaml"))
	if err != nil {
		return nil, nil, nil, err
	}
	scenarioCfg, err := LoadScenarioConfig(filepath.Join(configDir, "scenarios.yaml"))
	if err != nil {
		return nil, nil, nil, err
	}
	return parkCfg, protoCfg, scenarioCfg, nil
}

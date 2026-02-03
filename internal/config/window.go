package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WindowState 定义窗口状态
type WindowState struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Maximised bool `json:"maximised"`
}

// 默认窗口配置
var DefaultWindowState = WindowState{
	Width:  1024,
	Height: 768,
	X:      -1, // -1 表示让系统自动决定位置
	Y:      -1,
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	configDir, _ := os.UserConfigDir()
	appDir := filepath.Join(configDir, "mesh-drop")
	_ = os.MkdirAll(appDir, 0755)
	return filepath.Join(appDir, "window.json")
}

// LoadWindowState 读取配置
func LoadWindowState() WindowState {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultWindowState
	}

	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return DefaultWindowState
	}
	return state
}

// SaveWindowState 保存配置
func SaveWindowState(state WindowState) error {
	path := GetConfigPath()
	data, _ := json.MarshalIndent(state, "", "  ")
	return os.WriteFile(path, data, 0644)
}

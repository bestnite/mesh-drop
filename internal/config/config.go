package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// WindowState 定义窗口状态
type WindowState struct {
	Width     int  `mapstructure:"width"`
	Height    int  `mapstructure:"height"`
	X         int  `mapstructure:"x"`
	Y         int  `mapstructure:"y"`
	Maximised bool `mapstructure:"maximised"`
}

type Config struct {
	mu          sync.RWMutex
	WindowState WindowState `mapstructure:"window_state"`
	SavePath    string      `mapstructure:"save_path"`
}

// 默认窗口配置
var defaultWindowState = WindowState{
	Width:  1024,
	Height: 768,
	X:      -1,
	Y:      -1,
}

func getConfigDir() string {
	configPath, err := os.UserConfigDir()
	if err != nil {
		configPath = "/tmp"
	}
	return filepath.Join(configPath, "mesh-drop")
}

func getUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}

// New 读取配置
func Load() *Config {
	configDir := getConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		slog.Error("Failed to create config directory", "error", err)
	}
	configFile := filepath.Join(configDir, "config.json")

	// 设置默认值
	defaultSavePath := filepath.Join(getUserHomeDir(), "Downloads")
	viper.SetDefault("window_state", defaultWindowState)
	viper.SetDefault("save_path", defaultSavePath)

	viper.SetConfigFile(configFile)
	viper.SetConfigType("json")

	// 尝试读取配置
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, using defaults")
		} else {
			slog.Warn("Failed to read config file, using defaults", "error", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
	}

	return &config
}

// Save 保存配置到磁盘
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	if err := viper.WriteConfig(); err != nil {
		slog.Error("Failed to write config", "error", err)
		return err
	}

	return nil
}

// SetSavePath 修改配置
func (c *Config) SetSavePath(savePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.SavePath = savePath
	viper.Set("save_path", savePath)
}

func (c *Config) GetSavePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SavePath
}

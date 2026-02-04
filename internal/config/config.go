package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
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
	v  *viper.Viper
	mu sync.RWMutex

	WindowState WindowState `mapstructure:"window_state"`
	ID          string      `mapstructure:"id"`
	SavePath    string      `mapstructure:"save_path"`
	HostName    string      `mapstructure:"host_name"`
	AutoAccept  bool        `mapstructure:"auto_accept"`
	SaveHistory bool        `mapstructure:"save_history"`
}

// 默认窗口配置
var defaultWindowState = WindowState{
	Width:  1024,
	Height: 768,
	X:      -1,
	Y:      -1,
}

func GetConfigDir() string {
	configPath, err := os.UserConfigDir()
	if err != nil {
		configPath = "/tmp"
	}
	return filepath.Join(configPath, "mesh-drop")
}

func GetUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}

// New 读取配置
func Load() *Config {
	v := viper.New()
	configDir := GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		slog.Error("Failed to create config directory", "error", err)
	}
	configFile := filepath.Join(configDir, "config.json")

	// 设置默认值
	defaultSavePath := filepath.Join(GetUserHomeDir(), "Downloads")
	v.SetDefault("window_state", defaultWindowState)
	v.SetDefault("save_path", defaultSavePath)
	defaultHostName, err := os.Hostname()
	if err != nil {
		defaultHostName = "localhost"
	}
	v.SetDefault("host_name", defaultHostName)
	v.SetDefault("id", uuid.New().String())

	v.SetConfigFile(configFile)
	v.SetConfigType("json")

	// 尝试读取配置
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, using defaults")
		} else {
			slog.Warn("Failed to read config file, using defaults", "error", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
	}

	config.v = v

	return &config
}

// Save 保存配置到磁盘
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	if err := c.v.WriteConfig(); err != nil {
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
	c.v.Set("save_path", savePath)
	_ = os.MkdirAll(savePath, 0755)
}

func (c *Config) GetSavePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SavePath
}

func (c *Config) SetHostName(hostName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HostName = hostName
	c.v.Set("host_name", hostName)
}

func (c *Config) GetHostName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.HostName
}

func (c *Config) GetID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ID
}

func (c *Config) SetAutoAccept(autoAccept bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AutoAccept = autoAccept
	c.v.Set("auto_accept", autoAccept)
}

func (c *Config) GetAutoAccept() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AutoAccept
}

func (c *Config) SetSaveHistory(saveHistory bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SaveHistory = saveHistory
	c.v.Set("save_history", saveHistory)
}

func (c *Config) GetSaveHistory() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SaveHistory
}

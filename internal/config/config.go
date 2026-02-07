package config

import (
	"log/slog"
	"mesh-drop/internal/security"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// WindowState 定义窗口状态
type WindowState struct {
	Width  int `mapstructure:"width"`
	Height int `mapstructure:"height"`
}

var Version = "next"

type Language string

const (
	LanguageEnglish Language = "en"
	LanguageChinese Language = "zh-Hans"
)

type configData struct {
	WindowState WindowState       `mapstructure:"window_state"`
	ID          string            `mapstructure:"id"`
	PrivateKey  string            `mapstructure:"private_key"`
	PublicKey   string            `mapstructure:"public_key"`
	SavePath    string            `mapstructure:"save_path"`
	HostName    string            `mapstructure:"host_name"`
	AutoAccept  bool              `mapstructure:"auto_accept"`
	SaveHistory bool              `mapstructure:"save_history"`
	TrustedPeer map[string]string `mapstructure:"trusted_peer"` // ID -> PublicKey

	Language       Language `mapstructure:"language"`
	CloseToSystray bool     `mapstructure:"close_to_systray"`
}

type Config struct {
	v    *viper.Viper
	mu   sync.RWMutex
	data configData
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
func Load(defaultState WindowState) *Config {
	v := viper.New()
	configDir := GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		slog.Error("Failed to create config directory", "error", err)
	}
	configFile := filepath.Join(configDir, "config.json")

	// 设置默认值
	defaultSavePath := filepath.Join(GetUserHomeDir(), "Downloads")
	v.SetDefault("window_state", defaultState)
	v.SetDefault("save_path", defaultSavePath)
	defaultHostName, err := os.Hostname()
	if err != nil {
		defaultHostName = "localhost"
	}
	v.SetDefault("host_name", defaultHostName)
	v.SetDefault("id", uuid.New().String())
	v.SetDefault("save_history", true)

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

	// 确保默认保存路径存在
	err = os.MkdirAll(defaultSavePath, 0755)
	if err != nil {
		slog.Error("Failed to create default save path", "path", defaultSavePath, "error", err)
	}

	var data configData
	if err := v.Unmarshal(&data); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
	}

	config := Config{
		v:    v,
		data: data,
	}

	// 如果没有密钥对，生成新的
	if config.data.PrivateKey == "" || config.data.PublicKey == "" {
		priv, pub, err := security.GenerateKey()
		if err != nil {
			slog.Error("Failed to generate identity keys", "error", err)
		} else {
			config.data.PrivateKey = priv
			config.data.PublicKey = pub
			v.Set("private_key", priv)
			v.Set("public_key", pub)
			// 保存新生成的密钥
			if err := config.Save(); err != nil {
				slog.Error("Failed to save generated keys", "error", err)
			}
		}
	}

	// 初始化 TrustedPeer map if nil
	if config.data.TrustedPeer == nil {
		config.data.TrustedPeer = make(map[string]string)
	}

	return &config
}

// Save 保存配置到磁盘
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.save()
}

func (c *Config) save() error {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	if err := c.v.WriteConfig(); err != nil {
		slog.Error("Failed to write config", "error", err)
		return err
	}

	// 设置配置文件权限为 0600 (仅所有者读写)
	configFile := c.v.ConfigFileUsed()
	if configFile != "" {
		if err := os.Chmod(configFile, 0600); err != nil {
			slog.Warn("Failed to set config file permissions", "error", err)
		}
	}

	return nil
}

// update 是一个辅助函数，用于在锁保护下更新配置并保存
func (c *Config) update(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fn()

	if err := c.save(); err != nil {
		slog.Error("Failed to save config", "error", err)
	}
}

// SetSavePath 修改配置
func (c *Config) SetSavePath(savePath string) {
	c.update(func() {
		c.data.SavePath = savePath
		c.v.Set("save_path", savePath)
		_ = os.MkdirAll(savePath, 0755)
	})
}

func (c *Config) GetSavePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.SavePath
}

func (c *Config) SetHostName(hostName string) {
	c.update(func() {
		c.data.HostName = hostName
		c.v.Set("host_name", hostName)
	})
}

func (c *Config) GetHostName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.HostName
}

func (c *Config) GetID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.ID
}

func (c *Config) SetAutoAccept(autoAccept bool) {
	c.update(func() {
		c.data.AutoAccept = autoAccept
		c.v.Set("auto_accept", autoAccept)
	})
}

func (c *Config) GetAutoAccept() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.AutoAccept
}

func (c *Config) SetSaveHistory(saveHistory bool) {
	c.update(func() {
		c.data.SaveHistory = saveHistory
		c.v.Set("save_history", saveHistory)
	})
}

func (c *Config) GetSaveHistory() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.SaveHistory
}

func (c *Config) GetVersion() string {
	return Version
}

func (c *Config) SetWindowState(state WindowState) {
	c.update(func() {
		c.data.WindowState = state
		c.v.Set("window_state", state)
	})
}

func (c *Config) GetWindowState() WindowState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.WindowState
}

func (c *Config) AddTrust(peerID string, publicKey string) {
	c.update(func() {
		if c.data.TrustedPeer == nil {
			c.data.TrustedPeer = make(map[string]string)
		}
		c.data.TrustedPeer[peerID] = publicKey
		c.v.Set("trusted_peer", c.data.TrustedPeer)
	})
}

func (c *Config) GetTrusted() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.TrustedPeer
}

func (c *Config) RemoveTrust(peerID string) {
	c.update(func() {
		delete(c.data.TrustedPeer, peerID)
		c.v.Set("trusted_peer", c.data.TrustedPeer)
	})
}

func (c *Config) IsTrusted(peerID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.data.TrustedPeer[peerID]
	return exists
}

func (c *Config) SetLanguage(language Language) {
	c.update(func() {
		c.data.Language = language
		c.v.Set("language", language)
	})
}

func (c *Config) GetLanguage() Language {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.Language
}

func (c *Config) SetCloseToSystray(closeToSystray bool) {
	c.update(func() {
		c.data.CloseToSystray = closeToSystray
		c.v.Set("close_to_systray", closeToSystray)
	})
}

func (c *Config) GetCloseToSystray() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.CloseToSystray
}

func (c *Config) GetPrivateKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.PrivateKey
}

func (c *Config) GetPublicKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.PublicKey
}

package config

import (
	"encoding/json"
	"log/slog"
	"mesh-drop/internal/security"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// WindowState 定义窗口状态
type WindowState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

var Version = "next"

type Language string

const (
	LanguageEnglish Language = "en"
	LanguageChinese Language = "zh-Hans"
)

type configData struct {
	WindowState WindowState       `json:"window_state"`
	ID          string            `json:"id"`
	PrivateKey  string            `json:"private_key"`
	PublicKey   string            `json:"public_key"`
	SavePath    string            `json:"save_path"`
	HostName    string            `json:"host_name"`
	AutoAccept  bool              `json:"auto_accept"`
	SaveHistory bool              `json:"save_history"`
	TrustedPeer map[string]string `json:"trusted_peer"` // ID -> PublicKey

	Language       Language `json:"language"`
	CloseToSystray bool     `json:"close_to_systray"`
}

type Config struct {
	mu         sync.RWMutex
	data       configData
	configPath string
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
	configDir := GetConfigDir()
	_ = os.MkdirAll(configDir, 0755)
	configFile := filepath.Join(configDir, "config.json")

	// 设置默认值
	defaultSavePath := filepath.Join(GetUserHomeDir(), "Downloads")
	defaultHostName, err := os.Hostname()
	if err != nil {
		defaultHostName = "localhost"
	}

	cfgData := configData{
		WindowState:    defaultState,
		SavePath:       defaultSavePath,
		AutoAccept:     false,
		SaveHistory:    true,
		Language:       LanguageEnglish,
		CloseToSystray: false,
		ID:             uuid.New().String(),
		HostName:       defaultHostName,
		TrustedPeer:    make(map[string]string),
	}

	fileBytes, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read config file", "error", err)
		} else {
			slog.Info("Config file not found, creating new one")
		}
	} else {
		if err := json.Unmarshal(fileBytes, &cfgData); err != nil {
			slog.Error("Failed to unmarshal config", "error", err)
		}
	}

	config := Config{
		data:       cfgData,
		configPath: configFile,
	}

	// 确保默认保存路径存在
	err = os.MkdirAll(defaultSavePath, 0755)
	if err != nil {
		slog.Error("Failed to create default save path", "path", defaultSavePath, "error", err)
	}

	// 如果没有密钥对，生成新的
	if config.data.PrivateKey == "" || config.data.PublicKey == "" {
		priv, pub, err := security.GenerateKey()
		if err != nil {
			slog.Error("Failed to generate identity keys", "error", err)
		} else {
			config.data.PrivateKey = priv
			config.data.PublicKey = pub
		}
	}

	// 初始化 TrustedPeer map if nil
	if config.data.TrustedPeer == nil {
		config.data.TrustedPeer = make(map[string]string)
	}

	// 保存
	if err := config.Save(); err != nil {
		slog.Error("Failed to save config", "error", err)
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
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}

	// 设置配置文件权限为 0600 (仅所有者读写)
	if c.configPath != "" {
		if err := os.WriteFile(c.configPath, jsonData, 0600); err != nil {
			slog.Warn("Failed to write config file", "error", err)
			return err
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

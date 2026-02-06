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
	Width     int  `mapstructure:"width"`
	Height    int  `mapstructure:"height"`
	X         int  `mapstructure:"x"`
	Y         int  `mapstructure:"y"`
	Maximised bool `mapstructure:"maximised"`
}

var Version = "next"

type Language string

const (
	LanguageEnglish Language = "en"
	LanguageChinese Language = "zh-Hans"
)

type Config struct {
	v  *viper.Viper
	mu sync.RWMutex

	WindowState WindowState       `mapstructure:"window_state"`
	ID          string            `mapstructure:"id"`
	PrivateKey  string            `mapstructure:"private_key"`
	PublicKey   string            `mapstructure:"public_key"`
	SavePath    string            `mapstructure:"save_path"`
	HostName    string            `mapstructure:"host_name"`
	AutoAccept  bool              `mapstructure:"auto_accept"`
	SaveHistory bool              `mapstructure:"save_history"`
	TrustedPeer map[string]string `mapstructure:"trusted_peer"` // ID -> PublicKey

	Language Language `mapstructure:"language"`
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

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
	}

	config.v = v

	// 如果没有密钥对，生成新的
	if config.PrivateKey == "" || config.PublicKey == "" {
		priv, pub, err := security.GenerateKey()
		if err != nil {
			slog.Error("Failed to generate identity keys", "error", err)
		} else {
			config.PrivateKey = priv
			config.PublicKey = pub
			v.Set("private_key", priv)
			v.Set("public_key", pub)
			// 保存新生成的密钥
			if err := config.Save(); err != nil {
				slog.Error("Failed to save generated keys", "error", err)
			}
		}
	}

	// 初始化 TrustedPeer map if nil
	if config.TrustedPeer == nil {
		config.TrustedPeer = make(map[string]string)
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

// SetSavePath 修改配置
func (c *Config) SetSavePath(savePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.SavePath = savePath
	c.v.Set("save_path", savePath)
	_ = os.MkdirAll(savePath, 0755)
	_ = c.save()
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
	_ = c.save()
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
	_ = c.save()
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
	_ = c.save()
}

func (c *Config) GetSaveHistory() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SaveHistory
}

func (c *Config) GetVersion() string {
	return Version
}

func (c *Config) SetWindowState(state WindowState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WindowState = state
	c.v.Set("window_state", state)
	_ = c.save()
}

func (c *Config) GetWindowState() WindowState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WindowState
}

func (c *Config) AddTrustedPeer(peerID string, publicKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.TrustedPeer == nil {
		c.TrustedPeer = make(map[string]string)
	}
	c.TrustedPeer[peerID] = publicKey
	c.v.Set("trusted_peer", c.TrustedPeer)
	_ = c.save()
}

func (c *Config) GetTrustedPeer() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TrustedPeer
}

func (c *Config) RemoveTrustedPeer(peerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.TrustedPeer, peerID)
	c.v.Set("trusted_peer", c.TrustedPeer)
	_ = c.save()
}

func (c *Config) IsTrustedPeer(peerID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.TrustedPeer[peerID]
	return exists
}

func (c *Config) SetLanguage(language Language) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Language = language
	c.v.Set("language", language)
	_ = c.save()
}

func (c *Config) GetLanguage() Language {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Language
}

func (c *Config) GetLanguageByString(str string) Language {
	switch str {
	case string(LanguageEnglish):
		return LanguageEnglish
	case string(LanguageChinese):
		return LanguageChinese
	default:
		return LanguageEnglish
	}
}

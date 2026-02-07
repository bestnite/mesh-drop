package transfer

import (
	"encoding/json"
	"log/slog"
	"mesh-drop/internal/config"
	"os"
	"path/filepath"
)

func (s *Service) SaveHistory() {
	if !s.config.GetSaveHistory() {
		return
	}
	configDir := config.GetConfigDir()
	historyPath := filepath.Join(configDir, "history.json")
	tempPath := historyPath + ".tmp"

	// 序列化传输列表
	historyJson, err := json.MarshalIndent(s.GetTransferList(), "", "  ")
	if err != nil {
		slog.Error("Failed to marshal history", "error", err, "component", "transfer")
		return
	}

	// 写入临时文件
	if err := os.WriteFile(tempPath, historyJson, 0644); err != nil {
		slog.Error("Failed to write temp history file", "error", err, "component", "transfer")
		return
	}

	// 原子性重命名
	if err := os.Rename(tempPath, historyPath); err != nil {
		slog.Error("Failed to rename temp history file", "error", err, "component", "transfer")
		// 清理临时文件
		_ = os.Remove(tempPath)
		return
	}

	slog.Info("History saved successfully", "path", historyPath, "component", "transfer")
}

func (s *Service) LoadHistory() {
	configDir := config.GetConfigDir()
	historyPath := filepath.Join(configDir, "history.json")
	file, err := os.Open(historyPath)
	if err != nil {
		return
	}
	defer file.Close()
	var history []*Transfer
	err = json.NewDecoder(file).Decode(&history)
	if err != nil {
		return
	}
	s.StoreTransfersToList(history)
}

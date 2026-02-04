package transfer

import (
	"encoding/json"
	"log/slog"
	"mesh-drop/internal/config"
	"os"
	"path/filepath"
)

func (s *Service) SaveHistory() {
	configDir := config.GetConfigDir()
	historyPath := filepath.Join(configDir, "history.json")
	historyJson, err := json.Marshal(s.GetTransferList())
	if err != nil {
		return
	}
	file, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	_, err = file.Write(historyJson)
	if err != nil {
		slog.Error("Failed to write history", "error", err)
	}
}

func (s *Service) LoadHistory() {
	configDir := config.GetConfigDir()
	historyPath := filepath.Join(configDir, "history.json")
	file, err := os.Open(historyPath)
	if err != nil {
		return
	}
	defer file.Close()
	var history []Transfer
	err = json.NewDecoder(file).Decode(&history)
	if err != nil {
		return
	}
	for _, item := range history {
		s.StoreTransferToList(&item)
	}
}

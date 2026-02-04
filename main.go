package main

import (
	"embed"
	"log/slog"
	"mesh-drop/internal/config"
	"mesh-drop/internal/discovery"
	"mesh-drop/internal/transfer"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

type FilesDroppedEvent struct {
	Files  []string `json:"files"`
	Target string   `json:"target"`
}

func main() {
	state := config.LoadWindowState()

	app := application.New(application.Options{
		Name: "mesh-drop",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// 获取用户目录
	userHomePath, err := os.UserHomeDir()
	if err != nil {
		userHomePath = "/tmp/mesh-drop"
	}

	// 设置默认保存路径
	defaultSavePath := filepath.Join(userHomePath, "Downloads")

	// 创建保存路径
	err = os.MkdirAll(defaultSavePath, 0755)
	if err != nil {
		slog.Error("Failed to create save path", "path", defaultSavePath, "error", err)
	}

	// 文件传输端口
	port := 9989
	name, _ := os.Hostname()

	// 初始化发现服务
	discoveryService := discovery.NewService(app, name, port)
	discoveryService.Start()

	// 初始化传输服务
	transferService := transfer.NewService(app, port, defaultSavePath, discoveryService)
	transferService.Start()

	slog.Info("Backend Service Started", "discovery_port", discovery.DiscoveryPort, "transfer_port", port)

	app.RegisterService(application.NewService(discoveryService))
	app.RegisterService(application.NewService(transferService))

	windows := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "mesh drop",
		Width:          state.Width,
		Height:         state.Height,
		X:              state.X,
		Y:              state.Y,
		EnableFileDrop: true,
	})

	windows.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		details := event.Context().DropTargetDetails()
		app.Event.Emit("files-dropped", FilesDroppedEvent{
			Files:  files,
			Target: details.ElementID,
		})
	})

	application.RegisterEvent[FilesDroppedEvent]("files-dropped")

	// Initialize structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err = app.Run()
	if err != nil {
		panic(err)
	}
}

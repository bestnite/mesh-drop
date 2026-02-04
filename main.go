package main

import (
	"embed"
	"log/slog"
	"mesh-drop/internal/config"
	"mesh-drop/internal/discovery"
	"mesh-drop/internal/transfer"
	"os"

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
	conf := config.Load()

	app := application.New(application.Options{
		Name: "mesh-drop",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// 创建保存路径
	err := os.MkdirAll(conf.SavePath, 0755)
	if err != nil {
		slog.Error("Failed to create save path", "path", conf.SavePath, "error", err)
	}

	// 文件传输端口
	port := 9989

	// 初始化发现服务
	discoveryService := discovery.NewService(conf, app, port)
	discoveryService.Start()

	// 初始化传输服务
	transferService := transfer.NewService(conf, app, port, discoveryService)
	transferService.Start()
	// 加载传输历史
	if conf.GetSaveHistory() {
		transferService.LoadHistory()
	}

	slog.Info("Backend Service Started", "discovery_port", discovery.DiscoveryPort, "transfer_port", port)

	app.RegisterService(application.NewService(discoveryService))
	app.RegisterService(application.NewService(transferService))
	app.RegisterService(application.NewService(conf))

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "mesh drop",
		Width:          conf.WindowState.Width,
		Height:         conf.WindowState.Height,
		X:              conf.WindowState.X,
		Y:              conf.WindowState.Y,
		EnableFileDrop: true,
		Linux: application.LinuxWindow{
			WebviewGpuPolicy: application.WebviewGpuPolicyAlways,
		},
	})

	// 窗口文件拖拽事件
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		details := event.Context().DropTargetDetails()
		app.Event.Emit("files-dropped", FilesDroppedEvent{
			Files:  files,
			Target: details.ElementID,
		})
	})

	// 窗口关闭事件
	window.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		// 保存配置
		x, y := window.Position()
		width, height := window.Size()
		conf.WindowState = config.WindowState{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}
		_ = conf.Save()
		// 保存传输历史
		if conf.GetSaveHistory() {
			transferService.SaveHistory()
		}
	})

	application.RegisterEvent[FilesDroppedEvent]("files-dropped")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err = app.Run()
	if err != nil {
		panic(err)
	}
}

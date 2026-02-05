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
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

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
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.nite07.mesh-drop",
		},
		Icon: icon,
	})

	// 初始化通知服务
	notifier := notifications.New()
	authorized, err := notifier.RequestNotificationAuthorization()
	if err != nil {
		slog.Error("Failed to request notification authorization", "error", err)
	}
	if !authorized {
		slog.Error("Notification authorization not granted")
	}

	port := 9989

	// 初始化发现服务
	discoveryService := discovery.NewService(conf, app, port)
	discoveryService.Start()

	// 初始化传输服务
	transferService := transfer.NewService(conf, app, notifier, port, discoveryService)
	transferService.Start()
	// 加载传输历史
	if conf.GetSaveHistory() {
		transferService.LoadHistory()
	}

	slog.Info("Backend Service Started", "discovery_port", discovery.DiscoveryPort, "transfer_port", port)

	app.RegisterService(application.NewService(discoveryService))
	app.RegisterService(application.NewService(transferService))
	app.RegisterService(application.NewService(conf))
	app.RegisterService(application.NewService(notifier))

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
	// window.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
	// })

	// 应用关闭事件
	app.OnShutdown(func() {
		x, y := window.Position()
		width, height := window.Size()
		conf.SetWindowState(config.WindowState{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		})

		// 保存传输历史
		if conf.GetSaveHistory() {
			transferService.SaveHistory()
		}

		// 保存配置
		err := conf.Save()
		if err != nil {
			slog.Error("Failed to save config", "error", err)
		}
	})

	// 注册事件
	application.RegisterEvent[FilesDroppedEvent]("files-dropped")
	application.RegisterEvent[[]discovery.Peer]("peers:update")
	application.RegisterEvent[application.Void]("transfer:refreshList")

	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// 运行应用
	err = app.Run()
	if err != nil {
		panic(err)
	}
}

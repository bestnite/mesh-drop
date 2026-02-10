package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
	"mesh-drop/internal/config"
	"mesh-drop/internal/discovery"
	"mesh-drop/internal/transfer"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type FilesDroppedEvent struct {
	Files  []File `json:"files"`
	Target string `json:"target"`
}

type App struct {
	app              *application.App
	mainWindows      *application.WebviewWindow
	conf             *config.Config
	discoveryService *discovery.Service
	transferService  *transfer.Service
	notifier         *notifications.NotificationService
}

func init() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
}

func NewApp() *App {
	app := application.New(application.Options{
		Name: "MeshDrop",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.nite07.mesh-drop",
		},
		Icon: icon,
	})

	// 获取默认屏幕大小
	defaultWidth := 1024
	defaultHeight := 768

	screen := app.Screen.GetPrimary()
	if screen != nil {
		defaultWidth = int(float64(screen.Size.Width) * 0.8)
		defaultHeight = int(float64(screen.Size.Height) * 0.8)
		slog.Info(
			"Primary screen found",
			"width",
			screen.Size.Width,
			"height",
			screen.Size.Height,
			"defaultWidth",
			defaultWidth,
			"defaultHeight",
			defaultHeight,
		)
	} else {
		slog.Info("No primary screen found, using defaults")
	}

	conf := config.Load(config.WindowState{
		Width:  defaultWidth,
		Height: defaultHeight,
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "MeshDrop",
		Width:          conf.GetWindowState().Width,
		Height:         conf.GetWindowState().Height,
		EnableFileDrop: true,
		Linux: application.LinuxWindow{
			WebviewGpuPolicy: application.WebviewGpuPolicyAlways,
		},
	})

	return &App{
		app:         app,
		mainWindows: win,
		conf:        conf,
	}
}

func (a *App) registerServices() {
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
	discoveryService := discovery.NewService(a.conf, a.app, port)
	discoveryService.Start()

	// 初始化传输服务
	transferService := transfer.NewService(a.conf, a.app, notifier, port, discoveryService)
	transferService.Start()
	// 加载传输历史
	if a.conf.GetSaveHistory() {
		transferService.LoadHistory()
	}

	a.discoveryService = discoveryService
	a.transferService = transferService
	a.notifier = notifier

	a.app.RegisterService(application.NewService(discoveryService))
	a.app.RegisterService(application.NewService(transferService))
	a.app.RegisterService(application.NewService(a.conf))
	a.app.RegisterService(application.NewService(notifier))
}

func (a *App) registerCustomEvents() {
	application.RegisterEvent[FilesDroppedEvent]("files-dropped")
	application.RegisterEvent[[]discovery.Peer]("peers:update")
	application.RegisterEvent[application.Void]("transfer:refreshList")
}

func (a *App) setupEvents() {
	// 窗口文件拖拽事件
	a.mainWindows.OnWindowEvent(
		events.Common.WindowFilesDropped,
		func(event *application.WindowEvent) {
			files := make([]File, 0)
			for _, file := range event.Context().DroppedFiles() {
				files = append(files, File{
					Name: filepath.Base(file),
					Path: file,
				})
			}
			details := event.Context().DropTargetDetails()
			a.app.Event.Emit("files-dropped", FilesDroppedEvent{
				Files:  files,
				Target: details.ElementID,
			})
		},
	)

	// 窗口关闭事件
	a.mainWindows.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if a.conf.GetCloseToSystray() {
			event.Cancel()
			a.mainWindows.Hide()
			return
		}

		w, h := a.mainWindows.Size()

		a.conf.SetWindowState(config.WindowState{
			Width:  w,
			Height: h,
		})

		slog.Info("Window closed", "width", w, "height", h)
	})

	// 应用关闭事件
	a.app.OnShutdown(func() {
		// 保存传输历史
		if a.conf.GetSaveHistory() {
			// 将 pending 状态的任务改为 canceled
			a.transferService.GetTransferSyncMap().Range(func(key, value any) bool {
				t := value.(*transfer.Transfer)
				if t.Status == transfer.TransferStatusPending {
					t.Status = transfer.TransferStatusCanceled
				}
				return true
			})
			// 保存传输历史
			a.transferService.SaveHistory()
		}
	})
}

func (a *App) setupSystray() {
	systray := a.app.SystemTray.New()
	systray.SetIcon(icon)
	systray.SetLabel("Mesh Drop")

	menu := a.app.NewMenu()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		a.app.Quit()
	})

	systray.OnClick(func() {
		if a.mainWindows.IsVisible() {
			a.mainWindows.Hide()
		} else {
			a.mainWindows.Show()
			a.mainWindows.Focus()
		}
	})

	systray.SetMenu(menu)
}

func (a *App) Run() {
	a.registerServices()
	a.setupSystray()
	a.registerCustomEvents()
	a.setupEvents()
	err := a.app.Run()
	if err != nil {
		panic(err)
	}
}

func main() {
	app := NewApp()
	app.Run()
}

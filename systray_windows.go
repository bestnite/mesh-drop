//go:build windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func setupSystray(app *application.App, window *application.WebviewWindow) {
	systray := app.SystemTray.New()
	systray.SetIcon(icon)
	systray.SetLabel("Mesh Drop")

	menu := app.NewMenu()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	systray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show()
			window.Focus()
		}
	})

	systray.SetMenu(menu)

	window.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		window.Hide()
	})
}

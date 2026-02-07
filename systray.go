//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func setupSystray(app *application.App, window *application.WebviewWindow) {}

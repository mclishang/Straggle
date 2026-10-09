package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"Straggle/internal/i18n"
)

// assets is the production frontend build (vite build), embedded at compile time.
//
//go:embed all:frontend/dist
var assets embed.FS

// appIcon is the tray and window icon.
//
//go:embed build/windows/icon.ico
var appIcon []byte

func main() {
	app := NewApp(appIcon)
	err := wails.Run(&options.App{
		Title:            appName,
		Width:            1024,
		Height:           680,
		MinWidth:         880,
		MinHeight:        520,
		DisableResize:    false,
		BackgroundColour: &options.RGBA{R: 0xF4, G: 0xFB, B: 0xFB, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
		// Second launches hand over to the running window: two instances cannot
		// share the WebView2 user data folder, so the second one dies on startup.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               appName,
			OnSecondInstanceLaunch: func(options.SecondInstanceData) { app.showWindow() },
		},
		// Frameless: the system title bar is replaced by the traffic lights the
		// frontend draws in the top bar, which doubles as the drag region
		// (--wails-draggable: drag).
		Frameless: true,
		Windows: &windows.Options{
			// SystemDefault picks the first-frame theme for WebView2; later
			// changes arrive over the theme:changed event.
			Theme: windows.SystemDefault,
		},
	})
	if err != nil {
		println(i18n.T("app.startFailed", err.Error()))
	}
}

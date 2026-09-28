package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Video Downloader",
		Width:  720,
		Height: 640,
		MinWidth:  640,
		MinHeight: 520,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Frameless window: we draw our own dark title bar (Task 3).
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 20, A: 1},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

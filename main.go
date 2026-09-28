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

// version is the app version reported to the UI. It is kept in step with
// wails.json's info.productVersion.
const version = "0.1.0"

func main() {
	app := NewApp(version)

	err := wails.Run(&options.App{
		Title:     "Video Downloader",
		Width:     720,
		Height:    640,
		MinWidth:  640,
		MinHeight: 520,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Frameless + transparent so our own rounded, dark window chrome shows
		// through (matching the reference design). The rounded corners are drawn
		// in CSS by Layout.tsx.
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
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

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
		// Frameless, but WITH the default window decorations: on Windows 11 the
		// DWM then draws the rounded corners and the drop shadow itself, cleanly.
		// The previous combination (WindowIsTranslucent + WebviewIsTransparent)
		// left a light fringe along the CSS-rounded corner, because the CSS
		// radius and DWM's corner did not match. DWM now owns the shape.
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 0x12, G: 0x12, B: 0x14, A: 0xFF},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			// Keep DWM's rounded corners + Aero shadow (the default).
			DisableFramelessWindowDecorations: false,
			Theme:                             windows.Dark,
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

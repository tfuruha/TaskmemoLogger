package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// ── Single-instance guard ──────────────────────────────────────────────────
	// Prevents two instances from corrupting tags.json simultaneously.
	mutex, lockErr := acquireSingleInstanceLock()
	if lockErr != nil {
		println("Warning: could not acquire instance lock:", lockErr.Error())
		// Non-fatal: continue rather than blocking the user.
	} else if mutex == 0 {
		// Another instance is already running — exit silently.
		os.Exit(0)
	} else {
		defer windows.ReleaseMutex(mutex)
		defer windows.CloseHandle(mutex)
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:       "TaskmemoLogger",
		Width:       520,
		Height:      630,
		MinWidth:    400,
		MinHeight:   460,
		StartHidden: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 27, B: 30, A: 255},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

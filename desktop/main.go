package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	var app *application.App
	term, err := newTerminalService(func(res string) {
		fmt.Println("RELAY_BENCH_RESULT " + res)
		app.Quit()
	})
	if err != nil {
		log.Fatal(err)
	}
	app = application.New(application.Options{
		Name:        "Relay",
		Description: "SSH connection manager and terminal",
		Services:    []application.Service{application.NewService(term)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Relay",
		Width:            1100,
		Height:           700,
		BackgroundColour: application.NewRGB(18, 18, 22),
		URL:              "/",
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

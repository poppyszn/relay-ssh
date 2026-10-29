package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"runtime"

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
		Services:    []application.Service{application.NewService(term), application.NewService(&ProfilesService{}), application.NewService(&ToolsService{}), application.NewService(&FilesService{}), application.NewService(&ForwardsService{})},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	app.Menu.Set(buildMenu(app))
	app.Window.NewWithOptions(windowOptions())
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// windowRadius is the corner radius, in points, of the frameless macOS window.
// macOS draws its own very round corners on native windows; this is ours.
const windowRadius = 10

// customChrome reports whether Relay draws its own window frame and buttons
// (macOS only). RELAY_NATIVE_FRAME=1 falls back to the native title bar.
func customChrome() bool {
	return runtime.GOOS == "darwin" && os.Getenv("RELAY_NATIVE_FRAME") == ""
}

func windowOptions() application.WebviewWindowOptions {
	opts := application.WebviewWindowOptions{
		Title:            "Relay",
		Width:            1280,
		Height:           800,
		MinWidth:         720,
		MinHeight:        420,
		BackgroundColour: application.NewRGB(15, 22, 32),
		URL:              "/",
	}
	if customChrome() {
		// Borderless with our own corners: the tab strip is the title bar and
		// draws the window buttons. A transparent window lets the rounded
		// corners show through, and the shadow follows the rounded shape.
		opts.Frameless = true
		opts.BackgroundType = application.BackgroundTypeTransparent
		opts.Mac = application.MacWindow{CornerRadius: windowRadius}
	} else if runtime.GOOS == "darwin" {
		opts.Mac = application.MacWindow{
			TitleBar:                application.MacTitleBarHiddenInset,
			InvisibleTitleBarHeight: 40,
		}
	}
	return opts
}

// buildMenu returns the application menu. Shell items emit a "menu" event the
// frontend acts on. macOS gets Cmd shortcuts; elsewhere Ctrl+Shift is used so
// shortcuts never collide with terminal control keys (Ctrl+D, Ctrl+W, ...).
func buildMenu(app *application.App) *application.Menu {
	acc := func(mac, other string) string {
		if runtime.GOOS == "darwin" {
			return mac
		}
		return other
	}
	send := func(id string) func(*application.Context) {
		return func(*application.Context) { app.Event.Emit("menu", id) }
	}
	menu := application.NewMenu()
	menu.AddRole(application.AppMenu)

	shell := menu.AddSubmenu("Shell")
	shell.Add("New Tab").SetAccelerator(acc("Cmd+T", "Ctrl+Shift+T")).OnClick(send("new-tab"))
	shell.Add("New Connection…").SetAccelerator(acc("Cmd+N", "Ctrl+Shift+N")).OnClick(send("new-connection"))
	shell.Add("Find…").SetAccelerator(acc("Cmd+F", "Ctrl+Shift+F")).OnClick(send("find"))
	shell.Add("Browse Files…").SetAccelerator(acc("Cmd+Shift+O", "Ctrl+Shift+O")).OnClick(send("files"))
	shell.Add("Port Forwards…").SetAccelerator(acc("Cmd+Shift+P", "Ctrl+Shift+P")).OnClick(send("forwards"))
	shell.Add("Tools…").OnClick(send("tools"))
	shell.Add("Settings…").SetAccelerator(acc("Cmd+,", "Ctrl+Shift+,")).OnClick(send("settings"))
	shell.Add("Quick Open…").SetAccelerator(acc("Cmd+K", "Ctrl+Shift+K")).OnClick(send("quick-open"))
	shell.AddSeparator()
	shell.Add("Split Right").SetAccelerator(acc("Cmd+D", "Ctrl+Shift+D")).OnClick(send("split-right"))
	shell.Add("Split Down").SetAccelerator(acc("Cmd+Shift+D", "Ctrl+Alt+D")).OnClick(send("split-down"))
	shell.AddSeparator()
	shell.Add("Enable Command Blocks (this session)").OnClick(send("enable-blocks"))
	shell.AddSeparator()
	shell.Add("Close Pane").SetAccelerator(acc("Cmd+W", "Ctrl+Shift+W")).OnClick(send("close"))
	shell.Add("Close Tab").SetAccelerator(acc("Cmd+Shift+W", "Ctrl+Alt+W")).OnClick(send("close-tab"))
	shell.Add("Next Tab").SetAccelerator(acc("Cmd+Shift+]", "Ctrl+Tab")).OnClick(send("next-tab"))
	shell.Add("Previous Tab").SetAccelerator(acc("Cmd+Shift+[", "Ctrl+Shift+Tab")).OnClick(send("prev-tab"))

	menu.AddRole(application.EditMenu)
	view := menu.AddSubmenu("View")
	view.Add("Toggle Connections").SetAccelerator(acc("Cmd+B", "Ctrl+Shift+B")).OnClick(send("toggle-sidebar"))
	menu.AddRole(application.WindowMenu)
	return menu
}

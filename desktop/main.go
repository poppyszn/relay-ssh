package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	prepareDirs()
	if os.Getenv("RELAY_BENCH") != "" {
		keychainEnabled = false // test runs must never write to the real Keychain
	}
	var app *application.App
	term, err := newTerminalService(func(res string) {
		fmt.Println("RELAY_BENCH_RESULT " + res)
		app.Quit()
	})
	if err != nil {
		log.Fatal(err)
	}

	p, err := paths.Default()
	if err != nil {
		log.Fatal(err)
	}
	certStorePath := filepath.Join(filepath.Dir(p.StoreFile), "rdp_hosts.toml")
	certStore, err := LoadRdpCertStore(certStorePath)
	if err != nil {
		log.Fatal(err)
	}
	ReapOrphanTunnels() // end ssh tunnels a crashed or force-quit earlier run left behind
	rdpGw, err := NewRdpGateway(certStore, func(ev RdpCertEvent) {
		if app != nil {
			app.Event.Emit("rdp:cert", ev)
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	rdpService := NewRdpService(rdpGw)

	app = application.New(application.Options{
		Name:        "Relay",
		Description: "SSH connection manager and terminal",
		Services: []application.Service{
			application.NewService(term),
			application.NewService(&ProfilesService{}),
			application.NewService(&ToolsService{}),
			application.NewService(&FilesService{}),
			application.NewService(&ForwardsService{}),
			application.NewService(&SnippetsService{}),
			application.NewService(&VaultService{}),
			application.NewService(rdpService),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:    application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	app.OnShutdown(func() {
		getVault().Lock()
		rdpService.CloseAll()
		_ = rdpGw.Close()
	})
	app.Menu.Set(buildMenu(app))
	app.Window.NewWithOptions(windowOptions())
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// windowRadius is the corner radius, in points, of the frameless macOS window: a
// small one. macOS draws very round corners on native windows; this is ours.
// Keep it equal to --window-radius in theme.css.
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
		// On macOS transparency comes from the backdrop, not BackgroundType; without it
		// the window stays opaque and fills the rounded corners with a square background.
		opts.Mac = application.MacWindow{CornerRadius: windowRadius, Backdrop: application.MacBackdropTransparent}
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
	shell.Add("New Folder").SetAccelerator(acc("Cmd+Shift+N", "Ctrl+Alt+N")).OnClick(send("new-folder"))
	shell.Add("New Connection…").SetAccelerator(acc("Cmd+N", "Ctrl+Shift+N")).OnClick(send("new-connection"))
	shell.Add("Find…").SetAccelerator(acc("Cmd+F", "Ctrl+Shift+F")).OnClick(send("find"))
	shell.Add("Browse Files…").SetAccelerator(acc("Cmd+Shift+O", "Ctrl+Shift+O")).OnClick(send("files"))
	shell.Add("Port Forwards…").SetAccelerator(acc("Cmd+Shift+P", "Ctrl+Shift+P")).OnClick(send("forwards"))
	shell.Add("Snippets…").SetAccelerator(acc("Cmd+Shift+S", "Ctrl+Shift+S")).OnClick(send("snippets"))
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
	// Cmd+1..8 pick that tab and Cmd+9 the last one, as in browsers and Terminal. Alt on Windows and
	// Linux, where Ctrl+digit can reach the shell.
	for i := 1; i <= 9; i++ {
		label := fmt.Sprintf("Tab %d", i)
		if i == 9 {
			label = "Last Tab"
		}
		shell.Add(label).SetAccelerator(acc(fmt.Sprintf("Cmd+%d", i), fmt.Sprintf("Alt+%d", i))).OnClick(send(fmt.Sprintf("tab-%d", i)))
	}

	menu.AddRole(application.EditMenu)
	view := menu.AddSubmenu("View")
	view.Add("Toggle Connections").SetAccelerator(acc("Cmd+B", "Ctrl+Shift+B")).OnClick(send("toggle-sidebar"))
	menu.AddRole(application.WindowMenu)
	return menu
}

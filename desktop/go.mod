module github.com/poppyszn/relay-ssh/desktop

go 1.27.3

require (
	filippo.io/age v1.3.2
	github.com/BurntSushi/toml v1.6.0
	github.com/coder/websocket v1.8.15
	github.com/creack/pty v1.1.24
	github.com/poppyszn/relay-ssh v0.0.0
	github.com/wailsapp/wails/v3 v3.1.0
)

require (
	filippo.io/hpke v0.4.0 // indirect
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/term v0.46.0 // indirect
)

replace github.com/poppyszn/relay-ssh => ../

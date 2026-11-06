package main

// version identifies this build. Release builds stamp it with
// -ldflags "-X main.version=$(git describe --tags --always --dirty)"; anything else reports "dev".
var version = "dev"

// Version reports which build of Relay is running, for the status bar. Bound to the frontend.
func (t *ToolsService) Version() string { return version }

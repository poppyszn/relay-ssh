package main

import (
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestWindowOptionsCustomAndNativeFrame(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the custom frame is macOS only")
	}
	t.Setenv("RELAY_NATIVE_FRAME", "")
	o := windowOptions()
	if !o.Frameless || o.Mac.CornerRadius != windowRadius || o.Mac.Backdrop != application.MacBackdropTransparent || o.BackgroundType != application.BackgroundTypeTransparent {
		t.Errorf("custom chrome = frameless:%v radius:%v backdrop:%v bg:%v", o.Frameless, o.Mac.CornerRadius, o.Mac.Backdrop, o.BackgroundType)
	}
	if !customChrome() {
		t.Error("customChrome should be on by default")
	}

	t.Setenv("RELAY_NATIVE_FRAME", "1")
	o = windowOptions()
	if o.Frameless || o.Mac.TitleBar != application.MacTitleBarHiddenInset {
		t.Errorf("native fallback = frameless:%v titlebar:%+v", o.Frameless, o.Mac.TitleBar)
	}
	if customChrome() {
		t.Error("RELAY_NATIVE_FRAME=1 must turn the custom chrome off")
	}
}

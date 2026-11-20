package main

import (
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"
)

func matchNativeTitlebar(window *mygo.Window) {
	if handle := window.NativeHandle(); handle != 0 {
		// Keep the native title and traffic lights, and let the window's theme
		// surface show through instead of AppKit's default title-bar material.
		objc.ID(handle).Send(objc.RegisterName("setTitlebarAppearsTransparent:"), true)
	}
}

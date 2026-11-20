package main

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) windowTitle() string {
	if a.repo() == "" {
		return "Gitviz"
	}
	title := a.repoLabel(a.repo())
	tracking := a.snapshot.Tracking
	branch := tracking.Branch
	if tracking.Detached {
		branch = "detached HEAD"
		if a.snapshot.Head != "" {
			branch += " · " + shortSHA(a.snapshot.Head)
		}
	}
	if branch != "" {
		title += " · " + branch
	}
	return title + " — Gitviz"
}

func (a *App) updateWindowTitle() {
	if a.win != nil && !a.win.IsDestroyed() {
		a.win.SetTitle(a.windowTitle())
	}
}

func (a *App) createMainWindow() {
	a.win = mygo.NewWindow(mygo.WindowOptions{Title: a.windowTitle(), Width: 1380, Height: 900, MinWidth: 1000, MinHeight: 620, Content: ui.View(a.view), StateKey: "main", BackgroundColor: a.windowBackgroundColor()})
	a.syncWindowChrome()
	a.win.OnClose(a.closeMainWindow)
}

func (a *App) closeMainWindow(e *mygo.CloseEvent) {
	if a.quitting {
		return
	}
	// Keep the native view and its pending updates alive while the window is
	// dismissed. A Dock activation can restore the same graph and scroll state.
	e.PreventDefault()
	a.persist()
	if a.win != nil {
		a.win.Hide()
	}
}

func (a *App) showMainWindow() {
	if a.quitting || a.win == nil || a.win.IsDestroyed() {
		return
	}
	a.win.Show()
	a.win.Focus()
}

func (a *App) beginQuit() {
	a.quitting = true
	a.persist()
	if a.cancelLoad != nil {
		a.cancelLoad()
	}
	if a.cancelDetail != nil {
		a.cancelDetail()
	}
	if a.cancelDiff != nil {
		a.cancelDiff()
	}
}

func (a *App) installWindowLifecycle() {
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.beginQuit() })
	mygo.App.OnActivate(func(visible bool) {
		if !visible {
			a.showMainWindow()
		}
	})
}

func (a *App) windowBackgroundColor() string {
	color := a.uiTheme.Surface
	return fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)
}

func (a *App) syncWindowChrome() {
	if a.win == nil || a.win.IsDestroyed() {
		return
	}
	if err := a.win.SetBackgroundColor(a.windowBackgroundColor()); err != nil {
		a.fail(err)
		return
	}
	matchNativeTitlebar(a.win)
}

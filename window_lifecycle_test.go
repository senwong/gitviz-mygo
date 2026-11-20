package main

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestWindowTitleUsesCurrentRepositoryAndCheckedOutBranch(t *testing.T) {
	a := uiFixture()
	a.repoNames = map[string]string{a.repo(): "产品服务"}
	a.filterRef = "refs/remotes/origin/feature"
	if got := a.windowTitle(); got != "产品服务 · release — Gitviz" {
		t.Fatal(got)
	}
	a.snapshot.Tracking.Branch = "feature/ui"
	if got := a.windowTitle(); got != "产品服务 · feature/ui — Gitviz" {
		t.Fatal(got)
	}
	a.snapshot.Head = "abcdef0123456789"
	a.snapshot.Tracking.Detached = true
	if got := a.windowTitle(); got != "产品服务 · detached HEAD · abcdef01 — Gitviz" {
		t.Fatal(got)
	}
	a.snapshot = Snapshot{}
	if got := a.windowTitle(); got != "产品服务 — Gitviz" {
		t.Fatal("loading reused stale branch:", got)
	}
	a.active = -1
	if got := a.windowTitle(); got != "Gitviz" {
		t.Fatal(got)
	}
}

func TestCloseWindowKeepsAppAliveAndQuitAllowsClose(t *testing.T) {
	a := uiFixture()
	cancelled := false
	a.cancelLoad = func() { cancelled = true }
	closeEvent := &mygo.CloseEvent{}
	a.closeMainWindow(closeEvent)
	if !closeEvent.DefaultPrevented() || a.quitting || cancelled {
		t.Fatal("closing window terminated the app or cancelled pending work")
	}
	if a.prefs.LastRepository != a.repo() {
		t.Fatal("closing window did not preserve current repository")
	}
	a.beginQuit()
	quitClose := &mygo.CloseEvent{}
	a.closeMainWindow(quitClose)
	if quitClose.DefaultPrevented() || !cancelled {
		t.Fatal("quit was prevented or pending loads were not cancelled")
	}
}

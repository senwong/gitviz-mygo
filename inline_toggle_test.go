package main

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestCommitMouseToggleDoesNotReopenFromListSelection(t *testing.T) {
	a := uiFixture()
	a.prefs.InlineDetails = true
	a.post = func(func()) {}
	defer func() {
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
	}()
	other := a.snapshot.Commits[0]
	other.SHA = "other-sha"
	other.Subject = "Other commit"
	a.snapshot.Commits = append(a.snapshot.Commits, other)
	a.rebuildVisible()
	tt := ui.NewTester(a.view, 1380, 900)
	for _, step := range []struct {
		label string
		open  bool
		sha   string
	}{
		{"A test commit", true, a.snapshot.Commits[0].SHA},
		{"A test commit", false, a.snapshot.Commits[0].SHA},
		{"A test commit", true, a.snapshot.Commits[0].SHA},
		{"Other commit", true, other.SHA},
		{"Other commit", false, other.SHA},
	} {
		if err := tt.Click(step.label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		_, shown := tt.Find("Inline commit details")
		if shown != step.open || a.detailVisible != step.open || a.revision != step.sha {
			t.Fatalf("click %q: shown=%v detailVisible=%v revision=%s want open=%v sha=%s", step.label, shown, a.detailVisible, a.revision, step.open, step.sha)
		}
	}
	tt.Key(0, ui.KeyEnter)
	if !a.detailVisible {
		t.Fatal("Enter should reopen selected commit")
	}
	tt.Key(0, ui.KeyEnter)
	if !a.detailVisible {
		t.Fatal("keyboard activation should keep details open")
	}
}

func TestExpandedCommitBackgroundIsContinuousAcrossGraphAndDetails(t *testing.T) {
	a := uiFixture()
	a.prefs.Theme = "One Dark Pro"
	a.setTheme()
	a.prefs.InlineDetails = true
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.selected = 0
	a.listSelected = 0
	a.detailVisible = true
	tt := ui.NewTester(a.view, 1380, 900)
	panel, ok := tt.Find("Inline commit details")
	if !ok {
		t.Fatal("inline details missing")
	}
	title, _ := tt.Find("A test commit")
	frame := tt.Image()
	for _, point := range [][2]int{
		{30, int(panel.Y + panel.H - 15)},
		{int(panel.X + 15), int(panel.Y + panel.H - 15)},
		{int(panel.X + panel.W*.75), int(panel.Y + panel.H - 15)},
	} {
		pixel := frame.RGBAAt(point[0], point[1])
		surface := a.uiTheme.Surface
		if !nearColor(pixel.R, pixel.G, pixel.B, surface) {
			t.Fatalf("background seam at %v: pixel=%v surface=%v", point, pixel, surface)
		}
	}
	pixel := frame.RGBAAt(700, int(title.Y+5))
	selected := a.listTheme.Accent
	if !nearColor(pixel.R, pixel.G, pixel.B, selected) {
		t.Fatal("expanded commit row lost selection highlight")
	}
	captureMigrationPreview(t, tt, "continuous-inline-background")
}

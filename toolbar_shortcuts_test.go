package main

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestToolbarShowsActualShortcutsWithoutOverflow(t *testing.T) {
	for _, size := range [][2]int{{1380, 900}, {1000, 620}} {
		for _, loading := range []bool{false, true} {
			a := uiFixture()
			a.loading = loading
			tt := ui.NewTester(a.view, size[0], size[1])
			for _, key := range []string{"⌘P", "⌘B", "⌘F", "⌘R", "⌘,"} {
				if !tt.HasText(key) {
					t.Fatal("missing toolbar shortcut", key)
				}
			}
			for _, label := range []string{"Switch repository", "Select branches", "Remotes", "Find", "Actions", "Fetch remotes", "Refresh graph", "Settings"} {
				r, ok := tt.Find(label)
				if !ok || r.X < 0 || r.X+r.W > float32(size[0]) || r.Y+r.H > 54 {
					t.Fatalf("toolbar %q overflows %dx%d loading=%v: %+v", label, size[0], size[1], loading, r)
				}
			}
			captureMigrationPreview(t, tt, fmt.Sprintf("toolbar-shortcuts-%d-loading-%v", size[0], loading))
			if err := tt.Click("Find"); err != nil {
				t.Fatal(err)
			}
			if !a.showSearch {
				t.Fatal("Find no longer opens search")
			}
		}
	}
}

func TestToolbarRepositoryHintFollowsConfiguredShortcut(t *testing.T) {
	a := uiFixture()
	a.prefs.RepositoryShortcut = "cmd-shift-p"
	tt := ui.NewTester(a.view, 1000, 620)
	if !tt.HasText("⌘⇧P") || tt.HasText("⌘P") {
		t.Fatal("repository hint ignored configured shortcut")
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyP)
	if !a.showRepos {
		t.Fatal("displayed shortcut does not open repository picker")
	}
	if got := shortcutLabel("ctrl-alt-b"); got != "⌃⌥B" {
		t.Fatal(got)
	}
}

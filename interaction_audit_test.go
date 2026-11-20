package main

import (
	"context"
	"github.com/egoist/mygo/ui"
	"os"
	"strings"
	"testing"
)

func TestCommandSearchKeyboardAndEscape(t *testing.T) {
	a := uiFixture()
	a.showCommands = true
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Type("stashes")
	if !tt.HasText("Show stashes") || tt.HasText("Push current branch") {
		t.Fatal("command filtering failed")
	}
	tt.Key(0, ui.KeyEnter)
	if a.showCommands || !a.showStashes {
		t.Fatal("Enter did not execute filtered command")
	}
	tt.Key(0, ui.KeyEscape)
	if a.showStashes {
		t.Fatal("Escape did not dismiss stashes")
	}
	a.showCommands = true
	tt.Frame()
	tt.Type("missing-command")
	if !tt.HasText("No matching commands") {
		t.Fatal("missing empty command state")
	}
	tt.Key(0, ui.KeyEnter)
	if !a.showCommands {
		t.Fatal("empty command result executed")
	}
	tt.Key(0, ui.KeyEscape)
	if a.showCommands {
		t.Fatal("Escape did not dismiss commands")
	}
}

func TestInteractionScreenshotInventory(t *testing.T) {
	states := []struct {
		name  string
		setup func(*App)
	}{
		{"main", func(a *App) {}},
		{"palette-open", func(a *App) { a.openRepositoryPicker() }},
		{"commands-open", func(a *App) { a.showCommands = true }},
		{"branches-open", func(a *App) {
			a.showBranches = true
			a.snapshot.Branches = []Branch{{Name: "release", Full: "refs/heads/release", Current: true}, {Name: "origin/release", Full: "refs/remotes/origin/release", Remote: true}}
		}},
		{"stashes-open", func(a *App) { a.showStashes = true }},
		{"stashes-populated", func(a *App) {
			a.showStashes = true
			a.snapshot.Stashes = []Stash{{Ref: "stash@{0}", Message: "WIP on release: pending changes"}}
		}},
		{"prompt-open", func(a *App) {
			a.ask("Create branch", "Enter a branch name. It will start at HEAD.", "Create", func(string) {})
		}},
		{"checkout-sync", func(a *App) {
			a.syncPrompt = &SyncPrompt{Repo: a.repo(), Tracking: a.snapshot.Tracking}
			a.syncOpen = true
		}},
		{"search-empty", func(a *App) { a.showSearch = true; a.query = "no-result" }},
		{"error", func(a *App) { a.errorText = "Git operation failed: working tree contains local changes" }},
	}
	for _, section := range settingsSections {
		section := section
		states = append(states, struct {
			name  string
			setup func(*App)
		}{"settings-" + section, func(a *App) { a.showSettings = true; a.settingsSection = section }})
	}
	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			a := uiFixture()
			if repo := os.Getenv("GITVIZ_AUDIT_REPO"); repo != "" {
				a.repos = []string{repo}
				a.active = 0
				var err error
				a.snapshot, err = loadSnapshotConfig(context.Background(), repo, 100, a.config, "")
				if err != nil {
					t.Fatal(err)
				}
				a.rebuildVisible()
			}
			state.setup(a)
			tt := ui.NewTester(a.view, 1380, 900)
			captureMigrationPreview(t, tt, state.name)
			tt.Key(0, ui.KeyEscape)
			if a.showRepos || a.showCommands || a.showBranches || a.showStashes || a.showSettings || a.promptOpen || a.syncOpen {
				t.Fatal("Escape left modal open")
			}
			tt.SetSize(1000, 620)
			state.setup(a)
			tt.Frame()
			captureMigrationPreview(t, tt, state.name+"-minimum")
		})
	}
}

func TestModalDoesNotTriggerBackgroundShortcuts(t *testing.T) {
	a := uiFixture()
	a.openRepositoryPicker()
	tt := ui.NewTester(a.view, 1380, 900)
	before := a.detailVisible
	tt.Key(ui.Cmd, ui.KeyD)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Key(ui.Cmd, ui.KeyComma)
	if a.detailVisible != before || a.showBranches || a.showSettings {
		t.Fatal("modal allowed background shortcuts")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Key(ui.Cmd, ui.KeyB)
	if !a.showBranches {
		t.Fatal("shortcut did not resume after modal closed")
	}
}

func TestDetailAndContextMenuScreenshots(t *testing.T) {
	a := uiFixture()
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.detailVisible = true
	a.files = []FileChange{{Path: "src/main.go", Status: "M"}, {Path: "src/view.go", Status: "A"}}
	a.rebuildFileRows()
	a.diffLines = []string{"diff --git a/src/main.go b/src/main.go", "@@ -1 +1 @@", "-old implementation", "+new implementation"}
	tt := ui.NewTester(a.view, 1380, 900)
	captureMigrationPreview(t, tt, "detail-open")
	captureMigrationPreview(t, tt, "diff-open")
	if err := tt.RightClick("A test commit"); err != nil {
		t.Fatal(err)
	}
	labels := strings.Join(tt.Menu(), "|")
	for _, label := range []string{"Copy Commit Hash to Clipboard", "Create Branch…", "Cherry Pick…", "Revert…", "Reset current branch to this Commit…"} {
		if !strings.Contains(labels, label) {
			t.Fatalf("missing context action %s", label)
		}
	}
	captureMigrationPreview(t, tt, "menu-open")
	tt.CloseMenu()
	for _, label := range []string{"Checkout…", "Create Branch…", "Add Tag…", "Cherry Pick…", "Revert…", "Reset current branch to this Commit…"} {
		if err := tt.RightClick("A test commit"); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(label); err != nil {
			t.Fatal(err)
		}
		if a.contextForm == nil {
			t.Fatalf("%s missing confirmation", label)
		}
		tt.Key(0, ui.KeyEscape)
		if a.contextForm != nil {
			t.Fatalf("%s not cancelled", label)
		}
		if len(a.busy) != 0 {
			t.Fatal("cancel started write")
		}
	}
}

func TestBranchAndPromptFocus(t *testing.T) {
	a := uiFixture()
	a.showBranches = true
	a.snapshot.Branches = []Branch{{Name: "main", Full: "refs/heads/main", Current: true}, {Name: "feature", Full: "refs/heads/feature"}}
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Type("missing-branch")
	if !tt.HasText("No matching branches") {
		t.Fatal("branch input did not receive focus or empty state missing")
	}
	captureMigrationPreview(t, tt, "branches-empty")
	tt.Key(0, ui.KeyEscape)
	var submitted string
	a.ask("Example", "Type a value", "Confirm", func(value string) { submitted = value })
	tt.Frame()
	tt.Type("typed value")
	tt.Key(0, ui.KeyEnter)
	if submitted != "typed value" || a.promptOpen {
		t.Fatalf("focused prompt submission failed: %q", submitted)
	}
	a.prefs.Theme = "MyGo Light"
	a.setTheme()
	tt.Frame()
	captureMigrationPreview(t, tt, "light-main")
}

func TestAuditRepositoryDetails(t *testing.T) {
	repo := os.Getenv("GITVIZ_AUDIT_REPO")
	if repo == "" {
		t.Skip("set GITVIZ_AUDIT_REPO for real repository screenshots")
	}
	a := uiFixture()
	a.repos = []string{repo}
	a.active = 0
	var err error
	a.snapshot, err = loadSnapshotConfig(context.Background(), repo, 100, a.config, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.snapshot.Commits) == 0 {
		t.Fatal("audit repository has no commits")
	}
	a.rebuildVisible()
	a.selectedCommit = &a.snapshot.Commits[0]
	a.selected = 0
	a.revision = a.selectedCommit.SHA
	a.detailVisible = true
	a.files, err = changedFiles(context.Background(), repo, a.revision)
	if err != nil {
		t.Fatal(err)
	}
	a.rebuildFileRows()
	if len(a.files) > 0 {
		a.selectedFile = 0
		text, err := fileDiff(context.Background(), repo, a.revision, "", a.files[0])
		if err != nil {
			t.Fatal(err)
		}
		a.diffLines = strings.Split(text, "\n")
	}
	tt := ui.NewTester(a.view, 1380, 900)
	captureMigrationPreview(t, tt, "detail-real")
	tt.SetSize(1000, 620)
	captureMigrationPreview(t, tt, "detail-real-minimum")
}

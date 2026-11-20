package main

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestAllThemeAccentTextHasReadableContrast(t *testing.T) {
	for _, name := range themeNames() {
		a := uiFixture()
		a.prefs.Theme = name
		a.setTheme()
		first, second := colorLuminance(a.uiTheme.Accent), colorLuminance(a.uiTheme.AccentText)
		ratio := (math.Max(first, second) + .05) / (math.Min(first, second) + .05)
		if ratio < 4.5 {
			t.Fatalf("%s selected text contrast %.2f is unreadable", name, ratio)
		}
	}
}

func TestShadcnThemeAndSettingsSpacingPreviews(t *testing.T) {
	a := uiFixture()
	a.prefs.Theme = "Shadcn Dark"
	a.setTheme()
	a.showSettings = true
	a.settingsSection = "Appearance"
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.Click("Shadcn Dark"); err != nil {
		t.Fatal(err)
	}
	captureMigrationPreview(t, tt, "shadcn-readable-theme-selection")
	tt.Key(0, ui.KeyEscape)
	a.settingsSection = "Menus"
	tt.Frame()
	first, ok := tt.Find("Copy Commit Hash to Clipboard")
	second, ok2 := tt.Find("Copy Commit Subject to Clipboard")
	if !ok || !ok2 || second.Y-first.Y < 32 {
		t.Fatalf("menu options still overlap: first=%+v second=%+v", first, second)
	}
	captureMigrationPreview(t, tt, "spaced-menu-settings")
	a.settingsSection = "Remotes"
	tt.Frame()
	auto, ok := tt.Find("Automatically fetch when switching repositories")
	header, ok2 := tt.Find("Configured remotes")
	if !ok || !ok2 || header.Y-auto.Y-auto.H < 30 {
		t.Fatalf("automatic fetch still touches next section: auto=%+v header=%+v", auto, header)
	}
	captureMigrationPreview(t, tt, "spaced-remote-settings")
}

func TestRepositoryQuickOpenGreaterThanSwitchesToCommands(t *testing.T) {
	a := uiFixture()
	tt := ui.NewTester(a.view, 1380, 900)
	for _, query := range []string{">", "> Fetch", ">pull"} {
		tt.Key(ui.Cmd, ui.KeyP)
		if !a.showRepos {
			t.Fatal("Cmd+P did not open repositories")
		}
		tt.Type(query)
		tt.Frame()
		if a.showRepos || !a.showCommands || !tt.Focused("Search commands") {
			t.Fatalf("%q did not switch mode/focus", query)
		}
		want := strings.TrimSpace(strings.TrimPrefix(query, ">"))
		if strings.TrimSpace(a.commandQuery) != want {
			t.Fatalf("query lost during switch: got=%q want=%q", a.commandQuery, want)
		}
		if want == "Fetch" && !tt.HasText("Fetch all remotes") {
			t.Fatal("command filter lost query after >")
		}
		captureMigrationPreview(t, tt, "quick-open-command-mode")
		tt.Key(0, ui.KeyEscape)
		if a.showCommands {
			t.Fatal("Escape did not close command mode")
		}
	}
	a.repoQuery = "> Refresh"
	if !a.switchRepositoryQueryToCommands() {
		t.Fatal("pasted command prefix ignored")
	}
	tt.Frame()
	if a.commandQuery != "Refresh" || !tt.HasText("Refresh graph") {
		t.Fatal("pasted query was cleared on command open")
	}
}

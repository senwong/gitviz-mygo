package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func assertNoReviewControls(t *testing.T, tt *ui.Tester) {
	t.Helper()
	for _, label := range []string{"Start review", "End review", "Code reviews", "Files & Review", "Reviewed: file.txt"} {
		if tt.HasText(label) {
			t.Fatalf("unrequested review control remains: %s", label)
		}
	}
}

func TestDetailViewsAndMenusHaveNoReviewFeature(t *testing.T) {
	for _, inline := range []bool{false, true} {
		a := uiFixture()
		a.prefs.InlineDetails = inline
		a.selectedCommit = &a.snapshot.Commits[0]
		a.revision = a.selectedCommit.SHA
		a.detailVisible = true
		a.selected = 0
		a.listSelected = 0
		a.files = []FileChange{{Path: "file.txt", Status: "M"}}
		a.selectedFile = 0
		a.rebuildFileRows()
		a.diffLines = []string{"@@ -1 +1 @@", "-old", "+new"}
		tt := ui.NewTester(a.view, 1380, 900)
		assertNoReviewControls(t, tt)
		if _, ok := tt.Find("Tree / List"); !ok {
			t.Fatal("file toolbar lost tree/list control")
		}
		if err := tt.RightClick("file.txt"); err != nil {
			t.Fatal(err)
		}
		for _, label := range tt.Menu() {
			if strings.Contains(strings.ToLower(label), "review") {
				t.Fatalf("review file action remains: %s", label)
			}
		}
		tt.CloseMenu()
		a.largeDiffVisible = inline
		tt.Frame()
		assertNoReviewControls(t, tt)
		captureMigrationPreview(t, tt, "files-without-review")
		a.largeDiffVisible = false
		a.showSettings = true
		a.settingsSection = "Files"
		tt.Frame()
		assertNoReviewControls(t, tt)
		if !tt.HasText("File tree") {
			t.Fatal("file settings disappeared")
		}
		a.showSettings = false
		a.showCommands = true
		a.commandQuery = "review"
		tt.Frame()
		for _, text := range tt.Texts() {
			if strings.Contains(text, "code review") {
				t.Fatalf("review command remains: %s", text)
			}
		}
	}
}

func TestProductionContextMenuUsesApplicationPopover(t *testing.T) {
	for _, theme := range []string{"One Dark Pro", "GitHub Light"} {
		t.Run(theme, func(t *testing.T) {
			a := uiFixture()
			a.prefs.Theme = theme
			a.setTheme()
			tt := ui.NewTester(a.view, 1380, 900)
			if err := tt.RightClick("A test commit"); err != nil {
				t.Fatal(err)
			}
			row, ok := tt.Find("Add Tag…")
			if !ok || row.H <= 0 {
				t.Fatal("menu was not rendered inside the application")
			}
			tt.Move(row.X+row.W/2, row.Y+row.H/2)
			captureMigrationPreview(t, tt, "application-context-menu-"+theme)
			tt.Key(0, ui.KeyEscape)
			if len(tt.Menu()) > 0 {
				t.Fatal("Escape did not close context menu")
			}
			if a.contextForm != nil || len(a.busy) > 0 {
				t.Fatal("cancelled menu started an action")
			}
		})
	}
}

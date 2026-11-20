package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFileSelectionUsesQuietThemeHighlight(t *testing.T) {
	for _, theme := range []string{"One Dark Pro", "GitHub Light"} {
		t.Run(theme, func(t *testing.T) {
			a := uiFixture()
			a.prefs.Theme = theme
			a.setTheme()
			a.prefs.InlineDetails = true
			a.selectedCommit = &a.snapshot.Commits[0]
			a.revision = a.selectedCommit.SHA
			a.detailVisible = true
			a.selected = 0
			a.listSelected = 0
			a.files = []FileChange{{Path: "file.txt", Status: "M"}}
			a.selectedFile = -1
			a.rebuildFileRows()
			tt := ui.NewTester(a.view, 1380, 900)
			rect, ok := tt.Find("File diff: file.txt")
			if !ok {
				t.Fatal("file missing")
			}
			pixel := tt.Image().RGBAAt(int(rect.X+rect.W-15), int(rect.Y+2))
			if !nearColor(pixel.R, pixel.G, pixel.B, a.uiTheme.Surface) {
				t.Fatal("unselected file has a selection background")
			}
			captureMigrationPreview(t, tt, "unselected-file-"+theme)
			a.selectedFile = 0
			a.diffLines = []string{"@@ -1 +1 @@", "-old", "+new"}
			a.rebuildFileRows()
			a.largeDiffVisible = true
			tt.Frame()
			rect, ok = tt.Find("File diff: file.txt")
			if !ok {
				t.Fatal("selected file missing")
			}
			pixel = tt.Image().RGBAAt(int(rect.X+rect.W-15), int(rect.Y+2))
			if !nearColor(pixel.R, pixel.G, pixel.B, a.listTheme.Accent) {
				t.Fatalf("selected file did not use quiet selection color: pixel=%v selected=%v", pixel, a.listTheme.Accent)
			}
			captureMigrationPreview(t, tt, "selected-file-"+theme)
		})
	}
}

package main

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDiffHasRoomAtMinimumSizeAndCanExpandWithoutLosingSelection(t *testing.T) {
	a := uiFixture()
	a.selectedCommit = &a.snapshot.Commits[0]
	a.selectedCommit.Body = "First message line\nSecond message line"
	a.revision = a.selectedCommit.SHA
	a.detailVisible = true
	a.files = []FileChange{{Path: "app/service/convert/convert.ts", Status: "M"}}
	a.selectedFile = 0
	a.rebuildFileRows()
	a.diffLines = []string{"@@ -1,30 +1,30 @@"}
	for i := 0; i < 30; i++ {
		a.diffLines = append(a.diffLines, fmt.Sprintf("+line %d", i))
	}
	tt := ui.NewTester(a.view, 1000, 620)
	viewport, ok := tt.Find("Diff viewport")
	if !ok || viewport.H < 160 {
		t.Fatalf("code viewport too short at minimum window: %+v", viewport)
	}
	captureMigrationPreview(t, tt, "larger-diff-minimum")
	detail, _ := tt.Find("Commit details")
	y := detail.Y - .5
	tt.Press(100, y)
	tt.Move(100, y-30)
	tt.Release(100, y-30)
	if a.detailPercent >= 40 {
		t.Fatal("divider cannot enlarge details")
	}
	percent := a.detailPercent
	if err := tt.Click("Expand details"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	full, ok := tt.Find("Diff viewport")
	if !ok || full.H <= viewport.H+150 || !a.detailMaximized {
		t.Fatalf("expand did not enlarge diff: before=%+v after=%+v", viewport, full)
	}
	if a.selectedFile != 0 || a.revision != a.selectedCommit.SHA {
		t.Fatal("expanding lost selected commit or file")
	}
	captureMigrationPreview(t, tt, "expanded-diff")
	if err := tt.Click("Restore graph"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.detailMaximized || a.detailPercent != percent || !tt.HasText("Graph") {
		t.Fatal("restore lost graph split size")
	}
	if err := tt.Click("Expand details"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Close details"); err != nil {
		t.Fatal(err)
	}
	if a.detailMaximized || a.detailVisible {
		t.Fatal("close did not restore graph")
	}
}

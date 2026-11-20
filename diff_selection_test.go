package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDiffSelectionCopiesAcrossLinesWithoutGutter(t *testing.T) {
	for _, search := range []bool{false, true} {
		t.Run(fmt.Sprintf("search=%t", search), func(t *testing.T) {
			a := diffFindFixture()
			a.diffLines = []string{"@@ -1,3 +1,3 @@", "-const old = 1;", "", "+const 新值 = '👋';", " unchanged"}
			if search {
				a.openSearch()
				a.diffQuery = "const"
			}
			tt := ui.NewTester(a.view, 1000, 700)
			first, ok := tt.Find("-const old = 1;")
			if !ok {
				t.Fatal("first line missing")
			}
			last, ok := tt.Find("+const 新值 = '👋';")
			if !ok {
				t.Fatal("last line missing")
			}
			// From the start of the first line to the end of the third, including a blank line.
			tt.Press(first.X, first.Y+first.H/2)
			tt.Move(last.X+last.W-1, last.Y+last.H/2)
			tt.Release(last.X+last.W-1, last.Y+last.H/2)
			tt.Key(ui.Cmd, ui.KeyC)
			want := "-const old = 1;\n\n+const 新值 = '👋';"
			if got := tt.Clipboard(); got != want {
				t.Fatalf("multiline copy=%q, want %q", got, want)
			}
			captureMigrationPreview(t, tt, fmt.Sprintf("diff-selection-search-%t", search))
			// Reverse dragging has the same result.
			tt.Advance(time.Second)
			tt.Press(last.X+last.W-1, last.Y+last.H/2)
			tt.Move(first.X, first.Y+first.H/2)
			tt.Release(first.X, first.Y+first.H/2)
			tt.Key(ui.Cmd, ui.KeyC)
			if got := tt.Clipboard(); got != want {
				t.Fatalf("reverse multiline copy=%q", got)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Key(ui.Cmd, ui.KeyC)
			if got := tt.Clipboard(); got != strings.Join(a.diffLines, "\n") {
				t.Fatalf("Select All includes gutters or omits lines: %q", got)
			}
		})
	}
}

func TestDiffSelectionIncludesOffscreenLinesAndResetsForAnotherFile(t *testing.T) {
	a := diffFindFixture()
	a.diffLines = nil
	for i := range 5000 {
		a.diffLines = append(a.diffLines, fmt.Sprintf("+line %03d 中文", i))
	}
	tt := ui.NewTester(a.view, 1000, 700)
	r, ok := tt.Find(a.diffLines[0])
	if !ok {
		t.Fatal("first line missing")
	}
	tt.ClickAt(r.X+2, r.Y+r.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != strings.Join(a.diffLines, "\n") {
		t.Fatalf("offscreen Select All copied %d of %d lines", len(strings.Split(got, "\n")), len(a.diffLines))
	}
	// Search must still reveal a result near the end of a long diff.
	a.openSearch()
	a.diffQuery = "line 4999"
	tt.Frame()
	tt.Frame()
	if a.diffScroll.Y <= 0 {
		t.Fatal("find did not scroll to offscreen result")
	}
	a.showSearch = false
	tt.Frame()
	if got := tt.Clipboard(); got != strings.Join(a.diffLines, "\n") {
		t.Fatal("search changed clipboard")
	}
	// A new file must not inherit the previous file's selection.
	a.selectedFile = 1
	a.diffLines = []string{"+other file"}
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got == "+other file" {
		t.Fatal("selection leaked to another file")
	}
}

func TestDiffDragScrollKeepsOffscreenAnchor(t *testing.T) {
	a := diffFindFixture()
	a.diffLines = nil
	for i := range 80 {
		a.diffLines = append(a.diffLines, fmt.Sprintf("+  line %02d", i))
	}
	tt := ui.NewTester(a.view, 1000, 420)
	first, _ := tt.Find(a.diffLines[0])
	viewport, _ := tt.Find("Diff viewport")
	x, y := viewport.X+viewport.W-25, viewport.Y+viewport.H+12
	tt.Press(first.X, first.Y+first.H/2)
	tt.Move(x, y)
	for i := 0; i < 120 && a.diffScroll.Y < a.diffScroll.MaxY; i++ {
		tt.Advance(33 * time.Millisecond)
		tt.Frame()
	}
	tt.Release(x, y)
	tt.Key(ui.Cmd, ui.KeyC)
	if got, want := tt.Clipboard(), strings.Join(a.diffLines, "\n"); got != want {
		t.Fatalf("drag scrolling lost selection: copied %d lines, want %d; scroll %.0f/%.0f", len(strings.Split(got, "\n")), len(a.diffLines), a.diffScroll.Y, a.diffScroll.MaxY)
	}
}

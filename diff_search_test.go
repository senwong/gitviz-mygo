package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func diffFindFixture() *App {
	a := searchGraphFixture()
	a.prefs.InlineDetails = true
	a.selected = 0
	a.listSelected = 0
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.detailVisible = true
	a.largeDiffVisible = true
	a.files = []FileChange{{Path: "src/message.ts", Status: "M"}, {Path: "src/other.ts", Status: "M"}}
	a.selectedFile = 0
	a.rebuildFileRows()
	a.diffLines = []string{"diff --git a/src/message.ts b/src/message.ts", "--- a/src/message.ts", "+++ b/src/message.ts", "@@ -1,2 +1,2 @@", "-const old = 'needle';", "+const message = '中文文案 NEEDLE needle [literal]';", " unchanged"}
	return a
}

func TestDiffFindHighlightsOccurrencesAndPreservesGraphSearch(t *testing.T) {
	a := diffFindFixture()
	a.query = "feature"
	a.rebuildVisible()
	before := append([]int(nil), a.searchMatches...)
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("needle")
	if !a.searchDiff || !tt.Focused("Search file diff") || !tt.HasText("1 / 3") {
		t.Fatalf("diff search missing: %v", tt.Texts())
	}
	if a.query != "feature" || !reflect.DeepEqual(before, a.searchMatches) {
		t.Fatal("file search modified graph search")
	}
	matches := a.diffMatches
	if len(matches) != 3 || matches[1].Line != matches[2].Line {
		t.Fatalf("same-line occurrences not counted: %+v", matches)
	}
	lines := a.displayedDiffLines()
	spans := a.diffSearchSpans(matches[1].Line, lines[matches[1].Line])
	var plain strings.Builder
	highlighted := 0
	for _, span := range spans {
		plain.WriteString(span.Text)
		if span.Background != ui.Transparent {
			highlighted++
		}
	}
	if plain.String() != lines[matches[1].Line] || highlighted != 2 {
		t.Fatalf("highlight altered diff or missed repeated matches: %+v", spans)
	}
	tt.Key(ui.Cmd, ui.KeyG)
	if a.diffMatchCursor != 1 || !tt.HasText("2 / 3") {
		t.Fatal("Cmd+G did not advance diff occurrence")
	}
	tt.Key(0, ui.KeyEnter)
	if a.diffMatchCursor != 2 {
		t.Fatal("Enter did not advance")
	}
	tt.Key(0, ui.KeyEnter)
	if a.diffMatchCursor != 0 {
		t.Fatal("next did not wrap")
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyG)
	if a.diffMatchCursor != 2 {
		t.Fatal("previous did not wrap")
	}
	captureMigrationPreview(t, tt, "diff-find-occurrences")
	tt.Key(0, ui.KeyEscape)
	if a.showSearch || !a.largeDiffVisible || len(a.diffMatches) != 0 {
		t.Fatal("Escape did not close only diff search")
	}
}

func TestDiffFindIsLiteralUnicodeAndTracksFileAndHeaders(t *testing.T) {
	a := diffFindFixture()
	a.openSearch()
	for _, tc := range []struct {
		query string
		count int
	}{{"文案", 1}, {"[literal]", 1}, {".*", 0}, {"NEEDLE", 3}, {" ", -1}} {
		a.diffQuery = tc.query
		a.rebuildDiffSearch()
		if tc.count >= 0 && len(a.diffMatches) != tc.count {
			t.Fatalf("%q matched %d, want %d", tc.query, len(a.diffMatches), tc.count)
		}
		if tc.count < 0 && len(a.diffMatches) == 0 {
			t.Fatal("whitespace query was trimmed away")
		}
	}
	a.diffQuery = "message.ts"
	a.rebuildDiffSearch()
	if len(a.diffMatches) != 0 {
		t.Fatal("hidden metadata was searched")
	}
	a.diffMetadata = true
	a.rebuildDiffSearch()
	if len(a.diffMatches) != 4 {
		t.Fatalf("visible metadata missing: %v", a.diffMatches)
	}
	a.diffQuery = "needle"
	a.diffMetadata = false
	a.rebuildDiffSearch()
	a.moveFindMatch(1)
	a.selectedFile = 1
	a.diffLoading = true
	a.diffLines = nil
	a.rebuildDiffSearch()
	if len(a.diffMatches) != 0 {
		t.Fatal("old file matches retained while loading")
	}
	a.diffLoading = false
	a.diffLines = []string{"@@ -1 +1 @@", "+different NEEDLE"}
	a.rebuildDiffSearch()
	if len(a.diffMatches) != 1 || a.diffMatchCursor != 0 || a.diffMatches[0].Line != 1 {
		t.Fatal("new file did not rebuild matches")
	}
}

func TestDiffFindJumpsToOffscreenMatchesAndKeepsInputWidth(t *testing.T) {
	for _, size := range [][2]int{{1380, 900}, {1000, 620}} {
		a := diffFindFixture()
		a.diffLines = []string{"@@ -1,250 +1,250 @@"}
		for i := 0; i < 250; i++ {
			a.diffLines = append(a.diffLines, fmt.Sprintf(" line %d", i))
		}
		a.diffLines = append(a.diffLines, "+offscreen 文案 needle")
		tt := ui.NewTester(a.view, size[0], size[1])
		tt.Key(ui.Cmd, ui.KeyF)
		initial, _ := tt.Find("Search file diff")
		tt.Type("needle")
		tt.Frame()
		if !tt.HasText("1 / 1") {
			t.Fatal("missing offscreen match count")
		}
		found, ok := tt.Find("+offscreen 文案 needle")
		viewport, _ := tt.Find("Diff viewport")
		if !ok || found.Y < viewport.Y || found.Y+found.H > viewport.Y+viewport.H+1 {
			t.Fatalf("offscreen match was not revealed: match=%+v viewport=%+v", found, viewport)
		}
		after, _ := tt.Find("Search file diff")
		if initial != after {
			t.Fatal("typing resized diff search input")
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type("not present")
		tt.Frame()
		if !tt.HasText("0 matches") {
			t.Fatal("missing no-results state")
		}
		after, _ = tt.Find("Search file diff")
		if initial != after {
			t.Fatal("no results resized diff search input")
		}
		captureMigrationPreview(t, tt, fmt.Sprintf("diff-find-empty-%d", size[0]))
	}
}

func TestCmdWReturnsFromDiffToGraphWithoutClosingWindow(t *testing.T) {
	for _, inline := range []bool{true, false} {
		a := diffFindFixture()
		a.prefs.InlineDetails = inline
		a.largeDiffVisible = inline
		tt := ui.NewTester(a.view, 1380, 900)
		tt.Key(ui.Cmd, ui.KeyF)
		tt.Type("needle")
		tt.Key(ui.Cmd, ui.KeyW)
		tt.Frame()
		if a.isDiffView() || a.searchDiff || a.quitting || a.revision != "merge-sha" || a.selectedFile != 0 {
			t.Fatal("Cmd+W did not return to graph preserving selection")
		}
		if !tt.HasText("feature work") {
			t.Fatal("graph not restored")
		}
	}
}

func TestDiffFindHighlightContrastAcrossThemes(t *testing.T) {
	for _, name := range themeNames() {
		a := diffFindFixture()
		a.prefs.Theme = name
		a.setTheme()
		a.openSearch()
		a.diffQuery = "needle"
		a.rebuildDiffSearch()
		for _, match := range a.diffMatches {
			for _, span := range a.diffSearchSpans(match.Line, a.displayedDiffLines()[match.Line]) {
				if span.Background != ui.Transparent && contrastRatio(span.Color, span.Background) < 4.5 {
					t.Fatal("unreadable match in", name)
				}
			}
		}
	}
}

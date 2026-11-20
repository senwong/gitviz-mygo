package main

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRepositorySearchMatchesDisplayedNameOnly(t *testing.T) {
	a := uiFixture()
	a.repos = []string{"/workspace/parent-common/api", "/workspace/parent-common/ui", "/workspace/hidden-name/on-disk"}
	a.repoNames = map[string]string{a.repos[0]: "Backend", a.repos[1]: "Frontend", a.repos[2]: "产品服务"}
	for _, tc := range []struct {
		query string
		want  []int
	}{
		{"parent-common", nil}, {"workspace", nil}, {"hidden-name", nil}, {"api", nil}, {"on-disk", nil},
		{" BACK ", []int{0}}, {"front", []int{1}}, {"产品", []int{2}}, {"", []int{0, 1, 2}},
	} {
		a.repoQuery = tc.query
		if got := a.filteredRepositories(); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("query %q: got=%v want=%v", tc.query, got, tc.want)
		}
	}
}

func TestLargeDiffReturnsToGraphByButtonAndEscape(t *testing.T) {
	a := uiFixture()
	a.prefs.InlineDetails = true
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.detailVisible = true
	a.selected = 0
	a.listSelected = 0
	a.files = []FileChange{{Path: "file.txt", Status: "M"}}
	a.selectedFile = 0
	a.rebuildFileRows()
	a.diffLines = []string{"@@ -1 +1 @@", "-old", "+new"}
	a.largeDiffVisible = true
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.Click("Back to Git Graph"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.largeDiffVisible || a.selectedFile != 0 || a.revision != a.selectedCommit.SHA {
		t.Fatal("back button lost graph context")
	}
	if _, ok := tt.Find("Inline commit details"); !ok {
		t.Fatal("back button did not show graph")
	}
	a.largeDiffVisible = true
	tt.Frame()
	captureMigrationPreview(t, tt, "large-diff-back-navigation")
	if err := tt.RightClick("file.txt"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if !a.largeDiffVisible || len(tt.Menu()) > 0 {
		t.Fatal("Escape should close context menu before leaving diff")
	}
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("new")
	tt.Key(0, ui.KeyEscape)
	if a.showSearch || !a.largeDiffVisible {
		t.Fatal("Escape should close search before leaving diff")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.largeDiffVisible || a.selectedFile != 0 {
		t.Fatal("Escape did not return to graph with file selection preserved")
	}
}

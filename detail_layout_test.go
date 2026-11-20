package main

import (
	"github.com/egoist/mygo/ui"
	"reflect"
	"strings"
	"testing"
)

func TestCompactDiffKeepsChangesAndBinaryMetadata(t *testing.T) {
	lines := []string{"diff --git a/a b/a", "index abc..def 100644", "--- a/a", "+++ b/a", "@@ -10,2 +10,2 @@", "--- removed content", "+++ added content", "diff --git a/b b/b", "new file mode 100644", "Binary files /dev/null and b/b differ"}
	want := []string{"@@ -10,2 +10,2 @@", "--- removed content", "+++ added content", "new file mode 100644", "Binary files /dev/null and b/b differ"}
	if got := compactDiffLines(lines); !reflect.DeepEqual(got, want) {
		t.Fatalf("diff content lost: %q", got)
	}
}
func TestCompactDetailsPrioritizeDiffAndExpandMessage(t *testing.T) {
	a := uiFixture()
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.detailVisible = true
	a.selectedCommit.Body = "First line https://example.test/change\n\nSecond line\n\nThird line\nFourth line"
	a.files = []FileChange{{Path: "static-page/co/privacy-policy.html", Status: "M"}}
	a.selectedFile = 0
	a.rebuildFileRows()
	a.diffLines = []string{"diff --git a/file b/file", "index 123..456", "--- a/file", "+++ b/file", "@@ -1 +1 @@", "-old content", "+new content"}
	tt := ui.NewTester(a.view, 1380, 900)
	if tt.HasText("diff --git a/file b/file") || !tt.HasText("+new content") {
		t.Fatal("diff headers were not collapsed")
	}
	if !tt.HasText("privacy-policy.html") {
		t.Fatal("file basename missing")
	}
	if !tt.HasText("1 changed file") {
		t.Fatal("combined file toolbar missing")
	}
	title, _ := tt.Find("Close details")
	code, ok := tt.Find("+new content")
	if !ok || code.Y-title.Y > 200 {
		t.Fatalf("metadata still crowds out diff: title=%+v code=%+v", title, code)
	}
	captureMigrationPreview(t, tt, "compact-detail")
	if err := tt.Click("Toggle commit message"); err != nil {
		t.Fatal(err)
	}
	if !a.messageExpanded || !tt.HasText("Show less") {
		t.Fatal("message did not expand")
	}
	if !strings.Contains(strings.Join(tt.Texts(), " "), "Fourth line") {
		t.Fatal("expanded message lost body")
	}
	tt.Click("Toggle commit message")
	if err := tt.Click("Show headers"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("diff --git a/file b/file") {
		t.Fatal("raw headers unavailable")
	}
	if err := tt.Click("Hide headers"); err != nil {
		t.Fatal(err)
	}
	tt.SetSize(1000, 620)
	captureMigrationPreview(t, tt, "compact-detail-minimum")
	if rect, ok := tt.Find("Tree / List"); !ok || rect.X+rect.W > 1000 {
		t.Fatal("file toolbar control inaccessible")
	}
	a.clearDetail()
	if a.messageExpanded {
		t.Fatal("expanded state leaked into next commit")
	}
}

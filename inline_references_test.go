package main

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
)

func TestAllReferencesRenderInlineAndKeepTheirMenus(t *testing.T) {
	a := uiFixture()
	a.snapshot.Commits[0].Refs = []string{"origin/release", "tag:publish-cn-20261019", "tag:publish-jp", "feature", "tag:v5"}
	a.snapshot.Branches = []Branch{{Name: "origin/release", Full: "refs/remotes/origin/release", Remote: true}, {Name: "feature", Full: "refs/heads/feature"}}
	a.snapshot.Tags = map[string]Tag{"publish-jp": {Name: "publish-jp", Annotated: true}, "v5": {Name: "v5"}}
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git"}
	tt := newMenuTester(a, 1380, 900)
	var previous ui.Rect
	for i, label := range []string{"origin/release", "publish-cn-20261019", "publish-jp", "feature", "v5"} {
		rect, ok := tt.Find(label)
		if !ok {
			t.Fatalf("reference %d %s is hidden", i, label)
		}
		if i > 0 && (rect.X <= previous.X || rect.Y != previous.Y) {
			t.Fatalf("references are not inline: %+v %+v", previous, rect)
		}
		previous = rect
	}
	if tt.HasText("+3") {
		t.Fatal("count-based reference overflow remains")
	}
	captureMigrationPreview(t, tt, "all-references-inline")
	for _, test := range []struct{ label, want string }{{"publish-jp", "View Details"}, {"feature", "Rename Branch…"}, {"v5", "Delete Tag…"}} {
		if err := tt.RightClick(test.label); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.Join(tt.Menu(), "|"), test.want) {
			t.Fatalf("wrong menu for %s: %v", test.label, tt.Menu())
		}
		tt.CloseMenu()
	}
	tt.SetSize(1000, 620)
	captureMigrationPreview(t, tt, "all-references-minimum")
	date, ok := tt.Find(a.dateText(a.snapshot.Commits[0]))
	if !ok || date.X+date.W > 1000 {
		t.Fatalf("references pushed date column outside the viewport: %+v", date)
	}
	if !tt.HasText("A test commit") {
		t.Fatal("commit subject was removed")
	}
}

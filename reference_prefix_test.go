package main

import (
	"strings"
	"testing"
)

func TestCombinedReferencesUseGitGraphRemoteSuffixAndKeepBothMenus(t *testing.T) {
	a := uiFixture()
	first := a.snapshot.Commits[0]
	first.Refs = []string{"origin/release"}
	second := first
	second.SHA = "local-sha"
	second.Subject = "local branch commit"
	second.Refs = []string{"dev-10.24.4\x1forigin"}
	a.snapshot.Commits = []Commit{first, second}
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git"}
	a.snapshot.Branches = []Branch{{Name: "origin/release", Full: "refs/remotes/origin/release", Remote: true}, {Name: "dev-10.24.4", Full: "refs/heads/dev-10.24.4", Current: true}, {Name: "origin/dev-10.24.4", Full: "refs/remotes/origin/dev-10.24.4", Remote: true}}
	a.rebuildVisible()
	tt := newMenuTester(a, 1380, 900)
	if !tt.HasText("origin/release") {
		t.Fatal("remote-only label changed")
	}
	prefix, ok := tt.Find("origin")
	if !ok {
		t.Fatal("combined remote suffix missing")
	}
	branch, ok := tt.Find("dev-10.24.4")
	if !ok || prefix.X <= branch.X {
		t.Fatal("remote suffix is not after local branch name")
	}
	if _, ok := tt.Find("origin/dev-10.24.4"); !ok {
		t.Fatal("combined label has no accessible canonical name")
	}
	for _, item := range []struct{ target, want, absent string }{{"origin", "Delete Remote Branch…", "Rename Branch…"}, {"dev-10.24.4", "Rename Branch…", "Delete Remote Branch…"}} {
		if err := tt.RightClick(item.target); err != nil {
			t.Fatal(err)
		}
		menu := strings.Join(tt.Menu(), "|")
		if !strings.Contains(menu, item.want) || strings.Contains(menu, item.absent) {
			t.Fatalf("%s routed incorrectly: %s", item.target, menu)
		}
		tt.CloseMenu()
	}
	captureMigrationPreview(t, tt, "gitgraph-reference-suffix")
	a.query = "origin/dev-10.24.4"
	a.rebuildVisible()
	if len(a.searchMatches) != 1 || a.searchMatches[0] != 1 {
		t.Fatal("search does not match displayed branch name")
	}
}
func TestReferenceDisplayNamePreservesFullReferenceSetting(t *testing.T) {
	for ref, want := range map[string]string{
		"release\x1forigin":              "origin/release",
		"feature/a\x1fupstream":          "upstream/feature/a",
		"refs/heads/feature/a\x1forigin": "refs/remotes/origin/feature/a",
		"origin/release":                 "origin/release",
		"local-only":                     "local-only",
		"tag:v1":                         "v1",
		"tag:refs/tags/v1":               "refs/tags/v1",
	} {
		if got := referenceDisplayName(ref); got != want {
			t.Fatalf("%q: got %q want %q", ref, got, want)
		}
	}
}

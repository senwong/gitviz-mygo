package main

import (
	"context"
	"github.com/egoist/mygo/ui"
	"os"
	"reflect"
	"testing"
	"time"
)

func searchGraphFixture() *App {
	a := uiFixture()
	base := a.snapshot.Commits[0]
	a.snapshot.Commits = nil
	for _, item := range []struct {
		sha, subject string
		parents      []string
	}{
		{"merge-sha", "needle merge", []string{"main-sha", "feature-sha"}},
		{"feature-sha", "feature work", []string{"base-sha"}},
		{"main-sha", "needle main", []string{"base-sha"}},
		{"base-sha", "shared base", []string{"root-sha"}},
		{"root-sha", "initial commit", nil},
	} {
		c := base
		c.SHA = item.sha
		c.Subject = item.subject
		c.Parents = item.parents
		c.Refs = nil
		a.snapshot.Commits = append(a.snapshot.Commits, c)
	}
	a.snapshot.Lanes = assignLanes(a.snapshot.Commits)
	a.rebuildVisible()
	return a
}
func nearColor(r, g, b uint8, color ui.Color) bool {
	return abs(int(r)-int(color.R)) <= 6 && abs(int(g)-int(color.G)) <= 6 && abs(int(b)-int(color.B)) <= 6
}
func TestSearchPreservesMergeGraphAndUnmatchedRows(t *testing.T) {
	a := searchGraphFixture()
	tt := ui.NewTester(a.view, 1380, 900)
	before := tt.Image()
	rects := map[string]ui.Rect{}
	for _, commit := range a.snapshot.Commits {
		r, ok := tt.Find(commit.Subject)
		if !ok {
			t.Fatal("missing initial row")
		}
		rects[commit.Subject] = r
	}
	visible := append([]int(nil), a.visible...)
	captureMigrationPreview(t, tt, "graph-before-search")
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("needle")
	if !reflect.DeepEqual(a.visible, visible) || !reflect.DeepEqual(a.searchMatches, []int{0, 2}) {
		t.Fatalf("search removed intermediate commits: visible=%v matches=%v", a.visible, a.searchMatches)
	}
	for _, commit := range a.snapshot.Commits {
		r, ok := tt.Find(commit.Subject)
		if !ok || r != rects[commit.Subject] {
			t.Fatalf("search moved or removed %s", commit.Subject)
		}
	}
	after := tt.Image()
	pixels := 0
	for y := int(rects["needle merge"].Y); y < int(rects["initial commit"].Y+28); y++ {
		for x := 16; x < int(max(2, a.snapshot.Lanes)*18+32); x++ {
			pixel := before.RGBAAt(x, y)
			for _, color := range a.colors() {
				if nearColor(pixel.R, pixel.G, pixel.B, color) {
					next := after.RGBAAt(x, y)
					if !nearColor(next.R, next.G, next.B, color) {
						t.Fatalf("search erased graph connection at %d,%d", x, y)
					}
					pixels++
					break
				}
			}
		}
	}
	if pixels < 100 {
		t.Fatalf("graph regression check sampled too few line pixels: %d", pixels)
	}
	if !tt.HasText("1 / 2") {
		t.Fatal("match count includes nonmatches")
	}
	captureMigrationPreview(t, tt, "graph-during-search")
	tt.Type("-missing")
	if !tt.HasText("0 matches") || len(a.visible) != 5 {
		t.Fatal("no-match search erased the graph")
	}
	captureMigrationPreview(t, tt, "graph-no-matches")
	tt.Key(0, ui.KeyEscape)
	if len(a.searchMatches) != 0 || len(a.visible) != 5 {
		t.Fatal("closing search left stale results")
	}
}
func TestSearchNavigationVisitsOnlyMatchesAndKeepsPositionOnReload(t *testing.T) {
	a := searchGraphFixture()
	a.post = func(func()) {}
	defer func() {
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
	}()
	a.showSearch = true
	a.query = "needle"
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Key(ui.Cmd, ui.KeyG)
	if a.revision != "main-sha" || a.matchCursor != 1 {
		t.Fatalf("next visited a nonmatch: %s %d", a.revision, a.matchCursor)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyG)
	if a.revision != "merge-sha" {
		t.Fatal("previous did not return to first match")
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyG)
	if a.revision != "main-sha" {
		t.Fatal("previous did not wrap")
	}
	a.snapshot.Commits = append([]Commit{{SHA: "new-head", Subject: "other work"}}, a.snapshot.Commits...)
	a.rebuildVisible()
	if a.searchCurrent != "main-sha" || a.searchMatches[a.matchCursor] != 3 {
		t.Fatal("reload lost the current match")
	}
	a.query = "not found"
	a.rebuildVisible()
	before := a.revision
	a.moveSearchMatch(1)
	if a.revision != before {
		t.Fatal("no-match navigation changed selection")
	}
}
func TestSearchGraphWorkspacePreview(t *testing.T) {
	repo := os.Getenv("GITVIZ_SEARCH_GRAPH_REPO")
	if repo == "" {
		t.Skip("set GITVIZ_SEARCH_GRAPH_REPO for real history screenshots")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	a := uiFixture()
	a.repos = []string{repo}
	a.active = 0
	a.snapshot = Snapshot{}
	var err error
	a.snapshot, err = loadSnapshotConfig(ctx, repo, 500, a.config, "")
	if err != nil {
		t.Fatal(err)
	}
	a.rebuildVisible()
	count := len(a.visible)
	tt := ui.NewTester(a.view, 1380, 900)
	captureMigrationPreview(t, tt, "umu-node-before")
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("re")
	if len(a.visible) != count || len(a.searchMatches) == 0 {
		t.Fatalf("real history changed size: %d to %d, %d matches", count, len(a.visible), len(a.searchMatches))
	}
	captureMigrationPreview(t, tt, "umu-node-search-re")
	t.Logf("%d commits retained, %d matches, %d lanes", count, len(a.searchMatches), a.snapshot.Lanes)
}

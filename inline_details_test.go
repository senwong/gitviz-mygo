package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestInlineDetailsAndLargeDiffKeepCommitAndFileContext(t *testing.T) {
	repo := testRepo(t)
	testCommit(t, repo, "file.txt", "first\n", "base commit")
	sha := testCommit(t, repo, "file.txt", "first\nsecond\n", "latest commit\n\nFull commit message")
	h := newActionHarness(t, repo)
	a := h.a
	a.prefs.InlineDetails = true
	a.openCommit(0)
	h.drain()
	if a.selectedFile != -1 || a.fileRowSelected != -1 || a.diffLoading || len(a.diffLines) > 0 {
		t.Fatal("opening inline commit details selected or loaded a file before it was clicked")
	}
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Frame()
	commit, _ := tt.Find("latest commit")
	detail, ok := tt.Find("Inline commit details")
	next, _ := tt.Find("base commit")
	if !ok || detail.Y <= commit.Y || next.Y < detail.Y+detail.H {
		t.Fatalf("details not between selected commit and next row: commit=%+v detail=%+v next=%+v", commit, detail, next)
	}
	if _, ok := tt.Find("Diff viewport"); ok {
		t.Fatal("commit selection opened diff before file was clicked")
	}
	pixel := tt.Image().RGBAAt(22, int(detail.Y+40))
	color := a.colors()[a.snapshot.Commits[0].Lane]
	if !nearColor(pixel.R, pixel.G, pixel.B, color) {
		t.Fatal("graph track does not continue through inline details")
	}
	captureMigrationPreview(t, tt, "inline-commit-details")
	if err := tt.Click("file.txt"); err != nil {
		t.Fatal(err)
	}
	h.drain()
	tt.Frame()
	viewport, ok := tt.Find("Diff viewport")
	if !a.largeDiffVisible || !ok || viewport.H < 650 || !tt.HasText("+second") {
		t.Fatalf("file did not open large diff: %+v", viewport)
	}
	captureMigrationPreview(t, tt, "large-file-diff")
	tt.SetSize(1000, 620)
	viewport, ok = tt.Find("Diff viewport")
	if !ok || viewport.H < 400 {
		t.Fatalf("diff too short at minimum width: %+v", viewport)
	}
	captureMigrationPreview(t, tt, "large-file-diff-minimum")
	if err := tt.Click("Back to Git Graph"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.largeDiffVisible || a.revision != sha || a.selectedFile != 0 {
		t.Fatal("back lost commit or file context")
	}
	if _, ok := tt.Find("Inline commit details"); !ok {
		t.Fatal("back did not restore inline details")
	}
	if err := tt.Click("Close details"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.detailVisible {
		t.Fatal("inline details did not close")
	}
	if err := tt.Click("latest commit"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find("Inline commit details"); !ok {
		t.Fatal("same commit cannot reopen inline details")
	}
}

func TestManyReferencesWrapWithoutClippingTagsOrCoveringColumns(t *testing.T) {
	a := searchGraphFixture()
	refs := []string{"branch-first", "branch-second", "hotfix-cn-20261016\x1forigin", "tag:publish-cn-20261112", "tag:publish-com-20261116", "tag:bak_publish-cn-20261112_" + strings.Repeat("long-reference-", 12) + "visible-suffix"}
	a.snapshot.Commits[0].Refs = refs
	a.snapshot.Tags = map[string]Tag{"publish-com-20261116": {Name: "publish-com-20261116"}}
	for _, size := range [][2]int{{1380, 900}, {1000, 620}} {
		tt := newMenuTester(a, size[0], size[1])
		tt.Frame()
		tt.Frame()
		date, _ := tt.Find(a.dateText(a.snapshot.Commits[0]))
		next, _ := tt.Find(a.snapshot.Commits[1].Subject)
		first, _ := tt.Find("branch-first")
		last, ok := tt.Find(strings.TrimPrefix(refs[len(refs)-1], "tag:"))
		if !ok || last.Y <= first.Y || last.X+last.W > date.X || last.Y+last.H > next.Y {
			t.Fatalf("tag was clipped or overlapped another row: first=%+v last=%+v date=%+v next=%+v", first, last, date, next)
		}
		if err := tt.RightClick("publish-com-20261116"); err != nil {
			t.Fatal(err)
		}
		if !containsAction(tt.Menu(), "Delete Tag…") {
			t.Fatal("wrapped tag lost menu")
		}
		tt.CloseMenu()
		captureMigrationPreview(t, tt, "wrapped-references-"+map[int]string{1380: "wide", 1000: "minimum"}[size[0]])
	}
}

func containsAction(actions []string, want string) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}

func TestInlineDetailsWorkspacePreview(t *testing.T) {
	repo := os.Getenv("GITVIZ_INLINE_REPO")
	if repo == "" {
		t.Skip("set GITVIZ_INLINE_REPO for a read-only workspace preview")
	}
	h := newActionHarness(t, repo)
	a := h.a
	a.prefs.InlineDetails = true
	a.config.FileTree = true
	selected := 0
	target := os.Getenv("GITVIZ_INLINE_SHA")
	if target == "" {
		target = a.snapshot.Head
	}
	for i, commit := range a.snapshot.Commits {
		if strings.HasPrefix(commit.SHA, target) {
			selected = i
			break
		}
	}
	a.openCommit(selected)
	h.drain()
	a.list.ScrollTo(selected, ui.Start)
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Frame()
	captureMigrationPreview(t, tt, "workspace-inline-details")
	if len(a.files) > 0 {
		if err := tt.Click(filepath.Base(a.files[0].Path)); err != nil {
			t.Fatal(err)
		}
		h.drain()
		tt.Frame()
		if !a.largeDiffVisible {
			t.Fatal("workspace file did not open diff")
		}
		captureMigrationPreview(t, tt, "workspace-large-diff")
	}
}

func TestInlineDetailsLinksKeepThemeAccent(t *testing.T) {
	a := uiFixture()
	a.prefs.Theme = "One Dark Pro"
	a.setTheme()
	a.prefs.InlineDetails = true
	a.selectedCommit = &a.snapshot.Commits[0]
	a.selectedCommit.Body = "Details https://example.test/change"
	a.revision = a.selectedCommit.SHA
	a.selected = 0
	a.listSelected = 0
	a.detailVisible = true
	tt := ui.NewTester(a.view, 1380, 900)
	rect, ok := tt.Find("https://example.test/change")
	if !ok {
		t.Fatal("link missing from inline details")
	}
	frame := tt.Image()
	pixels := 0
	accent := a.uiTheme.Accent
	for y := int(rect.Y); y < int(rect.Y+rect.H); y++ {
		for x := int(rect.X); x < int(rect.X+rect.W); x++ {
			p := frame.RGBAAt(x, y)
			if abs(int(p.R)-int(accent.R)) < 24 && abs(int(p.G)-int(accent.G)) < 24 && abs(int(p.B)-int(accent.B)) < 24 {
				pixels++
			}
		}
	}
	if pixels < 5 {
		t.Fatal("inline details inherited list selection color instead of theme accent")
	}
	captureMigrationPreview(t, tt, "inline-details-link-theme")
}

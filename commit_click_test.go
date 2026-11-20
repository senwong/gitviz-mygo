package main

import (
	"context"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCommitClickReopensSelectedDetailsAndLoadsDiff(t *testing.T) {
	repo := testRepo(t)
	first := testCommit(t, repo, "first.txt", "first content\n", "first commit")
	second := testCommit(t, repo, "second.txt", "second content\n", "second commit")
	h := newActionHarness(t, repo)
	a := h.a
	a.config.ShowDetail = false
	a.detailVisible = false
	a.selectCommit(0)
	h.drain()
	if a.detailVisible {
		t.Fatal("programmatic selection ignored saved visibility")
	}
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.Click("second commit"); err != nil {
		t.Fatal(err)
	}
	h.drain()
	tt.Frame()
	if !a.detailVisible || !a.config.ShowDetail || a.revision != second {
		t.Fatal("single click did not open selected commit details")
	}
	if !strings.Contains(a.diffText, "+second content") {
		t.Fatalf("diff not loaded: %q", a.diffText)
	}
	captureMigrationPreview(t, tt, "single-click-details")
	generation := a.detailGen
	if err := tt.Click("Close details"); err != nil {
		t.Fatal(err)
	}
	if a.detailVisible {
		t.Fatal("close did not hide details")
	}
	if err := tt.Click("second commit"); err != nil {
		t.Fatal(err)
	}
	if !a.detailVisible {
		t.Fatal("clicking selected commit did not reopen details")
	}
	if a.detailGen != generation {
		t.Fatal("reopening the same commit unnecessarily reloaded details")
	}
	if err := tt.Click("first commit"); err != nil {
		t.Fatal(err)
	}
	h.drain()
	tt.Frame()
	if a.revision != first || !strings.Contains(a.diffText, "+first content") {
		t.Fatal("clicking another commit did not switch details and diff")
	}
	if a.listSelected != 1 {
		t.Fatalf("row selection was lost: %d", a.listSelected)
	}
	if _, err := gitCommand(context.Background(), repo, "diff", "--exit-code"); err != nil {
		t.Fatal("viewing details modified repository")
	}
}

func TestCommitKeyboardSelectionAndEnterOpenDetails(t *testing.T) {
	a := uiFixture()
	a.post = func(func()) {}
	other := a.snapshot.Commits[0]
	other.SHA = "second-sha"
	other.Subject = "another commit"
	a.snapshot.Commits = append(a.snapshot.Commits, other)
	a.rebuildVisible()
	defer func() {
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
	}()
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.Click("A test commit"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyDown)
	if a.revision != "second-sha" {
		t.Fatal("click handler broke keyboard navigation")
	}
	tt.Click("Close details")
	tt.Key(0, ui.KeyUp)
	if !a.detailVisible || a.revision != "1234567890abcdef" {
		t.Fatal("keyboard selection did not open details")
	}
	if err := tt.Click("Close details"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if !a.detailVisible {
		t.Fatal("Enter did not reopen the selected commit")
	}

}

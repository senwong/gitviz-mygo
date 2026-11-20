package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestCommitLogIncludesDistinctAuthorAndCommitter(t *testing.T) {
	repo := testRepo(t)
	testCommit(t, repo, "file.txt", "content\n", "subject\n\nbody")
	testGit(t, repo, "commit", "--amend", "--no-edit", "--author", "Original Author <author@example.test>", "--date", "2020-01-02T03:04:05Z")
	commits, _, err := readLog(context.Background(), repo, 10, true, "")
	if err != nil {
		t.Fatal(err)
	}
	c := commits[0]
	if c.Author != "Original Author" || c.Email != "author@example.test" || c.Committer != "Gitviz Test" || c.CommitterEmail != "gitviz@example.test" {
		t.Fatalf("incorrect metadata: %+v", c)
	}
	if c.Time.Equal(c.CommitTime) || c.Time.IsZero() || c.CommitTime.IsZero() {
		t.Fatal("author and committer dates were lost")
	}
	if c.Subject != "subject" || c.Body != "body" {
		t.Fatal("adding committer fields corrupted commit message")
	}
}

func TestFileStatisticsIncludeRootRenameBinaryAndUnusualPaths(t *testing.T) {
	repo := testRepo(t)
	path := "folder/a\tfile.txt"
	first := testCommit(t, repo, path, "one\ntwo\nthree\n", "first")
	files, err := changedFiles(context.Background(), repo, first)
	if err != nil || len(files) != 1 || !files[0].StatsKnown || files[0].Additions != 3 || files[0].Deletions != 0 {
		t.Fatalf("root stats: %+v %v", files, err)
	}
	if err := os.WriteFile(filepath.Join(repo, path), []byte("one\nchanged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "binary.dat"), []byte{0, 1, 2, 3}, 0600); err != nil {
		t.Fatal(err)
	}
	testGit(t, repo, "add", "-A")
	testGit(t, repo, "commit", "-qm", "modify")
	files, err = changedFiles(context.Background(), repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	foundText, foundBinary := false, false
	for _, f := range files {
		if f.Path == path {
			foundText = f.StatsKnown && f.Additions == 1 && f.Deletions == 2 && !f.Binary
		}
		if f.Path == "binary.dat" {
			foundBinary = f.StatsKnown && f.Binary
		}
	}
	if !foundText || !foundBinary {
		t.Fatalf("incorrect text/binary stats: %+v", files)
	}
	testGit(t, repo, "config", "diff.renames", "true")
	target := "folder/renamed\tfile.txt"
	testGit(t, repo, "mv", path, target)
	testGit(t, repo, "commit", "-qm", "rename")
	files, err = changedFiles(context.Background(), repo, "HEAD")
	if err != nil || len(files) != 1 || files[0].Path != target || files[0].OldPath != path || !files[0].StatsKnown || files[0].Additions != 0 || files[0].Deletions != 0 {
		t.Fatalf("rename stats: %+v %v", files, err)
	}
}

func TestInlineSummaryShowsFullMetadataLinksAndFileStatistics(t *testing.T) {
	a := uiFixture()
	a.prefs.InlineDetails = true
	c := &a.snapshot.Commits[0]
	c.SHA = strings.Repeat("a", 40)
	c.Parents = []string{strings.Repeat("b", 40)}
	c.Committer = "Committer Name"
	c.CommitterEmail = "committer@example.test"
	c.CommitTime = c.Time.Add(time.Hour)
	c.Body = "Full body https://example.test/ticket\n\nCo-Authored-By: Contributor <contributor@example.test>"
	a.selectedCommit = c
	a.revision = c.SHA
	a.selected = 0
	a.listSelected = 0
	a.detailVisible = true
	a.files = []FileChange{{Path: "package.json", Status: "M", StatsKnown: true, Additions: 2, Deletions: 2}}
	a.selectedFile = -1
	a.rebuildFileRows()
	tt := ui.NewTester(a.view, 1380, 900)
	for _, text := range []string{c.SHA, c.Parents[0], c.Email, c.CommitterEmail, "(+2 | -2)", "Author Date:", "Committer Date:"} {
		if !strings.Contains(strings.Join(tt.Texts(), " "), text) {
			t.Fatalf("summary missing %q", text)
		}
	}
	if err := tt.Click(c.Email); err != nil {
		t.Fatal(err)
	}
	if urls := tt.OpenedURLs(); len(urls) != 1 || urls[0] != "mailto:"+c.Email {
		t.Fatalf("email link did not open mailto: %v", urls)
	}
	captureMigrationPreview(t, tt, "gitgraph-style-commit-summary")
}

func TestParentLinkNavigatesWithoutCheckingOut(t *testing.T) {
	repo := testRepo(t)
	parent := testCommit(t, repo, "file.txt", "one\n", "parent")
	child := testCommit(t, repo, "file.txt", "one\ntwo\n", "child")
	for _, unloaded := range []bool{false, true} {
		h := newActionHarness(t, repo)
		a := h.a
		a.prefs.InlineDetails = true
		if unloaded {
			a.snapshot.Commits = a.snapshot.Commits[:1]
			a.rebuildVisible()
		}
		a.openCommit(0)
		h.drain()
		tt := ui.NewTester(a.view, 1380, 900)
		if err := tt.Click(parent); err != nil {
			t.Fatal(err)
		}
		h.drain()
		tt.Frame()
		if a.revision != parent || a.snapshot.Head != child {
			t.Fatalf("parent link changed checkout or failed: revision=%s head=%s", a.revision, a.snapshot.Head)
		}
		if actual := strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD")); actual != child {
			t.Fatal("parent navigation checked out a commit")
		}
	}
}

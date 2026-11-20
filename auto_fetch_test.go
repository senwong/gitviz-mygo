package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSwitchRepositoryFetchesAfterShowingLocalHistory(t *testing.T) {
	remote := testRepo(t)
	base := testCommit(t, remote, "file.txt", "base\n", "base")
	local := filepath.Join(t.TempDir(), "clone")
	testGit(t, remote, "clone", "-q", remote, local)
	h := newActionHarness(t, local)
	a := h.a
	a.prefs.AutoFetchOnSwitch = true
	next := testCommit(t, remote, "remote.txt", "new remote content\n", "remote update")
	testGit(t, remote, "tag", "publish-cn-20261112")
	if err := os.WriteFile(filepath.Join(local, "file.txt"), []byte("uncommitted local content\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.active = -1
	a.switchRepo(0)
	select {
	case fn := <-h.queue:
		fn()
	case <-time.After(15 * time.Second):
		t.Fatal("local history did not load")
	}
	if a.loading || a.snapshot.Head != base || a.busy[local] != "Fetching remotes…" {
		t.Fatalf("local history was not available before fetch: loading=%v head=%s busy=%s", a.loading, a.snapshot.Head, a.busy[local])
	}
	h.drain()
	if a.snapshot.Tracking.Behind != 1 || a.snapshot.Tags["publish-cn-20261112"].SHA != next {
		t.Fatalf("remote status and tags did not refresh: tracking=%+v tags=%v", a.snapshot.Tracking, a.snapshot.Tags)
	}
	if got := strings.TrimSpace(testGit(t, local, "rev-parse", "HEAD")); got != base {
		t.Fatal("fetch changed local HEAD")
	}
	if content, err := os.ReadFile(filepath.Join(local, "file.txt")); err != nil || string(content) != "uncommitted local content\n" {
		t.Fatal("fetch changed working tree")
	}
	generation := a.loadGen
	a.switchRepo(a.active)
	if a.busy[local] != "" || a.loadGen != generation {
		t.Fatal("reselecting current repository triggered another fetch")
	}
}

func TestAutomaticFetchFailureKeepsLocalHistoryAndDoesNotOpenModal(t *testing.T) {
	repo := testRepo(t)
	sha := testCommit(t, repo, "file.txt", "base\n", "base")
	testGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	h := newActionHarness(t, repo)
	a := h.a
	a.autoFetchOnSwitch()
	if a.busy[repo] != "" {
		t.Fatal("disabled automatic fetch ran")
	}
	a.prefs.AutoFetchOnSwitch = true
	a.autoFetchOnSwitch()
	h.drain()
	if a.errorText != "" || a.autoFetchErrors[repo] == "" || a.snapshot.Head != sha {
		t.Fatal("fetch failure lost history or opened error modal")
	}
	tt := ui.NewTester(a.view, 1380, 900)
	if !tt.HasText("Fetch failed · Showing local history") {
		t.Fatal("automatic fetch failure is invisible")
	}
	captureMigrationPreview(t, tt, "auto-fetch-failure")
}

func TestAutomaticFetchCompletionDoesNotReloadAnotherRepository(t *testing.T) {
	remote := testRepo(t)
	testCommit(t, remote, "file.txt", "base\n", "base")
	local := filepath.Join(t.TempDir(), "clone")
	testGit(t, remote, "clone", "-q", remote, local)
	other := testRepo(t)
	otherSHA := testCommit(t, other, "other.txt", "other\n", "other repository")
	h := newActionHarness(t, local)
	a := h.a
	a.prefs.AutoFetchOnSwitch = true
	a.autoFetchOnSwitch()
	var complete func()
	select {
	case complete = <-h.queue:
	case <-time.After(15 * time.Second):
		t.Fatal("fetch did not complete")
	}
	a.prefs.AutoFetchOnSwitch = false
	a.repos = append(a.repos, other)
	a.switchRepo(1)
	generation := a.loadGen
	complete()
	if a.repo() != other || a.loadGen != generation || a.busy[local] != "" {
		t.Fatal("old fetch changed the active repository")
	}
	h.drain()
	if a.snapshot.Head != otherSHA {
		t.Fatal("old fetched snapshot overwrote new repository")
	}
}

func TestFetchScrollsToTopOnlyAfterSuccessfulAutomaticRefresh(t *testing.T) {
	remote := testRepo(t)
	tip := testCommit(t, remote, "file.txt", "base\n", "base")
	tree := strings.TrimSpace(testGit(t, remote, "rev-parse", "HEAD^{tree}"))
	for i := 0; i < 40; i++ {
		tip = strings.TrimSpace(testGit(t, remote, "commit-tree", tree, "-p", tip, "-m", fmt.Sprintf("history %d", i)))
	}
	testGit(t, remote, "update-ref", "refs/heads/main", tip)
	for _, tc := range []struct {
		name               string
		automatic, failure bool
	}{
		{"automatic success", true, false}, {"manual success", false, false}, {"automatic failure", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := filepath.Join(t.TempDir(), "clone")
			testGit(t, remote, "clone", "-q", remote, local)
			h := newActionHarness(t, local)
			a := h.a
			a.detailVisible = false
			tt := ui.NewTester(a.view, 1000, 620)
			a.list.ScrollToEnd()
			tt.Frame()
			if first, _ := a.list.Visible(); first == 0 {
				t.Fatal("test did not scroll away from top")
			}
			next := strings.TrimSpace(testGit(t, remote, "commit-tree", tree, "-p", tip, "-m", "new remote tip "+tc.name))
			testGit(t, remote, "update-ref", "refs/heads/main", next)
			if tc.failure {
				testGit(t, local, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
			}
			a.fetchRemoteUpdates(tc.automatic, a.config)
			h.drain()
			tt.Frame()
			first, _ := a.list.Visible()
			if tc.automatic && !tc.failure {
				if first != 0 {
					t.Fatalf("automatic fetch left graph at row %d", first)
				}
				row, ok := tt.Find("new remote tip " + tc.name)
				if !ok || row.Y < 0 || row.Y > 200 {
					t.Fatalf("newest fetched commit is not at top: %+v", row)
				}
				captureMigrationPreview(t, tt, "automatic-fetch-scroll-top")
			} else if first == 0 {
				t.Fatal("manual fetch or failed automatic fetch unexpectedly moved to top")
			}
		})
	}
}

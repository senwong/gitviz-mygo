package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestRemoteFreshnessChecksBranchesAndAnnotatedTagsWithoutFetching(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	remote := testRepo(t)
	base := testCommit(t, remote, "base.txt", "base", "base")
	local := filepath.Join(t.TempDir(), "clone")
	testGit(t, remote, "clone", "-q", remote, local)
	next := testCommit(t, remote, "next.txt", "next", "next")
	testGit(t, remote, "tag", "-a", "publish", "-m", "release")
	h := newActionHarness(t, local)
	h.a.prefs.AutoFetchOnSwitch = false
	h.a.active = -1
	h.a.switchRepo(0)
	h.drain()
	s := h.a.remoteState()
	if s.Branches != 1 || s.Tags != 1 || !s.fresh(time.Now()) {
		t.Fatalf("unexpected check: %+v", s)
	}
	if got := strings.TrimSpace(testGit(t, local, "rev-parse", "origin/main")); got != base {
		t.Fatal("check downloaded remote branch")
	}
	if tags := strings.TrimSpace(testGit(t, local, "tag")); tags != "" {
		t.Fatal("check downloaded tags")
	}
	tt := ui.NewTester(h.a.view, 1000, 700)
	if !tt.HasText("Fetch needed · 1 branches · 1 tags") {
		t.Fatal("missing visible update counts")
	}
	captureMigrationPreview(t, tt, "remote-fetch-needed")
	h.a.fetch()
	h.drain()
	s = h.a.remoteState()
	if s.Branches+s.Tags != 0 || s.FetchedBranches != 1 || s.FetchedTags != 1 || !s.fresh(time.Now()) {
		t.Fatalf("unexpected fetch result: %+v", s)
	}
	if got := strings.TrimSpace(testGit(t, local, "rev-parse", "HEAD")); got != base {
		t.Fatal("fetch changed HEAD")
	}
	if got := strings.TrimSpace(testGit(t, local, "rev-parse", "publish^{}")); got != next {
		t.Fatal("annotated tag missing")
	}
	// Changing only the annotation still changes the remote tag object.
	testGit(t, remote, "tag", "-f", "-a", "publish", "-m", "revised annotation")
	h.a.checkRemoteUpdates()
	h.drain()
	if h.a.remoteState().Tags != 1 {
		t.Fatal("moved annotated tag was not detected")
	}
	before := testGit(t, local, "rev-parse", "publish")
	h.a.fetchRemoteUpdates(true, h.a.config)
	h.drain()
	if h.a.errorText != "" || h.a.autoFetchErrors[local] == "" {
		t.Fatalf("automatic conflict: modal=%q errors=%v repo=%s state=%+v", h.a.errorText, h.a.autoFetchErrors, h.a.repo(), h.a.remoteState())
	}
	if testGit(t, local, "rev-parse", "publish") != before {
		t.Fatal("fetch overwrote a conflicting tag")
	}
}

func TestRemoteFreshnessRefspecsAndDeletedBranches(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	remote := testRepo(t)
	testCommit(t, remote, "base", "base", "base")
	testGit(t, remote, "branch", "ignored")
	testGit(t, remote, "branch", "removed")
	local := filepath.Join(t.TempDir(), "clone")
	testGit(t, remote, "clone", "-q", remote, local)
	testGit(t, local, "config", "--add", "remote.origin.fetch", "^refs/heads/ignored")
	testGit(t, remote, "branch", "-D", "removed")
	testGit(t, local, "tag", "local-only")
	s, err := inspectRemoteUpdates(context.Background(), local)
	if err != nil || s.Branches != 1 || s.RemovedBranches != 1 || s.Tags != 0 {
		t.Fatalf("unexpected removed/excluded refs: %+v %v", s, err)
	}
	h := newActionHarness(t, local)
	config := h.a.config
	config.FetchPrune = true
	h.a.fetchRemoteUpdates(false, config)
	h.drain()
	if s = h.a.remoteState(); s.Branches+s.Tags != 0 {
		t.Fatalf("prune did not resolve status: %+v", s)
	}
	testGit(t, local, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing"))
	h.a.checkRemoteUpdates()
	h.drain()
	if s = h.a.remoteState(); s.Error == "" || s.fresh(time.Now()) || h.a.errorText != "" {
		t.Fatalf("failed check reported fresh or opened modal: %+v", s)
	}
}

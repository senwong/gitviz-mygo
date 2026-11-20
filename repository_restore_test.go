package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func repositoryRestoreHarness(t *testing.T, p Preferences, path string) *actionTestHarness {
	t.Helper()
	a := newApp(p)
	a.prefsFile = path
	queue := make(chan func(), 256)
	a.post = func(fn func()) { queue <- fn }
	t.Cleanup(func() {
		if a.cancelLoad != nil {
			a.cancelLoad()
		}
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
		if a.cancelDiff != nil {
			a.cancelDiff()
		}
	})
	return &actionTestHarness{a, queue, t}
}

func TestRepositorySelectionSurvivesRestartAndDiscoveryOrder(t *testing.T) {
	first, second := testRepo(t), testRepo(t)
	testCommit(t, first, "first.txt", "first\n", "first")
	testCommit(t, second, "second.txt", "second\n", "second")
	first, _ = filepath.EvalSymlinks(first)
	second, _ = filepath.EvalSymlinks(second)
	path := filepath.Join(t.TempDir(), "settings.json")
	h := repositoryRestoreHarness(t, Preferences{Roots: []string{first, second}, Theme: "Nord", Dark: true, Remotes: true}, path)
	h.a.discoverRoots(h.a.roots)
	h.drain()
	for i, repo := range h.a.repos {
		if repo == second {
			h.a.switchRepo(i)
			break
		}
	}
	// Persistence must happen on selection, before loading or an orderly exit.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved Preferences
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.LastRepository != second || saved.Theme != "Nord" {
		t.Fatalf("selection/preferences not saved: %+v", saved)
	}
	h.drain()
	restarted := repositoryRestoreHarness(t, saved, path)
	restarted.a.discoverRoots([]string{second, first})
	restarted.drain()
	if restarted.a.repo() != second || len(restarted.a.snapshot.Commits) == 0 || restarted.a.snapshot.Commits[0].Subject != "second" {
		t.Fatal("restart did not load the last selected repository")
	}
	// Rescanning an already-open workspace prefers the current repository.
	restarted.a.prefs.LastRepository = first
	restarted.a.discoverRoots([]string{first, second})
	restarted.drain()
	if restarted.a.repo() != second {
		t.Fatal("rescan replaced the current repository with stale saved state")
	}
}

func TestRestoreRepositoryFallsBackWhenUnavailableOrExcluded(t *testing.T) {
	first, second := testRepo(t), testRepo(t)
	testCommit(t, first, "file.txt", "first\n", "first")
	testCommit(t, second, "file.txt", "second\n", "second")
	first, _ = filepath.EvalSymlinks(first)
	second, _ = filepath.EvalSymlinks(second)
	for _, tc := range []struct {
		name, last      string
		roots, excluded []string
		want            string
	}{
		{"legacy preferences", "", []string{first}, nil, first},
		{"missing repository", filepath.Join(t.TempDir(), "deleted"), []string{first}, nil, first},
		{"explicit other workspace", second, []string{first}, nil, first},
		{"excluded repository", second, []string{first, second}, []string{second}, first},
		{"no available repositories", second, []string{first}, []string{first}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			h := repositoryRestoreHarness(t, Preferences{Roots: tc.roots, LastRepository: tc.last, Excluded: tc.excluded, Dark: true, Remotes: true}, path)
			h.a.discoverRoots(tc.roots)
			h.drain()
			if h.a.repo() != tc.want || h.a.prefs.LastRepository != tc.want {
				t.Fatalf("incorrect fallback: selected=%q remembered=%q", h.a.repo(), h.a.prefs.LastRepository)
			}
		})
	}
}

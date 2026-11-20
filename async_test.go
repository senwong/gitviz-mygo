package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAsyncSelectionDropsOldDetailAndDiffResults(t *testing.T) {
	repo := testRepo(t)
	first := testCommit(t, repo, "first.txt", "first\n", "first commit")
	testCommit(t, repo, "second.txt", "second\n", "second commit")
	snapshot, err := loadSnapshot(context.Background(), repo, 500, true, "")
	if err != nil {
		t.Fatal(err)
	}
	a := newApp(Preferences{Dark: true, Remotes: true, Detail: true})
	a.repos = []string{repo}
	a.active = 0
	a.snapshot = snapshot
	a.rebuildVisible()

	queue := make(chan func(), 32)
	a.post = func(fn func()) { queue <- fn }
	a.selectCommit(0)
	a.selectCommit(1)
	timeout := time.After(15 * time.Second)
	for a.detailLoading || a.diffLoading || len(a.diffLines) == 0 {
		select {
		case fn := <-queue:
			fn()
		case <-timeout:
			t.Fatal("background details did not complete")
		}
	}
	for len(queue) > 0 {
		(<-queue)()
	}
	if a.revision != first || a.selectedCommit.SHA != first {
		t.Fatal("old selection overwrote the current selection")
	}
	if len(a.files) != 1 || a.files[0].Path != "first.txt" || !strings.Contains(a.diffText, "+first") {
		t.Fatalf("stale detail applied: %+v %q", a.files, a.diffText)
	}
	if a.errorText != "" {
		t.Fatalf("stale cancellation surfaced an error: %s", a.errorText)
	}
}

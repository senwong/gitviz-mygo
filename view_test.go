package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func uiFixture() *App {
	a := newApp(Preferences{Dark: true, Remotes: true, Detail: true})
	a.prefsFile = ""

	a.repos = []string{"/test/gitviz"}
	a.active = 0
	a.snapshot = Snapshot{Lanes: 1, Tracking: Tracking{Branch: "release", Upstream: "origin/release", Ahead: 2, Behind: 1}, Commits: []Commit{{SHA: "1234567890abcdef", Subject: "A test commit", Author: "Ada", Email: "ada@example.test", Time: time.Unix(1700000000, 0), Refs: []string{"release", "tag:v1"}}}}
	a.rebuildVisible()
	return a
}

func TestNativeStatusAndCheckoutDialog(t *testing.T) {
	a := uiFixture()
	a.busy[a.repo()] = "Checking out feature…"
	tt := ui.NewTester(a.view, 1380, 900)
	if !tt.HasText("Checking out feature…") {
		t.Fatalf("progress missing: %v", tt.Texts())
	}
	found := false
	for _, text := range tt.Texts() {
		if text == "release  ·  origin/release  ↑2 ↓1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tracking missing: %v", tt.Texts())
	}
	delete(a.busy, a.repo())
	a.syncPrompt = &SyncPrompt{Repo: a.repo(), Tracking: a.snapshot.Tracking}
	a.syncOpen = true
	tt.Frame()
	for _, label := range []string{"Not now", "Pull", "Pull Rebase"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("missing dialog choice %q: %v", label, tt.Texts())
		}
	}
	if err := tt.Click("Not now"); err != nil {
		t.Fatal(err)
	}
	if a.syncOpen || a.syncPrompt != nil {
		t.Fatal("cancel did not dismiss prompt")
	}
	if a.busy[a.repo()] != "" {
		t.Fatal("cancel started a Git operation")
	}
}

func TestNativeSearchAndTagSpacing(t *testing.T) {
	a := uiFixture()
	a.showSearch = true
	a.query = "not present"
	tt := ui.NewTester(a.view, 1380, 900)
	if !tt.HasText("0 matches") {
		t.Fatalf("empty search state missing: %v", tt.Texts())
	}
	if err := tt.Click("Close search"); err != nil {
		t.Fatal(err)
	}
	if a.query != "" || a.showSearch {
		t.Fatal("search was not cleared")
	}
	if !tt.HasText("A test commit") {
		t.Fatal("commits did not return after clearing search")
	}
	if rect, ok := tt.Find("v1"); !ok || rect.H > 22 {
		t.Fatalf("tag fills commit row: %+v %v", rect, ok)
	}
	tt.SetSize(1000, 620)
	if _, ok := tt.Find("Refresh graph"); !ok {
		t.Fatal("toolbar is inaccessible at the minimum window width")
	}
}

func TestGraphVirtualizesLargeHistory(t *testing.T) {
	a := uiFixture()
	template := a.snapshot.Commits[0]
	a.snapshot.Commits = nil
	for i := 0; i < 5000; i++ {
		c := template
		c.SHA = time.Unix(int64(i), 0).Format(time.RFC3339)
		a.snapshot.Commits = append(a.snapshot.Commits, c)
	}
	a.rebuildVisible()
	tt := ui.NewTester(a.view, 1380, 900)
	if len(tt.Texts()) > 1200 {
		t.Fatalf("the whole commit history was rendered: %d texts", len(tt.Texts()))
	}
}

package main

import (
	"strings"
	"testing"
)

func TestCurrentReferenceUsesCheckedOutLocalBranch(t *testing.T) {
	a := uiFixture()
	a.snapshot.Branches = []Branch{
		{Name: "dev-10.24.4", Full: "refs/heads/dev-10.24.4", SHA: "same", Current: true},
		{Name: "release", Full: "refs/heads/release", SHA: "same"},
		{Name: "origin/dev-10.24.4", Full: "refs/remotes/origin/dev-10.24.4", SHA: "same", Remote: true},
	}
	for ref, want := range map[string]bool{
		"dev-10.24.4": true, "dev-10.24.4\x1forigin": true, "refs/heads/dev-10.24.4\x1forigin": true,
		"release": false, "release\x1forigin": false, "origin/dev-10.24.4": false, "refs/remotes/origin/dev-10.24.4": false, "tag:dev-10.24.4": false,
	} {
		if got := a.currentReference(ref); got != want {
			t.Fatalf("%q: got %v want %v", ref, got, want)
		}
	}
	a.snapshot.Tracking.Detached = true
	if a.currentReference("dev-10.24.4\x1forigin") {
		t.Fatal("detached HEAD marked a branch as checked out")
	}
}
func TestCurrentBranchBadgeChangesWithCheckoutAndKeepsMenus(t *testing.T) {
	a := uiFixture()
	commit := a.snapshot.Commits[0]
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git"}
	a.snapshot.Branches = []Branch{{Name: "dev-10.24.4", Full: "refs/heads/dev-10.24.4", Current: true}, {Name: "release", Full: "refs/heads/release"}, {Name: "origin/dev-10.24.4", Full: "refs/remotes/origin/dev-10.24.4", Remote: true}, {Name: "origin/release", Full: "refs/remotes/origin/release", Remote: true}}
	a.snapshot.Commits[0].Refs = []string{"origin/release"}
	other := commit
	other.SHA = "other"
	other.Subject = "current branch commit"
	other.Refs = []string{"dev-10.24.4\x1forigin", "release", "tag:v1"}
	a.snapshot.Commits = append(a.snapshot.Commits, other)
	a.rebuildVisible()
	tt := newMenuTester(a, 1380, 900)
	marker, ok := tt.Find("Current branch")
	if !ok {
		t.Fatal("current branch check mark missing")
	}
	current, _ := tt.Find("origin/dev-10.24.4")
	if marker.X < current.X || marker.X+marker.W > current.X+current.W || marker.Y < current.Y || marker.Y+marker.H > current.Y+current.H {
		t.Fatal("current marker is outside the branch badge")
	}
	captureMigrationPreview(t, tt, "current-branch-dark")
	if err := tt.RightClick("origin"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Menu(), "|"), "Delete Remote Branch…") {
		t.Fatal("current badge lost remote menu")
	}
	tt.CloseMenu()
	if err := tt.RightClick("dev-10.24.4"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Menu(), "|"), "Rename Branch…") {
		t.Fatal("current badge lost local menu")
	}
	tt.CloseMenu()
	a.snapshot.Branches[0].Current = false
	a.snapshot.Branches[1].Current = true
	tt.Frame()
	marker, ok = tt.Find("Current branch")
	next, _ := tt.Find("release")
	if !ok || next.X-marker.X-marker.W < 0 || next.X-marker.X-marker.W > 8 || marker.Y+marker.H < next.Y || marker.Y > next.Y+next.H {
		t.Fatalf("marker did not follow newly checked out branch: marker=%+v release=%+v", marker, next)
	}
	a.snapshot.Tracking.Detached = true
	tt.Frame()
	if _, ok := tt.Find("Current branch"); ok {
		t.Fatal("marker remains under detached HEAD")
	}
	a.snapshot.Tracking.Detached = false
	a.snapshot.Branches[0].Current = true
	a.snapshot.Branches[1].Current = false
	a.prefs.Theme = "GitHub Light"
	a.setTheme()
	tt.Frame()
	captureMigrationPreview(t, tt, "current-branch-light")
	tt.SetSize(1000, 620)
	if _, ok := tt.Find("Current branch"); !ok {
		t.Fatal("marker lost at minimum width")
	}
}

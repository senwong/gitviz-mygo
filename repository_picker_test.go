package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestRepositoryPickerKeyboardAndSearch(t *testing.T) {
	a := uiFixture()
	a.repos = []string{"/work/alpha", "/work/beta", "/work/gamma"}
	a.active = 0
	a.post = func(func()) {}
	defer func() {
		if a.cancelLoad != nil {
			a.cancelLoad()
		}
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
	}()
	a.openRepositoryPicker()
	tt := ui.NewTester(a.view, 1380, 900)
	tt.Key(0, ui.KeyDown)
	if a.repoCursor != 1 {
		t.Fatalf("down did not move selection: %d", a.repoCursor)
	}
	tt.Key(0, ui.KeyUp)
	if a.repoCursor != 0 {
		t.Fatal("up did not move selection")
	}
	tt.Type("beta")
	if a.repoQuery != "beta" || a.repoCursor != 0 {
		t.Fatalf("search state %q %d", a.repoQuery, a.repoCursor)
	}
	if !tt.HasText("beta") || tt.HasText("gamma") {
		t.Fatal("repository filtering did not update")
	}
	tt.Key(0, ui.KeyEnter)
	if a.showRepos || a.repo() != "/work/beta" {
		t.Fatalf("Enter did not switch: %s, open=%v", a.repo(), a.showRepos)
	}
}

func TestRepositoryPickerCompactRowsAndEmptySearch(t *testing.T) {
	a := uiFixture()
	a.repos = []string{"/work/alpha", "/work/beta"}
	a.active = 0
	a.openRepositoryPicker()
	tt := ui.NewTester(a.view, 1380, 900)
	name, ok := tt.Find("alpha")
	if !ok {
		t.Fatal("repository name is missing")
	}
	row, ok := tt.Find("Repository: alpha")
	if !ok {
		t.Fatal("repository row is missing")
	}
	if row.H != 28 {
		t.Fatalf("row is not compact and left aligned: row=%+v name=%+v", row, name)
	}
	if tt.HasText("/work/alpha") || tt.HasText("/work") {
		t.Fatal("repository dropdown still displays paths")
	}
	captureMigrationPreview(t, tt, "repository-picker")
	tt.Type("no-such-repository")
	if !strings.Contains(strings.Join(tt.Texts(), " "), "No repositories match") {
		t.Fatal("empty search feedback missing")
	}
	tt.Key(0, ui.KeyEnter)
	if !a.showRepos || a.repo() != "/work/alpha" {
		t.Fatal("empty search switched repository")
	}
	tt.Key(0, ui.KeyEscape)
	if a.showRepos {
		t.Fatal("Escape did not close picker")
	}
}

func TestRepositoryPickerWorkspacePreview(t *testing.T) {
	workspace := os.Getenv("GITVIZ_TEST_WORKSPACE")
	if workspace == "" {
		t.Skip("set GITVIZ_TEST_WORKSPACE for the native picker preview")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	repos, err := discover(ctx, []string{workspace})
	if err != nil || len(repos) == 0 {
		t.Fatalf("workspace: %v", err)
	}
	a := newApp(readPreferences())
	a.prefsFile = ""

	a.repos = repos
	a.repoNames = workspaceNames([]string{workspace})
	a.active = 0
	a.config = a.configForRepo(a.repo())
	a.sortRepos(a.repo())
	a.snapshot, err = loadSnapshotConfig(ctx, a.repo(), 50, a.config, "")
	if err != nil {
		t.Fatal(err)
	}
	a.rebuildVisible()
	a.openRepositoryPicker()
	tt := ui.NewTester(a.view, 1380, 900)
	captureMigrationPreview(t, tt, "repository-picker-workspace")
	tt.Type("umu_course")
	if len(a.filteredRepositories()) == 0 {
		t.Fatal("workspace alias search failed")
	}
	for _, i := range a.filteredRepositories() {
		label := "Repository: " + a.repoLabel(a.repos[i])
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("filtered row missing: %s", label)
		}
	}
	captureMigrationPreview(t, tt, "repository-picker-filtered")
}

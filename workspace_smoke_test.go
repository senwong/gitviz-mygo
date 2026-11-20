package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestWorkspaceSmoke(t *testing.T) {
	path := os.Getenv("GITVIZ_TEST_WORKSPACE")
	if path == "" {
		t.Skip("set GITVIZ_TEST_WORKSPACE to test a real workspace")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repos, err := discover(ctx, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) == 0 {
		t.Fatal("workspace contains no repositories")
	}
	t.Logf("Discovered %d repositories", len(repos))
	for _, repo := range repos {
		s, err := loadSnapshotConfig(ctx, repo, 50, defaultConfig(), "")
		if err != nil {
			t.Errorf("%s: %v", repo, err)
			continue
		}
		t.Logf("%s: %d commits, %d branches, %d changed files, branch %s", repoName(repo), len(s.Commits), len(s.Branches), len(s.Files), s.Tracking.Branch)
	}
	a := newApp(Preferences{Dark: true, Remotes: true, Detail: true})
	a.prefsFile = ""

	a.repos = append([]string(nil), repos...)
	a.repoNames = workspaceNames([]string{path})
	a.active = -1
	queue := make(chan func(), 64)
	a.post = func(fn func()) { queue <- fn }
	defer func() {
		if a.cancelLoad != nil {
			a.cancelLoad()
		}
		if a.cancelDetail != nil {
			a.cancelDetail()
		}
		if a.cancelDiff != nil {
			a.cancelDiff()
		}
	}()
	for _, target := range []string{repos[0], repos[len(repos)-1]} {
		index := -1
		for i, repo := range a.repos {
			if repo == target {
				index = i
				break
			}
		}
		a.switchRepo(index)
		timeout := time.After(45 * time.Second)
		for a.loading || a.detailLoading || a.diffLoading {
			select {
			case fn := <-queue:
				fn()
			case <-timeout:
				t.Fatalf("switch to %s did not complete", target)
			}
		}
		if a.errorText != "" {
			t.Fatalf("switch to %s: %s", target, a.errorText)
		}
		view := ui.NewTester(a.view, 1380, 900)
		if a.repo() != target {
			t.Fatal("repository selection was lost")
		}
		if _, ok := view.Find("Switch repository"); !ok {
			t.Fatal("repository picker is missing")
		}
		shown := 0
		if a.selected >= 0 {
			shown = a.selected
		}
		if len(a.snapshot.Commits) > 0 && !view.HasText(a.messageText(a.snapshot.Commits[shown].Subject)) {
			t.Fatal("commit list did not render")
		}
		a.showRepos = true
		a.repoQuery = a.repoLabel(a.repo())
		view.Frame()
		if _, ok := view.Find(a.repoLabel(a.repo()) + "  ·  " + filepath.Dir(a.repo())); !ok {
			t.Fatal("workspace alias cannot be found in the repository picker")
		}
		a.showRepos = false
		t.Logf("Native UI switched to %s and rendered", a.repoLabel(a.repo()))
	}
}

func TestMergeDetailsUseFirstParent(t *testing.T) {
	repo := testRepo(t)
	testCommit(t, repo, "base.txt", "base\n", "base")
	testGit(t, repo, "switch", "-c", "feature")
	testCommit(t, repo, "feature.txt", "feature change\n", "feature")
	testGit(t, repo, "switch", "main")
	testCommit(t, repo, "main.txt", "main change\n", "main")
	testGit(t, repo, "merge", "--no-ff", "--no-edit", "feature")
	sha := strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD"))
	files, err := changedFiles(context.Background(), repo, sha)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "feature.txt" || !files[0].StatsKnown || files[0].Additions != 1 || files[0].Deletions != 0 {
		t.Fatalf("merge details %+v", files)
	}
	diff, err := fileDiff(context.Background(), repo, sha, "", files[0])
	if err != nil || !strings.Contains(diff, "+feature change") {
		t.Fatalf("merge diff %q %v", diff, err)
	}
}

func TestLongSubjectDoesNotPaintIntoDateColumn(t *testing.T) {
	a := uiFixture()
	a.snapshot.Commits[0].Subject = strings.Repeat("LONG TITLE ", 100)
	a.rebuildVisible()
	tt := ui.NewTester(a.view, 1380, 900)
	r, ok := tt.Find(a.dateText(a.snapshot.Commits[0]))
	if !ok {
		t.Fatal("date column is missing")
	}
	image := tt.Image()
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X) + 2; x < int(r.X+r.W)-2; x++ {
			pixel := image.RGBAAt(x, y)
			if pixel.R > 200 && pixel.G > 200 && pixel.B > 210 {
				t.Fatalf("title text leaked into the muted date column at %d,%d", x, y)
			}
		}
	}
}

func TestWorkspacePreservesFolderAliases(t *testing.T) {
	repo := testRepo(t)
	workspace := filepath.Join(t.TempDir(), "team.code-workspace")
	data, _ := json.Marshal(map[string]any{"folders": []map[string]string{{"name": "Custom repository name", "path": repo}}})
	if err := os.WriteFile(workspace, data, 0644); err != nil {
		t.Fatal(err)
	}
	a := uiFixture()
	root, _ := filepath.EvalSymlinks(repo)
	a.repos = []string{root}
	a.repoNames = workspaceNames([]string{workspace})
	a.active = 0
	a.showRepos = true
	a.repoQuery = "Custom repository name"
	tt := ui.NewTester(a.view, 1380, 900)
	if _, ok := tt.Find("Repository: Custom repository name"); !ok {
		t.Fatalf("alias was lost: %v", tt.Texts())
	}
	if tt.HasText(root) || tt.HasText(filepath.Dir(root)) {
		t.Fatal("repository path is still displayed")
	}
}

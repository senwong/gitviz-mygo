package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := gitCommand(context.Background(), repo, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func testRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	testGit(t, repo, "init", "-q", "-b", "main")
	testGit(t, repo, "config", "user.name", "Gitviz Test")
	testGit(t, repo, "config", "user.email", "gitviz@example.test")
	testGit(t, repo, "config", "commit.gpgsign", "false")
	return repo
}

func testCommit(t *testing.T, repo, path, text, message string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, path)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, path), []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	testGit(t, repo, "add", "--", path)
	testGit(t, repo, "commit", "-qm", message)
	return strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD"))
}

func TestLogDetailsAndUnusualPaths(t *testing.T) {
	repo := testRepo(t)
	path := "folder/a file\twith tab.txt"
	sha := testCommit(t, repo, path, "one\ntwo\n", "Subject\n\nA multiline\nbody")
	testGit(t, repo, "tag", "v1")
	s, err := loadSnapshot(context.Background(), repo, 500, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Commits) != 1 || s.Commits[0].SHA != sha || s.Commits[0].Body != "A multiline\nbody" {
		t.Fatalf("unexpected log: %+v", s.Commits)
	}
	if !strings.Contains(strings.Join(s.Commits[0].Refs, " "), "tag:v1") {
		t.Fatal("tag missing")
	}
	files, err := changedFiles(context.Background(), repo, sha)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != path {
		t.Fatalf("files: %+v", files)
	}
	diff, err := fileDiff(context.Background(), repo, sha, "", files[0])
	if err != nil || !strings.Contains(diff, "+two") {
		t.Fatalf("diff %q %v", diff, err)
	}
}

func TestWorkingTreeIncludesStagedUnstagedUntrackedAndUnborn(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	s, err := loadSnapshot(ctx, repo, 500, true, "")
	if err != nil || len(s.Commits) != 0 || s.Tracking.Branch != "main" {
		t.Fatalf("empty repo: %+v %v", s, err)
	}
	path := "new.txt"
	os.WriteFile(filepath.Join(repo, path), []byte("new\n"), 0644)
	diff, err := fileDiff(ctx, repo, "worktree", "", FileChange{Path: path, Untracked: true})
	if err != nil || !strings.Contains(diff, "+new") {
		t.Fatalf("untracked %q %v", diff, err)
	}
	testGit(t, repo, "add", "--", path)
	diff, err = fileDiff(ctx, repo, "worktree", "", FileChange{Path: path})
	if err != nil || !strings.Contains(diff, "+new") {
		t.Fatalf("unborn %q %v", diff, err)
	}
	testGit(t, repo, "commit", "-qm", "initial")
	os.WriteFile(filepath.Join(repo, path), []byte("staged\n"), 0644)
	testGit(t, repo, "add", "--", path)
	os.WriteFile(filepath.Join(repo, path), []byte("working\n"), 0644)
	diff, err = fileDiff(ctx, repo, "worktree", "", FileChange{Path: path})
	if err != nil || !strings.Contains(diff, "-new") || !strings.Contains(diff, "+working") {
		t.Fatalf("working %q %v", diff, err)
	}
	os.WriteFile(filepath.Join(repo, "untracked.txt"), []byte("hello"), 0644)
	files, err := readStatus(ctx, repo)
	if err != nil || len(files) != 2 {
		t.Fatalf("status %+v %v", files, err)
	}
	testGit(t, repo, "reset", "--mixed", "HEAD")
	testGit(t, repo, "restore", "--", path)
	testGit(t, repo, "mv", path, "renamed.txt")
	files, err = readStatus(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range files {
		if f.Path == "renamed.txt" && f.OldPath == path {
			found = true
		}
	}
	if !found {
		t.Fatalf("rename lost: %+v", files)
	}
}

func TestGraphMergesKeepLaneContinuity(t *testing.T) {
	commits := []Commit{{SHA: "merge", Parents: []string{"left", "right"}}, {SHA: "right", Parents: []string{"base"}}, {SHA: "left", Parents: []string{"base"}}, {SHA: "base"}}
	lanes := assignLanes(commits)
	if lanes < 2 || len(commits[0].Outgoing) != 1 {
		t.Fatalf("missing merge: %+v", commits)
	}
	if len(commits[3].Incoming) != 1 {
		t.Fatalf("missing convergence: %+v", commits[3])
	}
	for _, c := range commits {
		for _, l := range c.Through {
			if l == c.Lane {
				t.Fatal("node lane cannot be a through lane")
			}
		}
	}
	linear := []Commit{{SHA: "c", Parents: []string{"b"}}, {SHA: "b", Parents: []string{"a"}}, {SHA: "a"}}
	if assignLanes(linear) != 1 || !linear[1].Top || !linear[1].Bottom || linear[2].Bottom {
		t.Fatalf("linear graph %+v", linear)
	}
}

func TestCheckoutTrackingAndPullStrategies(t *testing.T) {
	for _, rebase := range []bool{false, true} {
		t.Run(map[bool]string{false: "merge", true: "rebase"}[rebase], func(t *testing.T) {
			ctx := context.Background()
			remote := testRepo(t)
			testCommit(t, remote, "base.txt", "base\n", "base")
			local := testRepo(t)
			testGit(t, local, "remote", "add", "origin", remote)
			testGit(t, local, "fetch", "origin")
			branches, _, err := readBranches(ctx, local)
			if err != nil {
				t.Fatal(err)
			}
			var target Branch
			for _, b := range branches {
				if b.Name == "origin/main" {
					target = b
				}
			}
			if target.Name == "" {
				t.Fatal("remote branch absent")
			}
			if err := checkoutBranch(ctx, local, target); err != nil {
				t.Fatal(err)
			}
			branches, _, _ = readBranches(ctx, local)
			tracking := readTracking(ctx, local, branches)
			if tracking.Branch != "main" || tracking.Upstream != "origin/main" || tracking.Ahead != 0 || tracking.Behind != 0 {
				t.Fatalf("tracking %+v", tracking)
			}
			testCommit(t, local, "local.txt", "local\n", "local change")
			testCommit(t, remote, "remote.txt", "remote\n", "remote change")
			testGit(t, local, "fetch", "origin")
			branches, _, _ = readBranches(ctx, local)
			tracking = readTracking(ctx, local, branches)
			if tracking.Ahead != 1 || tracking.Behind != 1 {
				t.Fatalf("direction reversed: %+v", tracking)
			}
			if pullBranch(ctx, local, "other", rebase) == nil {
				t.Fatal("pull allowed on the wrong branch")
			}
			testGit(t, local, "config", "pull.ff", "only")
			testGit(t, local, "config", "pull.rebase", "true")
			if err := pullBranch(ctx, local, "main", rebase); err != nil {
				t.Fatal(err)
			}
			commits, _, err := readLog(ctx, local, 500, true, "")
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if rebase {
				want = 1
			}
			if len(commits[0].Parents) != want {
				t.Fatalf("strategy mismatch: %+v", commits[0])
			}
			branches, _, _ = readBranches(ctx, local)
			tracking = readTracking(ctx, local, branches)
			if tracking.Behind != 0 {
				t.Fatalf("still behind: %+v", tracking)
			}
			testGit(t, local, "switch", "--detach")
			branches, _, _ = readBranches(ctx, local)
			if !readTracking(ctx, local, branches).Detached {
				t.Fatal("detached head not detected")
			}
		})
	}
}

func TestStashCancellationAndPathGuard(t *testing.T) {
	repo := testRepo(t)
	testCommit(t, repo, "a.txt", "one\n", "one")
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("two\n"), 0644)
	testGit(t, repo, "stash", "push", "-m", "saved work")
	stashes, err := readStashes(context.Background(), repo)
	if err != nil || len(stashes) != 1 || !strings.Contains(stashes[0].Message, "saved work") {
		t.Fatalf("stash %+v %v", stashes, err)
	}
	diff, err := fileDiff(context.Background(), repo, stashes[0].Ref, "", FileChange{Path: "a.txt"})
	if err != nil || !strings.Contains(diff, "+two") {
		t.Fatalf("stash diff %q %v", diff, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = gitCommand(ctx, repo, "status"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation %v", err)
	}
	if _, err = fileDiff(context.Background(), repo, "worktree", "", FileChange{Path: "../outside", Untracked: true}); err == nil {
		t.Fatal("nonlocal path accepted")
	}
}

func TestWorkspaceAndDiscovery(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "team.code-workspace")
	paths := parseWorkspace(file, `{"folders":[{"path":"../alpha"},{"path":"nested/repo"}]}`)
	if len(paths) != 2 || paths[1] != filepath.Join(base, "nested/repo") {
		t.Fatalf("paths %v", paths)
	}
	repo := testRepo(t)
	workspace := filepath.Join(base, "repositories.gitviz-workspace")
	os.WriteFile(workspace, []byte("# repositories\n"+repo+"\n"+repo+"\n"), 0644)
	repos, err := discover(context.Background(), []string{workspace, repo})
	canonical, _ := filepath.EvalSymlinks(repo)
	if err != nil || len(repos) != 1 || repos[0] != canonical {
		t.Fatalf("discovery %v %v", repos, err)
	}
}

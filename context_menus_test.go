package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func actionTitles(actions []contextAction) []string {
	out := []string{}
	for _, action := range actions {
		out = append(out, action.Title)
	}
	return out
}
func findAction(t *testing.T, actions []contextAction, key string) contextAction {
	t.Helper()
	for _, action := range actions {
		if action.Key == key {
			return action
		}
	}
	t.Fatalf("action %s missing in %v", key, actionTitles(actions))
	return contextAction{}
}
func formField(t *testing.T, a *App, label string) *actionField {
	t.Helper()
	if a.contextForm == nil {
		t.Fatal("no form")
	}
	for _, field := range a.contextForm.Fields {
		if field.Label == label {
			return field
		}
	}
	t.Fatalf("field %s missing", label)
	return nil
}
func TestGitGraphContextMenuNamesOrderAndConditions(t *testing.T) {
	a := uiFixture()
	c := a.snapshot.Commits[0]
	c.Parents = []string{"parent"}
	a.snapshot.Commits[0] = c
	a.snapshot.Head = c.SHA
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git"}
	current := Branch{Name: "release", Full: "refs/heads/release", SHA: c.SHA, Current: true}
	local := Branch{Name: "feature", Full: "refs/heads/feature", SHA: c.SHA}
	remote := Branch{Name: "origin/feature", Full: "refs/remotes/origin/feature", SHA: c.SHA, Remote: true}
	a.snapshot.Branches = []Branch{current, local, remote}
	tests := []struct {
		name    string
		actions []contextAction
		want    []string
	}{
		{"commit", a.commitActions(c), []string{"Add Tag…", "Create Branch…", "Checkout…", "Cherry Pick…", "Revert…", "Drop…", "Merge into current branch…", "Rebase current branch on this Commit…", "Reset current branch to this Commit…", "Copy Commit Hash to Clipboard", "Copy Commit Subject to Clipboard"}},
		{"local", a.branchActions(local), []string{"Checkout Branch", "Rename Branch…", "Delete Branch…", "Merge into current branch…", "Rebase current branch on Branch…", "Push Branch…", "Create Pull Request…", "Create Archive", "Select in Branches Dropdown", "Copy Branch Name to Clipboard"}},
		{"current", a.branchActions(current), []string{"Rename Branch…", "Push Branch…", "Create Pull Request…", "Create Archive", "Select in Branches Dropdown", "Copy Branch Name to Clipboard"}},
		{"remote", a.branchActions(remote), []string{"Checkout Branch…", "Delete Remote Branch…", "Fetch into local branch…", "Merge into current branch…", "Pull into current branch…", "Create Pull Request", "Create Archive", "Select in Branches Dropdown", "Copy Branch Name to Clipboard"}},
		{"annotated tag", a.tagActions(Tag{Name: "v1", Annotated: true}), []string{"View Details", "Delete Tag…", "Push Tag…", "Create Archive", "Copy Tag Name to Clipboard"}},
		{"lightweight tag", a.tagActions(Tag{Name: "v2"}), []string{"Delete Tag…", "Push Tag…", "Create Archive", "Copy Tag Name to Clipboard"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := actionTitles(test.actions); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %q\nwant %q", got, test.want)
			}
		})
	}
	a.filterRef = local.Full
	if got := findAction(t, a.branchActions(local), "branch-select").Title; got != "Unselect in Branches Dropdown" {
		t.Fatal(got)
	}
	var copied string
	a.clipboardWrite = func(value string) { copied = value }
	c.Body = "body must not be copied"
	findAction(t, a.commitActions(c), "copy-message").Run()
	if copied != c.Subject {
		t.Fatal("copy subject included body")
	}
	a.prefs.AlwaysAcceptCheckout = true
	if findAction(t, a.commitActions(c), "checkout").Title != "Checkout" {
		t.Fatal("Always Accept label mismatch")
	}
	a.snapshot.Remotes = nil
	for _, action := range a.tagActions(Tag{Name: "v1"}) {
		if action.Key == "tag-push" {
			t.Fatal("Push Tag without remote")
		}
	}
	a.snapshot.Commits[0].Parents = nil
	if a.dropPossible(a.snapshot.Commits[0]) {
		t.Fatal("root commit can be dropped")
	}
}
func TestGraphReferenceContextMenusDoNotFallThrough(t *testing.T) {
	a := uiFixture()
	c := a.snapshot.Commits[0]
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git"}
	a.snapshot.Branches = []Branch{{Name: "release", Full: "refs/heads/release", Current: true}, {Name: "origin/release", Full: "refs/remotes/origin/release", Remote: true}}
	a.snapshot.Tags = map[string]Tag{"v1": {Name: "v1", SHA: c.SHA, Annotated: true}}
	tt := newMenuTester(a, 1380, 900)
	for _, test := range []struct{ target, want, absent, name string }{{"A test commit", "Add Tag…", "Delete Tag…", "commit-menu"}, {"release", "Rename Branch…", "Add Tag…", "branch-current-menu"}, {"v1", "View Details", "Add Tag…", "tag-menu"}} {
		if err := tt.RightClick(test.target); err != nil {
			t.Fatal(err)
		}
		labels := strings.Join(tt.Menu(), "|")
		if !strings.Contains(labels, test.want) || strings.Contains(labels, test.absent) {
			t.Fatalf("%s: %s", test.target, labels)
		}
		captureMigrationPreview(t, tt, test.name)
		tt.CloseMenu()
	}
	a.snapshot.Commits[0].Refs = []string{"release\x1forigin", "tag:v1"}
	tt.Frame()
	if err := tt.RightClick("origin"); err != nil {
		t.Fatal(err)
	}
	labels := strings.Join(tt.Menu(), "|")
	if !strings.Contains(labels, "Delete Remote Branch…") || strings.Contains(labels, "Rename Branch…") {
		t.Fatalf("combined remote routed incorrectly: %s", labels)
	}
	captureMigrationPreview(t, tt, "branch-remote-menu")
	tt.CloseMenu()
	a.snapshot.Commits[0].Refs = []string{"refs/heads/release", "tag:refs/tags/v1"}
	tt.Frame()
	if err := tt.RightClick("refs/tags/v1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Menu(), "|"), "Delete Tag…") {
		t.Fatal("full ref tag misrouted")
	}
}
func TestContextFormsCancelAndValidate(t *testing.T) {
	a := uiFixture()
	c := a.snapshot.Commits[0]
	tt := newMenuTester(a, 1000, 620)
	actions := a.commitActions(c)
	for _, action := range actions {
		if !action.Mutating {
			continue
		}
		action.Run()
		tt.Frame()
		if a.contextForm == nil {
			t.Fatalf("%s has no form", action.Title)
		}
		captureMigrationPreview(t, tt, "form-"+action.Key)
		tt.Key(ui.Cmd, ui.KeyB)
		if a.showBranches {
			t.Fatal("form allowed background shortcut")
		}
		tt.Key(0, ui.KeyEscape)
		if a.contextForm != nil {
			t.Fatalf("%s did not cancel", action.Title)
		}
		if len(a.busy) > 0 {
			t.Fatal("cancel started operation")
		}
	}
	a.createBranchForm(c)
	tt.Frame()
	if err := tt.Click("Confirm Create Branch"); err != nil {
		t.Fatal(err)
	}
	if a.contextForm == nil || a.contextForm.Error == "" {
		t.Fatal("invalid name closed form")
	}
	a.contextForm.Repo = "other"
	a.submitActionForm(a.contextForm)
	if !strings.Contains(a.contextForm.Error, "repository changed") {
		t.Fatal("repository guard missing")
	}
}

type actionTestHarness struct {
	a     *App
	queue chan func()
	t     *testing.T
}

func newActionHarness(t *testing.T, repo string) *actionTestHarness {
	t.Helper()
	a := uiFixture()
	a.repos = []string{repo}
	a.active = 0
	a.configRepo = repo

	queue := make(chan func(), 256)
	a.post = func(fn func()) { queue <- fn }
	h := &actionTestHarness{a, queue, t}
	h.refresh()
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
	return h
}
func (h *actionTestHarness) refresh() {
	h.t.Helper()
	s, err := loadSnapshotConfig(context.Background(), h.a.repo(), 500, h.a.config, "")
	if err != nil {
		h.t.Fatal(err)
	}
	h.a.snapshot = s
	h.a.rebuildVisible()
}
func (h *actionTestHarness) drain() {
	h.t.Helper()
	deadline := time.After(15 * time.Second)
	for h.a.busy[h.a.repo()] != "" || h.a.loading || h.a.detailLoading || h.a.diffLoading || h.a.remoteState().Checking || len(h.queue) > 0 {
		select {
		case fn := <-h.queue:
			fn()
		case <-deadline:
			h.t.Fatal("operation did not settle")
		}
	}
	if h.a.errorText != "" {
		h.t.Fatal(h.a.errorText)
	}
}
func (h *actionTestHarness) submit() {
	h.t.Helper()
	f := h.a.contextForm
	if f == nil {
		h.t.Fatal("no form")
	}
	h.a.submitActionForm(f)
	if f.Error != "" {
		h.t.Fatal(f.Error)
	}
	h.drain()
}

func TestContextActionsCreatePushDeleteBranchAndTags(t *testing.T) {
	repo := testRepo(t)
	sha := testCommit(t, repo, "base.txt", "base\n", "base")
	remote := filepath.Join(t.TempDir(), "remote.git")
	testGit(t, repo, "init", "--bare", remote)
	testGit(t, repo, "remote", "add", "origin", remote)
	testGit(t, repo, "push", "-u", "origin", "main")
	h := newActionHarness(t, repo)
	a := h.a
	c := a.snapshot.Commits[0]
	findAction(t, a.commitActions(c), "create-branch").Run()
	formField(t, a, "Name").Value = "feature"
	h.submit()
	if got := strings.TrimSpace(testGit(t, repo, "rev-parse", "refs/heads/feature")); got != sha {
		t.Fatal("created branch wrong target")
	}
	h.refresh()
	var feature Branch
	for _, b := range a.snapshot.Branches {
		if b.Name == "feature" {
			feature = b
		}
	}
	findAction(t, a.branchActions(feature), "branch-push").Run()
	h.submit()
	if got := strings.TrimSpace(testGit(t, repo, "--git-dir="+remote, "rev-parse", "refs/heads/feature")); got != sha {
		t.Fatal("push did not use clicked branch")
	}
	if got := strings.TrimSpace(testGit(t, repo, "symbolic-ref", "--short", "HEAD")); got != "main" {
		t.Fatal("push switched branch")
	}
	findAction(t, a.commitActions(c), "create-tag").Run()
	formField(t, a, "Name").Value = "v1"
	formField(t, a, "Message").Value = "release notes"
	h.submit()
	if got := strings.TrimSpace(testGit(t, repo, "cat-file", "-t", "refs/tags/v1")); got != "tag" {
		t.Fatal("annotated tag not created")
	}
	if !strings.Contains(testGit(t, repo, "cat-file", "-p", "refs/tags/v1"), "release notes") {
		t.Fatal("tag message lost")
	}
	h.refresh()
	tag := a.snapshot.Tags["v1"]
	if !tag.Annotated {
		t.Fatal("tag metadata lost")
	}
	findAction(t, a.tagActions(tag), "tag-push").Run()
	h.submit()
	if _, err := gitCommand(context.Background(), remote, "rev-parse", "refs/tags/v1"); err != nil {
		t.Fatal(err)
	}
	findAction(t, a.tagActions(tag), "tag-delete").Run()
	formField(t, a, "Also delete on remote").Value = "origin"
	h.submit()
	if out := testGit(t, repo, "tag", "--list", "v1"); strings.TrimSpace(out) != "" {
		t.Fatal("local tag not deleted")
	}
	if out := testGit(t, repo, "ls-remote", "--tags", "origin"); strings.Contains(out, "refs/tags/v1") {
		t.Fatal("remote tag not deleted")
	}
	h.refresh()
	for _, b := range a.snapshot.Branches {
		if b.Name == "feature" {
			feature = b
		}
	}
	findAction(t, a.branchActions(feature), "branch-delete").Run()
	formField(t, a, "Delete this branch on the remote(s)").Checked = true
	h.submit()
	if out := testGit(t, repo, "branch", "--list", "feature"); strings.TrimSpace(out) != "" {
		t.Fatal("local branch not deleted")
	}
	if out := testGit(t, repo, "ls-remote", "--heads", "origin", "feature"); strings.TrimSpace(out) != "" {
		t.Fatal("remote branch not deleted")
	}
}
func TestContextMergeCherryPickAndDropOptions(t *testing.T) {
	t.Run("squash no commit", func(t *testing.T) {
		repo := testRepo(t)
		base := testCommit(t, repo, "base", "base", "base")
		testGit(t, repo, "switch", "-c", "feature")
		testCommit(t, repo, "feature", "feature", "feature")
		testGit(t, repo, "switch", "main")
		h := newActionHarness(t, repo)
		a := h.a
		var b Branch
		for _, candidate := range a.snapshot.Branches {
			if candidate.Name == "feature" {
				b = candidate
			}
		}
		findAction(t, a.branchActions(b), "branch-merge").Run()
		formField(t, a, "Squash Commits").Checked = true
		formField(t, a, "No Commit").Checked = true
		h.submit()
		if strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD")) != base {
			t.Fatal("squash unexpectedly committed")
		}
		if !strings.Contains(testGit(t, repo, "diff", "--cached", "--name-only"), "feature") {
			t.Fatal("squash did not stage changes")
		}
	})
	t.Run("merge mainline cherry pick", func(t *testing.T) {
		repo := testRepo(t)
		testCommit(t, repo, "base", "base", "base")
		testGit(t, repo, "switch", "-c", "feature")
		testCommit(t, repo, "feature", "feature", "feature")
		testGit(t, repo, "switch", "main")
		testCommit(t, repo, "main", "main", "main")
		testGit(t, repo, "merge", "--no-ff", "--no-edit", "feature")
		merge := strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD"))
		testGit(t, repo, "switch", "-c", "target", "HEAD^1")
		h := newActionHarness(t, repo)
		a := h.a
		var c Commit
		for _, candidate := range a.snapshot.Commits {
			if candidate.SHA == merge {
				c = candidate
			}
		}
		a.cherryPickForm(c)
		formField(t, a, "Record Origin").Checked = true
		h.submit()
		if !strings.Contains(testGit(t, repo, "log", "-1", "--format=%B"), "cherry picked from commit "+merge) {
			t.Fatal("origin not recorded")
		}
		if _, err := os.Stat(filepath.Join(repo, "feature")); err != nil {
			t.Fatal("merge cherry pick mainline failed")
		}
	})
	t.Run("drop interior commit", func(t *testing.T) {
		repo := testRepo(t)
		testCommit(t, repo, "base", "base", "base")
		drop := testCommit(t, repo, "drop", "drop", "drop")
		keep := testCommit(t, repo, "keep", "keep", "keep")
		h := newActionHarness(t, repo)
		a := h.a
		var c Commit
		for _, candidate := range a.snapshot.Commits {
			if candidate.SHA == drop {
				c = candidate
			}
		}
		findAction(t, a.commitActions(c), "drop").Run()
		h.submit()
		if strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD")) == keep {
			t.Fatal("descendant not rewritten")
		}
		if _, err := os.Stat(filepath.Join(repo, "drop")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("dropped file remains")
		}
		if _, err := os.Stat(filepath.Join(repo, "keep")); err != nil {
			t.Fatal("descendant lost")
		}
	})
}
func TestTagPushGuardDoesNotPublishUnpushedCommit(t *testing.T) {
	repo := testRepo(t)
	testCommit(t, repo, "base", "base", "base")
	remote := filepath.Join(t.TempDir(), "remote.git")
	testGit(t, repo, "init", "--bare", remote)
	testGit(t, repo, "remote", "add", "origin", remote)
	testGit(t, repo, "push", "-u", "origin", "main")
	sha := testCommit(t, repo, "private", "private", "private")
	testGit(t, repo, "tag", "private-tag")
	err := pushTagChecked(context.Background(), repo, Tag{Name: "private-tag", SHA: sha}, []string{"origin"}, false)
	var check *tagPushCheck
	if !errors.As(err, &check) {
		t.Fatalf("missing push confirmation: %v", err)
	}
	if out := testGit(t, repo, "ls-remote", "--tags", "origin"); strings.TrimSpace(out) != "" {
		t.Fatal("guard published tag")
	}
	if err := pushTagChecked(context.Background(), repo, check.Tag, check.Remotes, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(testGit(t, repo, "ls-remote", "--tags", "origin"), "private-tag") {
		t.Fatal("confirmed push failed")
	}
}

func TestRemoteBranchContextActionsUseClickedReference(t *testing.T) {
	repo := testRepo(t)
	testCommit(t, repo, "base", "base", "base")
	remote := filepath.Join(t.TempDir(), "remote.git")
	testGit(t, repo, "init", "--bare", remote)
	testGit(t, repo, "remote", "add", "origin", remote)
	testGit(t, repo, "push", "-u", "origin", "main")
	testGit(t, repo, "switch", "-c", "feature")
	featureSHA := testCommit(t, repo, "feature", "feature", "feature")
	testGit(t, repo, "push", "origin", "feature")
	testGit(t, repo, "switch", "main")
	testGit(t, repo, "branch", "-f", "feature", "main")
	h := newActionHarness(t, repo)
	a := h.a
	var clicked Branch
	for _, b := range a.snapshot.Branches {
		if b.Name == "origin/feature" {
			clicked = b
		}
	}
	findAction(t, a.branchActions(clicked), "remote-branch-fetch").Run()
	h.submit()
	if strings.TrimSpace(testGit(t, repo, "rev-parse", "feature")) != featureSHA {
		t.Fatal("fetch did not advance clicked local branch")
	}
	if strings.TrimSpace(testGit(t, repo, "symbolic-ref", "--short", "HEAD")) != "main" {
		t.Fatal("fetch switched current branch")
	}
	findAction(t, a.branchActions(clicked), "remote-branch-pull").Run()
	h.submit()
	if strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD")) != featureSHA {
		t.Fatal("pull did not use clicked remote branch")
	}
	h.refresh()
	a.remoteCheckoutForm(clicked)
	formField(t, a, "Name").Value = "custom-name"
	h.submit()
	if strings.TrimSpace(testGit(t, repo, "symbolic-ref", "--short", "HEAD")) != "custom-name" {
		t.Fatal("custom local checkout name ignored")
	}
	if strings.TrimSpace(testGit(t, repo, "rev-parse", "--abbrev-ref", "@{upstream}")) != "origin/feature" {
		t.Fatal("tracking incorrect")
	}
	a.contextForm = nil
	findAction(t, a.branchActions(clicked), "remote-branch-delete").Run()
	h.submit()
	if out := testGit(t, repo, "ls-remote", "--heads", "origin", "feature"); strings.TrimSpace(out) != "" {
		t.Fatal("remote branch not deleted")
	}
}
func TestBranchSelectionCombinesReferences(t *testing.T) {
	a := uiFixture()
	a.post = func(func()) {}
	a.toggleBranch("refs/heads/main")
	a.toggleBranch("refs/heads/feature")
	if a.cancelLoad != nil {
		defer a.cancelLoad()
	}
	if !a.selectedBranch("refs/heads/main") || !a.selectedBranch("refs/heads/feature") {
		t.Fatal("selection replaced earlier branch")
	}
	args := logArguments(a.config, 50, a.filterRef)
	joined := strings.Join(args, "|")
	if !strings.Contains(joined, "|refs/heads/main|refs/heads/feature|") {
		t.Fatalf("invalid multi-ref log args: %v", args)
	}
	a.toggleBranch("refs/heads/main")
	if a.filterRef != "refs/heads/feature" {
		t.Fatal("unselect removed wrong branch")
	}
	a.toggleBranch("refs/heads/feature")
	if a.filterRef != "" {
		t.Fatal("last unselect did not restore all branches")
	}
}
func TestContextRevertMergeResetAndRebase(t *testing.T) {
	t.Run("revert merge parent", func(t *testing.T) {
		repo := testRepo(t)
		testCommit(t, repo, "base", "base", "base")
		testGit(t, repo, "switch", "-c", "feature")
		testCommit(t, repo, "feature", "feature", "feature")
		testGit(t, repo, "switch", "main")
		testCommit(t, repo, "main", "main", "main")
		testGit(t, repo, "merge", "--no-ff", "--no-edit", "feature")
		h := newActionHarness(t, repo)
		a := h.a
		c := a.snapshot.Commits[0]
		a.revertForm(c)
		h.submit()
		if _, err := os.Stat(filepath.Join(repo, "feature")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("merge revert did not use mainline parent")
		}
		if _, err := os.Stat(filepath.Join(repo, "main")); err != nil {
			t.Fatal("revert removed mainline")
		}
	})
	t.Run("no ff merge", func(t *testing.T) {
		repo := testRepo(t)
		testCommit(t, repo, "base", "base", "base")
		testGit(t, repo, "switch", "-c", "feature")
		sha := testCommit(t, repo, "feature", "feature", "feature")
		testGit(t, repo, "switch", "main")
		h := newActionHarness(t, repo)
		a := h.a
		a.mergeForm(sha, sha, "", "")
		formField(t, a, "Create a new commit even if fast-forward is possible").Checked = true
		h.submit()
		parents := strings.Fields(testGit(t, repo, "show", "-s", "--format=%P", "HEAD"))
		if len(parents) != 2 {
			t.Fatal("no-ff ignored")
		}
	})
	t.Run("reset soft", func(t *testing.T) {
		repo := testRepo(t)
		base := testCommit(t, repo, "base", "base", "base")
		testCommit(t, repo, "next", "next", "next")
		h := newActionHarness(t, repo)
		a := h.a
		var c Commit
		for _, candidate := range a.snapshot.Commits {
			if candidate.SHA == base {
				c = candidate
			}
		}
		a.resetForm(c)
		formField(t, a, "Reset Mode").Value = "Soft - Keep all changes, but reset head"
		h.submit()
		if strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD")) != base {
			t.Fatal("reset wrong commit")
		}
		if !strings.Contains(testGit(t, repo, "diff", "--cached", "--name-only"), "next") {
			t.Fatal("soft reset discarded index")
		}
	})
	t.Run("rebase clicked commit", func(t *testing.T) {
		repo := testRepo(t)
		testCommit(t, repo, "base", "base", "base")
		testGit(t, repo, "switch", "-c", "feature")
		testCommit(t, repo, "feature", "feature", "feature")
		testGit(t, repo, "switch", "main")
		target := testCommit(t, repo, "main", "main", "main")
		testGit(t, repo, "switch", "feature")
		h := newActionHarness(t, repo)
		a := h.a
		a.rebaseForm(target, target)
		formField(t, a, "Ignore Date").Checked = false
		h.submit()
		if strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD^")) != target {
			t.Fatal("rebase used wrong target")
		}
		if _, err := os.Stat(filepath.Join(repo, "feature")); err != nil {
			t.Fatal("rebase lost local commit")
		}
	})
}
func TestArchiveAndBranchPullRequestTarget(t *testing.T) {
	repo := testRepo(t)
	sha := testCommit(t, repo, "base", "base", "base")
	testGit(t, repo, "branch", "feature")
	path := filepath.Join(t.TempDir(), "a path with spaces.zip")
	if err := createArchive(context.Background(), repo, sha, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatal("archive not written")
	}
	a := uiFixture()
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git"}
	var opened string
	a.openExternal = func(url string) error { opened = url; return nil }
	b := Branch{Name: "feature", Full: "refs/heads/feature"}
	findAction(t, a.branchActions(b), "create-pr").Run()
	formField(t, a, "Push branch before creating the Pull Request").Checked = false
	a.submitActionForm(a.contextForm)
	if !strings.Contains(opened, "main...feature") {
		t.Fatalf("PR used current branch instead of clicked branch: %s", opened)
	}
	command := terminalRebaseCommand("/tmp/a b'c", "refs/heads/feature")
	if !strings.Contains(command, "'\"'\"'") || !strings.Contains(command, "git rebase --interactive 'refs/heads/feature'") {
		t.Fatal("terminal command not shell quoted")
	}
}

func TestContextMenuOptionsScreenshots(t *testing.T) {
	a := uiFixture()
	a.snapshot.Remotes = map[string]string{"origin": "https://github.com/example/repo.git", "upstream": "https://github.com/upstream/repo.git"}
	c := a.snapshot.Commits[0]
	c.Parents = []string{"parent-one", "parent-two"}
	branch := Branch{Name: "feature", Full: "refs/heads/feature"}
	remote := Branch{Name: "origin/feature", Full: "refs/remotes/origin/feature", Remote: true}
	a.snapshot.Branches = []Branch{branch, remote}
	cases := []struct {
		name string
		open func()
	}{
		{"merge", func() { a.mergeForm(c.SHA, c.SHA, "", "") }},
		{"cherry-pick-merge", func() { a.cherryPickForm(c) }},
		{"revert-merge", func() { a.revertForm(c) }},
		{"rebase", func() { a.rebaseForm(c.SHA, c.SHA) }},
		{"create-tag", func() { a.createTagForm(c) }},
		{"push-branch", func() { a.pushBranchForm(branch) }},
		{"delete-branch", func() { a.deleteBranchForm(branch) }},
		{"checkout-remote", func() { a.remoteCheckoutForm(remote) }},
		{"push-tag", func() { a.tagPushForm(Tag{Name: "v1", SHA: c.SHA, Annotated: true}) }},
		{"delete-tag", func() { a.deleteTagForm(Tag{Name: "v1", SHA: c.SHA}) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.open()
			tt := newMenuTester(a, 1000, 620)
			captureMigrationPreview(t, tt, "options-"+test.name)
			if rect, ok := tt.Find("Confirm " + a.contextForm.Confirm); !ok || rect.Y+rect.H > 620 {
				t.Fatalf("confirmation clipped: %+v", rect)
			}
			tt.Key(0, ui.KeyEscape)
			if a.contextForm != nil {
				t.Fatal("dialog not dismissed")
			}
		})
	}
	a.showBranches = true
	tt := newMenuTester(a, 1380, 900)
	if err := tt.RightClick("feature"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Menu(), "|"), "Push Branch…") {
		t.Fatal("branch dropdown not using context actions")
	}
	captureMigrationPreview(t, tt, "branch-local-menu")
}

func newMenuTester(a *App, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) { c.SetMenuPresentation(ui.PopoverMenus); a.view(c) }, width, height)
}

func TestContextMenuUIExecutesCreateBranchAndTagReplacement(t *testing.T) {
	repo := testRepo(t)
	old := testCommit(t, repo, "base", "base", "base")
	testGit(t, repo, "tag", "v1")
	latest := testCommit(t, repo, "next", "next", "next")
	h := newActionHarness(t, repo)
	a := h.a
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.RightClick("next"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Create Branch…"); err != nil {
		t.Fatal(err)
	}
	tt.Type("from-ui")
	if err := tt.Click("Confirm Create Branch"); err != nil {
		t.Fatal(err)
	}
	h.drain()
	tt.Frame()
	if strings.TrimSpace(testGit(t, repo, "rev-parse", "from-ui")) != latest {
		t.Fatal("UI branch creation wrong target")
	}
	a.createTagForm(a.snapshot.Commits[0])
	formField(t, a, "Name").Value = "v1"
	formField(t, a, "Message").Value = "replacement"
	h.submit()
	if a.contextForm == nil || a.contextForm.Confirm != "Yes, replace the existing tag" {
		t.Fatal("duplicate tag did not require replacement confirmation")
	}
	if strings.TrimSpace(testGit(t, repo, "rev-parse", "v1")) != old {
		t.Fatal("tag replaced before confirmation")
	}
	h.submit()
	if strings.TrimSpace(testGit(t, repo, "rev-parse", "v1^{}")) != latest {
		t.Fatal("confirmed replacement wrong target")
	}
}

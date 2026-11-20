package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
)

func readTags(ctx context.Context, repo string) (map[string]Tag, error) {
	out, err := gitCommand(ctx, repo, "for-each-ref", "--format=%(refname:strip=2)%00%(objecttype)%00%(objectname)%00%(*objectname)", "refs/tags/")
	if err != nil {
		return nil, err
	}
	tags := map[string]Tag{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Split(line, "\x00")
		if len(f) != 4 {
			continue
		}
		sha := f[2]
		if f[3] != "" {
			sha = f[3]
		}
		tags[f[0]] = Tag{Name: f[0], SHA: sha, Annotated: f[1] == "tag"}
	}
	return tags, nil
}
func remoteNames(remotes map[string]string) []string {
	names := make([]string, 0, len(remotes))
	for name := range remotes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
func (a *App) remoteBranchParts(b Branch) (string, string) {
	names := remoteNames(a.snapshot.Remotes)
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		if branch, ok := strings.CutPrefix(b.Name, name+"/"); ok {
			return name, branch
		}
	}
	return "", ""
}
func (a *App) selectedBranch(full string) bool {
	for _, ref := range strings.Split(a.filterRef, "\x1e") {
		if ref == full {
			return true
		}
	}
	return false
}
func (a *App) toggleBranch(full string) {
	refs := []string{}
	found := false
	for _, ref := range strings.Split(a.filterRef, "\x1e") {
		if ref == full {
			found = true
		} else if ref != "" {
			refs = append(refs, ref)
		}
	}
	if !found {
		refs = append(refs, full)
	}
	a.filterRef = strings.Join(refs, "\x1e")
	a.showBranches = false
	a.load(false)
}
func (a *App) dropPossible(commit Commit) bool {
	if len(commit.Parents) != 1 {
		return false
	}
	children := map[string][]Commit{}
	for _, c := range a.snapshot.Commits {
		for _, parent := range c.Parents {
			children[parent] = append(children[parent], c)
		}
	}
	seen := map[string]bool{}
	for !seen[commit.SHA] {
		seen[commit.SHA] = true
		if len(commit.Parents) > 1 || len(children[commit.SHA]) > 1 {
			return false
		}
		if commit.SHA == a.snapshot.Head {
			return true
		}
		if len(children[commit.SHA]) != 1 {
			return false
		}
		commit = children[commit.SHA][0]
	}
	return false
}
func runCommands(ctx context.Context, repo string, commands ...[]string) error {
	for _, args := range commands {
		if _, err := gitCommand(ctx, repo, args...); err != nil {
			return err
		}
	}
	return nil
}
func validateRef(ctx context.Context, repo, name string, branch bool) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("Enter a name")
	}
	var err error
	if branch {
		_, err = gitCommand(ctx, repo, "check-ref-format", "--branch", name)
	} else {
		_, err = gitCommand(ctx, repo, "check-ref-format", "refs/tags/"+name)
	}
	return err
}
func (a *App) runCommands(label string, commands ...[]string) {
	a.operate(label, func(ctx context.Context, repo string) error { return runCommands(ctx, repo, commands...) }, false)
}
func quoteShell(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
func terminalRebaseCommand(repo, ref string) string {
	return "cd -- " + quoteShell(repo) + " && git rebase --interactive " + quoteShell(ref)
}
func launchInteractiveRebase(repo, ref string) error {
	command := terminalRebaseCommand(repo, ref)
	script := "tell application \"Terminal\"\nactivate\ndo script " + strconv.Quote(command) + "\nend tell"
	output, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Open Terminal: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
func (a *App) archive(ref string) {
	repo := a.repo()
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, Title: "Create Archive", DefaultPath: filepath.Base(ref) + ".zip"})
		a.update(func() {
			if err != nil {
				a.fail(err)
				return
			}
			if path == "" || repo != a.repo() {
				return
			}
			a.operate("Creating Archive…", func(ctx context.Context, repo string) error { return createArchive(ctx, repo, ref, path) }, false)
		})
	}()
}

func (a *App) copyText(value string) {
	if a.clipboardWrite != nil {
		a.clipboardWrite(value)
	} else {
		mygo.Clipboard.WriteText(value)
	}
}

func createArchive(ctx context.Context, repo, ref, path string) error {
	format := "zip"
	if strings.HasSuffix(path, ".tar") {
		format = "tar"
	}
	if strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz") {
		format = "tar.gz"
	}
	return runCommands(ctx, repo, []string{"archive", "--format=" + format, "--output=" + path, ref})
}

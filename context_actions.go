package main

import (
	"context"
	"fmt"
	"strings"
)

func (a *App) mergeForm(ref, name string, pullRemote, pullBranch string) {
	noff := checkField("Create a new commit even if fast-forward is possible", false)
	squash := checkField("Squash Commits", false)
	noCommit := checkField("No Commit", false)
	fields := []*actionField{noff, squash, noCommit}
	title, confirm := "Merge into current branch", "Yes, merge"
	if pullRemote != "" {
		fields = fields[:2]
		title = "Pull into current branch"
		confirm = "Yes, pull"
	}
	a.showActionForm(title, "Apply "+name+" to the current branch "+a.snapshot.Tracking.Branch+"?", confirm, fields, func() error {
		args := []string{"merge", "--no-edit"}
		if pullRemote != "" {
			args = []string{"pull", "--no-edit", "--no-rebase"}
		}
		if noff.Checked {
			args = append(args, "--no-ff")
		}
		if squash.Checked {
			args = append(args, "--squash")
		}
		if noCommit.Checked {
			args = append(args, "--no-commit")
		}
		if pullRemote != "" {
			args = append(args, "--", pullRemote, pullBranch)
		} else {
			args = append(args, "--", ref)
		}
		a.runGit(title+"…", args...)
		return nil
	})
}
func (a *App) rebaseForm(ref, name string) {
	interactive := checkField("Launch Interactive Rebase in new Terminal", false)
	ignoreDate := checkField("Ignore Date", true)
	a.showActionForm("Rebase current branch", "Rebase "+a.snapshot.Tracking.Branch+" on "+name+"?", "Yes, rebase", []*actionField{interactive, ignoreDate}, func() error {
		if interactive.Checked {
			return launchInteractiveRebase(a.repo(), ref)
		}
		args := []string{"-c", "core.editor=true", "rebase"}
		if ignoreDate.Checked {
			args = append(args, "--ignore-date")
		}
		args = append(args, ref)
		a.runGit("Rebasing…", args...)
		return nil
	})
}
func (a *App) commitCheckoutForm(c Commit) {
	if a.prefs.AlwaysAcceptCheckout {
		a.runGit("Checking out Commit…", "switch", "--detach", c.SHA)
		return
	}
	always := checkField("Always Accept", false)
	a.showActionForm("Checkout", "Check out commit "+shortSHA(c.SHA)+"? This will result in a detached HEAD state.", "Yes, checkout", []*actionField{always}, func() error {
		if always.Checked {
			a.prefs.AlwaysAcceptCheckout = true
			a.persist()
		}
		a.runGit("Checking out Commit…", "switch", "--detach", c.SHA)
		return nil
	})
}
func (a *App) cherryPickForm(c Commit) {
	fields := []*actionField{}
	var parent *actionField
	if len(c.Parents) > 1 {
		options := []string{}
		for i, p := range c.Parents {
			options = append(options, fmt.Sprintf("%d: %s", i+1, shortSHA(p)))
		}
		parent = selectField("Parent Hash", options[0], options...)
		fields = append(fields, parent)
	}
	origin := checkField("Record Origin", false)
	noCommit := checkField("No Commit", false)
	empty := checkField("Allow Empty Commits", false)
	fields = append(fields, origin, noCommit, empty)
	a.showActionForm("Cherry Pick", "Cherry pick commit "+shortSHA(c.SHA)+"?", "Yes, cherry pick", fields, func() error {
		args := []string{"-c", "core.editor=true", "cherry-pick"}
		if empty.Checked && !noCommit.Checked {
			args = append(args, "--keep-redundant-commits")
		}
		if noCommit.Checked {
			args = append(args, "--no-commit")
		}
		if origin.Checked {
			args = append(args, "-x")
		}
		if parent != nil {
			n, _, _ := strings.Cut(parent.Value, ":")
			args = append(args, "-m", n)
		}
		args = append(args, c.SHA)
		a.runGit("Cherry picking Commit…", args...)
		return nil
	})
}
func (a *App) revertForm(c Commit) {
	fields := []*actionField{}
	var parent *actionField
	if len(c.Parents) > 1 {
		options := []string{}
		for i, p := range c.Parents {
			options = append(options, fmt.Sprintf("%d: %s", i+1, shortSHA(p)))
		}
		parent = selectField("Parent Hash", options[0], options...)
		fields = append(fields, parent)
	}
	a.showActionForm("Revert", "Revert commit "+shortSHA(c.SHA)+"?", "Yes, revert", fields, func() error {
		args := []string{"revert", "--no-edit"}
		if parent != nil {
			n, _, _ := strings.Cut(parent.Value, ":")
			args = append(args, "-m", n)
		}
		args = append(args, c.SHA)
		a.runGit("Reverting Commit…", args...)
		return nil
	})
}
func (a *App) resetForm(c Commit) {
	mode := selectField("Reset Mode", "Mixed - Keep working tree, but reset index", "Soft - Keep all changes, but reset head", "Mixed - Keep working tree, but reset index", "Hard - Discard all changes")
	a.showActionForm("Reset current branch to this Commit", "Reset "+a.snapshot.Tracking.Branch+" to "+shortSHA(c.SHA)+"? Hard discards tracked working changes.", "Yes, reset", []*actionField{mode}, func() error {
		value, _, _ := strings.Cut(strings.ToLower(mode.Value), " ")
		if value != "soft" && value != "mixed" && value != "hard" {
			return fmt.Errorf("Choose a reset mode")
		}
		a.runGit("Resetting to Commit…", "reset", "--"+value, c.SHA)
		return nil
	})
}
func (a *App) createBranchForm(c Commit) {
	name := textField("Name", "")
	checkout := checkField("Check out", false)
	var submit func(bool) error
	submit = func(force bool) error {
		if err := validateRef(context.Background(), a.repo(), name.Value, true); err != nil {
			return err
		}
		exists := false
		for _, b := range a.snapshot.Branches {
			if !b.Remote && b.Name == name.Value {
				exists = true
			}
		}
		if exists && !force {
			a.showActionForm("Create Branch", "A branch named "+name.Value+" already exists. Replace the existing branch?", "Yes, replace the existing branch", nil, func() error { return submit(true) })
			return nil
		}
		if checkout.Checked && !force {
			a.runGit("Creating Branch…", "switch", "-c", name.Value, c.SHA)
		} else {
			args := []string{"branch"}
			if force {
				args = append(args, "--force")
			}
			args = append(args, "--", name.Value, c.SHA)
			commands := [][]string{args}
			if checkout.Checked {
				commands = append(commands, []string{"switch", "--", name.Value})
			}
			a.runCommands("Creating Branch…", commands...)
		}
		return nil
	}
	a.showActionForm("Create Branch", "Create branch at commit "+shortSHA(c.SHA)+":", "Create Branch", []*actionField{name, checkout}, func() error { return submit(false) })
}
func (a *App) createTagForm(c Commit) {
	name := textField("Name", "")
	kind := selectField("Type", "Annotated", "Annotated", "Lightweight")
	message := textField("Message", "")
	remote := selectField("Push to remote", "Don't push", append([]string{"Don't push"}, remoteNames(a.snapshot.Remotes)...)...)
	fields := []*actionField{name, kind, message}
	if len(a.snapshot.Remotes) > 0 {
		fields = append(fields, remote)
	}
	var submit func(bool) error
	submit = func(force bool) error {
		if err := validateRef(context.Background(), a.repo(), name.Value, false); err != nil {
			return err
		}
		if kind.Value == "Lightweight" && message.Value != "" {
			return fmt.Errorf("A message can only be added to an annotated tag")
		}
		if _, exists := a.snapshot.Tags[name.Value]; exists && !force {
			a.showActionForm("Add Tag", "A tag named "+name.Value+" already exists. Replace it?", "Yes, replace the existing tag", nil, func() error { return submit(true) })
			return nil
		}
		args := []string{"tag"}
		if force {
			args = append(args, "--force")
		}
		if kind.Value == "Annotated" {
			args = append(args, "--annotate", "--message", message.Value)
		}
		args = append(args, "--", name.Value, c.SHA)
		tag := Tag{Name: name.Value, SHA: c.SHA, Annotated: kind.Value == "Annotated"}
		if remote.Value == "Don't push" {
			a.runGit("Adding Tag…", args...)
		} else {
			skip := a.prefs.PushTagSkipRemoteCheck
			remoteName := remote.Value
			a.operate("Adding Tag…", func(ctx context.Context, repo string) error {
				if err := runCommands(ctx, repo, args); err != nil {
					return err
				}
				return pushTagChecked(ctx, repo, tag, []string{remoteName}, skip)
			}, false)
		}
		return nil
	}
	a.showActionForm("Add Tag", "Add tag to commit "+shortSHA(c.SHA)+":", "Add Tag", fields, func() error { return submit(false) })
}
func (a *App) remoteChoices(label string, preferred ...string) *actionField {
	names := remoteNames(a.snapshot.Remotes)
	selected := map[string]bool{}
	if len(names) > 0 {
		selected[names[0]] = true
	}
	if _, ok := a.snapshot.Remotes["origin"]; ok {
		selected = map[string]bool{"origin": true}
	}
	for _, upstream := range preferred {
		for _, name := range names {
			if upstream == name || strings.HasPrefix(upstream, name+"/") {
				selected = map[string]bool{name: true}
				break
			}
		}
	}
	return &actionField{Label: label, Multiple: true, Options: names, Selected: selected}
}
func selectedRemotes(f *actionField) []string {
	result := []string{}
	for _, option := range f.Options {
		if f.Selected[option] {
			result = append(result, option)
		}
	}
	return result
}
func (a *App) pushBranchForm(b Branch) {
	remotes := a.remoteChoices("Push to Remote(s)", b.Upstream)
	upstream := checkField("Set Upstream", true)
	mode := selectField("Push Mode", "Normal", "Normal", "Force With Lease", "Force")
	a.showActionForm("Push Branch", "Push branch "+b.Name+"?", "Yes, push", []*actionField{remotes, upstream, mode}, func() error {
		chosen := selectedRemotes(remotes)
		if len(chosen) == 0 {
			return fmt.Errorf("Select at least one remote")
		}
		commands := [][]string{}
		for _, remote := range chosen {
			args := []string{"push"}
			if upstream.Checked {
				args = append(args, "--set-upstream")
			}
			if mode.Value == "Force With Lease" {
				args = append(args, "--force-with-lease")
			}
			if mode.Value == "Force" {
				args = append(args, "--force")
			}
			args = append(args, "--", remote, b.Full+":refs/heads/"+b.Name)
			commands = append(commands, args)
		}
		a.runCommands("Pushing Branch…", commands...)
		return nil
	})
}
func (a *App) deleteBranchForm(b Branch) {
	force := checkField("Force Delete", false)
	onRemote := checkField("Delete this branch on the remote(s)", false)
	remotes := []string{}
	for _, remote := range remoteNames(a.snapshot.Remotes) {
		for _, other := range a.snapshot.Branches {
			if other.Remote && other.Name == remote+"/"+b.Name {
				remotes = append(remotes, remote)
			}
		}
	}
	fields := []*actionField{force}
	if len(remotes) > 0 {
		fields = append(fields, onRemote)
	}
	a.showActionForm("Delete Branch", "Delete branch "+b.Name+"? Force Delete can discard unmerged work.", "Yes, delete", fields, func() error {
		flag := "-d"
		if force.Checked {
			flag = "-D"
		}
		commands := [][]string{{"branch", flag, "--", b.Name}}
		if onRemote.Checked {
			for _, remote := range remotes {
				commands = append(commands, []string{"push", "--delete", "--", remote, "refs/heads/" + b.Name})
			}
		}
		a.runCommands("Deleting Branch…", commands...)
		return nil
	})
}
func (a *App) remoteCheckoutForm(b Branch) {
	remote, branch := a.remoteBranchParts(b)
	if branch == "" {
		branch = b.Name
	}
	name := textField("Name", branch)
	a.showActionForm("Checkout Branch", "Enter the name of the local branch to check out from "+b.Name+":", "Checkout Branch", []*actionField{name}, func() error {
		if err := validateRef(context.Background(), a.repo(), name.Value, true); err != nil {
			return err
		}
		for _, local := range a.snapshot.Branches {
			if !local.Remote && local.Name == name.Value {
				label := "Checkout the existing branch"
				if remote != "" {
					label += " & pull changes"
				}
				a.showActionForm("Checkout Branch", "The name "+name.Value+" is already used by another branch. Cancel to choose another name.", label, nil, func() error {
					a.operate("Checking out Branch…", func(ctx context.Context, repo string) error {
						if err := runCommands(ctx, repo, []string{"switch", "--", name.Value}); err != nil {
							return err
						}
						if remote != "" {
							return runCommands(ctx, repo, []string{"pull", "--no-rebase", "--no-edit", "--", remote, branch})
						}
						return nil
					}, true)
					return nil
				})
				return nil
			}
		}
		a.operate("Checking out "+name.Value+"…", func(ctx context.Context, repo string) error {
			return runCommands(ctx, repo, []string{"switch", "--track", "-c", name.Value, b.Full})
		}, true)
		return nil
	})
}
func (a *App) tagDetails(tag Tag) {
	repo := a.repo()
	go func() {
		out, err := gitCommand(context.Background(), repo, "cat-file", "-p", "refs/tags/"+tag.Name)
		a.update(func() {
			if repo != a.repo() {
				return
			}
			if err != nil {
				a.fail(err)
				return
			}
			a.showActionForm("Tag Details", out, "Close", nil, func() error { return nil })
		})
	}()
}
func (a *App) tagPushForm(tag Tag) {
	remotes := a.remoteChoices("Push to Remote(s)", a.snapshot.Tracking.Upstream)
	a.showActionForm("Push Tag", "Push tag "+tag.Name+" to the selected remote(s)?", "Yes, push", []*actionField{remotes}, func() error {
		chosen := selectedRemotes(remotes)
		if len(chosen) == 0 {
			return fmt.Errorf("Select at least one remote")
		}
		a.pushTags(tag, chosen, false)
		return nil
	})
}

type tagPushCheck struct {
	Tag              Tag
	Remotes, Missing []string
}

func (e *tagPushCheck) Error() string {
	return "The commit tagged by " + e.Tag.Name + " is not on remote(s): " + strings.Join(e.Missing, ", ")
}
func pushTagChecked(ctx context.Context, repo string, tag Tag, remotes []string, skip bool) error {
	if len(remotes) == 0 {
		return fmt.Errorf("Select at least one remote")
	}
	if !skip {
		missing := []string{}
		out, err := gitCommand(ctx, repo, "for-each-ref", "--format=%(refname)", "--contains="+tag.SHA, "refs/remotes/")
		if err != nil {
			return err
		}
		for _, remote := range remotes {
			found := false
			for _, ref := range strings.Fields(out) {
				if strings.HasPrefix(ref, "refs/remotes/"+remote+"/") {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, remote)
			}
		}
		if len(missing) > 0 {
			return &tagPushCheck{Tag: tag, Remotes: append([]string(nil), remotes...), Missing: missing}
		}
	}
	for _, remote := range remotes {
		if err := runCommands(ctx, repo, []string{"push", "--", remote, "refs/tags/" + tag.Name + ":refs/tags/" + tag.Name}); err != nil {
			return err
		}
	}
	return nil
}
func (a *App) pushTags(tag Tag, remotes []string, skip bool) {
	skip = skip || a.prefs.PushTagSkipRemoteCheck
	a.operate("Pushing Tag…", func(ctx context.Context, repo string) error {
		return pushTagChecked(ctx, repo, tag, remotes, skip)
	}, false)
}
func (a *App) confirmTagPush(e *tagPushCheck) {
	always := checkField("Always Proceed", false)
	a.showActionForm("Push Tag", e.Error()+". Pushing it may also publish commits that have not been pushed. Proceed?", "Proceed to Push", []*actionField{always}, func() error {
		if always.Checked {
			a.prefs.PushTagSkipRemoteCheck = true
			a.persist()
		}
		a.pushTags(e.Tag, e.Remotes, true)
		return nil
	})
}
func (a *App) deleteTagForm(tag Tag) {
	remote := selectField("Also delete on remote", "Don't delete on any remote", append([]string{"Don't delete on any remote"}, remoteNames(a.snapshot.Remotes)...)...)
	fields := []*actionField{}
	if len(a.snapshot.Remotes) > 0 {
		fields = append(fields, remote)
	}
	a.showActionForm("Delete Tag", "Delete tag "+tag.Name+"?", "Yes, delete", fields, func() error {
		commands := [][]string{{"tag", "--delete", "--", tag.Name}}
		if remote.Value != "Don't delete on any remote" {
			commands = append(commands, []string{"push", "--delete", "--", remote.Value, "refs/tags/" + tag.Name})
		}
		a.runCommands("Deleting Tag…", commands...)
		return nil
	})
}

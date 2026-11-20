package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

type contextAction struct {
	Key, Title string
	Group      int
	Mutating   bool
	Run        func()
}

func (a *App) renderContextMenu(m *ui.Menu, actions []contextAction) {
	group := -1
	for _, action := range actions {
		hidden := false
		for _, key := range a.config.HiddenActions {
			if key == action.Key {
				hidden = true
				break
			}
		}
		if hidden {
			continue
		}
		if group != -1 && group != action.Group {
			m.Separator()
		}
		group = action.Group
		if m.Item(action.Title).Disabled(action.Mutating && a.busy[a.repo()] != "").Chosen() {
			action.Run()
		}
	}
}
func (a *App) commitMenu(m *ui.Menu, c Commit) { a.renderContextMenu(m, a.commitActions(c)) }
func (a *App) branchMenu(m *ui.Menu, b Branch) { a.renderContextMenu(m, a.branchActions(b)) }
func (a *App) tagMenu(m *ui.Menu, tag Tag)     { a.renderContextMenu(m, a.tagActions(tag)) }
func (a *App) commitActions(c Commit) []contextAction {
	checkoutTitle := "Checkout…"
	if a.prefs.AlwaysAcceptCheckout {
		checkoutTitle = "Checkout"
	}
	actions := []contextAction{
		{"create-tag", "Add Tag…", 0, true, func() { a.createTagForm(c) }},
		{"create-branch", "Create Branch…", 0, true, func() { a.createBranchForm(c) }},
		{"checkout", checkoutTitle, 1, true, func() { a.commitCheckoutForm(c) }},
		{"cherry-pick", "Cherry Pick…", 1, true, func() { a.cherryPickForm(c) }},
		{"revert", "Revert…", 1, true, func() { a.revertForm(c) }},
	}
	if a.dropPossible(c) {
		actions = append(actions, contextAction{"drop", "Drop…", 1, true, func() {
			a.showActionForm("Drop", "Permanently drop commit "+shortSHA(c.SHA)+"? Descendant commits on the current branch will be rewritten.", "Yes, drop", nil, func() error {
				limit, config := max(500, len(a.snapshot.Commits)), cloneConfig(a.config)
				a.operate("Dropping Commit…", func(ctx context.Context, repo string) error {
					s, err := loadSnapshotConfig(ctx, repo, limit, config, "")
					if err != nil {
						return err
					}
					check := &App{snapshot: s}
					if !check.dropPossible(c) {
						return fmt.Errorf("The graph changed; this commit can no longer be dropped")
					}
					return runCommands(ctx, repo, []string{"-c", "core.editor=true", "rebase", "--onto", c.SHA + "^", c.SHA})
				}, false)
				return nil
			})
		}})
	}
	actions = append(actions,
		contextAction{"merge", "Merge into current branch…", 2, true, func() { a.mergeForm(c.SHA, shortSHA(c.SHA), "", "") }},
		contextAction{"rebase", "Rebase current branch on this Commit…", 2, true, func() { a.rebaseForm(c.SHA, shortSHA(c.SHA)) }},
		contextAction{"reset", "Reset current branch to this Commit…", 2, true, func() { a.resetForm(c) }},
		contextAction{"copy-sha", "Copy Commit Hash to Clipboard", 3, false, func() { a.copyText(c.SHA) }},
		contextAction{"copy-message", "Copy Commit Subject to Clipboard", 3, false, func() { a.copyText(c.Subject) }})
	return actions
}
func (a *App) branchActions(b Branch) []contextAction {
	actions := []contextAction{}
	add := func(key, title string, group int, mutating bool, run func()) {
		actions = append(actions, contextAction{key, title, group, mutating, run})
	}
	remote, branch := a.remoteBranchParts(b)
	if b.Remote {
		add("branch-checkout", "Checkout Branch…", 0, true, func() { a.remoteCheckoutForm(b) })
		if remote != "" {
			add("remote-branch-delete", "Delete Remote Branch…", 0, true, func() {
				a.showActionForm("Delete Remote Branch", "Delete remote branch "+b.Name+"?", "Yes, delete", nil, func() error {
					a.runGit("Deleting Remote Branch…", "push", "--delete", "--", remote, "refs/heads/"+branch)
					return nil
				})
			})
			for _, local := range a.snapshot.Branches {
				if !local.Remote && !local.Current && local.Name == branch {
					add("remote-branch-fetch", "Fetch into local branch…", 0, true, func() {
						force := checkField("Force Fetch", false)
						a.showActionForm("Fetch into local branch", "Fetch "+b.Name+" into local branch "+branch+"?", "Yes, fetch", []*actionField{force}, func() error {
							refspec := "refs/heads/" + branch + ":refs/heads/" + branch
							if force.Checked {
								refspec = "+" + refspec
							}
							a.runGit("Fetching Branch…", "fetch", "--", remote, refspec)
							return nil
						})
					})
					break
				}
			}
		}
		add("branch-merge", "Merge into current branch…", 0, true, func() { a.mergeForm(b.Full, b.Name, "", "") })
		if remote != "" {
			add("remote-branch-pull", "Pull into current branch…", 0, true, func() { a.mergeForm(b.Full, b.Name, remote, branch) })
		}
	} else {
		if !b.Current {
			add("branch-checkout", "Checkout Branch", 0, true, func() { a.checkout(b) })
		}
		add("branch-rename", "Rename Branch…", 0, true, func() {
			name := textField("Name", b.Name)
			a.showActionForm("Rename Branch", "Enter the new name for branch "+b.Name+":", "Rename Branch", []*actionField{name}, func() error {
				if err := validateRef(context.Background(), a.repo(), name.Value, true); err != nil {
					return err
				}
				a.runGit("Renaming Branch…", "branch", "-m", b.Name, name.Value)
				return nil
			})
		})
		if !b.Current {
			add("branch-delete", "Delete Branch…", 0, true, func() { a.deleteBranchForm(b) })
			add("branch-merge", "Merge into current branch…", 0, true, func() { a.mergeForm(b.Full, b.Name, "", "") })
			add("branch-rebase", "Rebase current branch on Branch…", 0, true, func() { a.rebaseForm(b.Full, b.Name) })
		}
		if len(a.snapshot.Remotes) > 0 {
			add("branch-push", "Push Branch…", 0, true, func() { a.pushBranchForm(b) })
		}
	}
	if action := a.viewIssueAction(b.Name); action != nil {
		actions = append(actions, *action)
	}
	source := b.Name
	url := a.branchPullRequestLink(b)
	if b.Remote {
		source = branch
	}
	if url != "" && (!b.Remote || remote != "" && branch != "HEAD") {
		if b.Remote {
			add("create-pr", "Create Pull Request", 1, false, func() { a.openURL(url) })
		} else {
			add("create-pr", "Create Pull Request…", 1, true, func() {
				push := checkField("Push branch before creating the Pull Request", true)
				a.showActionForm("Create Pull Request", "Create a Pull Request for branch "+source+"?", "Yes, create Pull Request", []*actionField{push}, func() error {
					if !push.Checked {
						a.openURL(url)
						return nil
					}
					names := remoteNames(a.snapshot.Remotes)
					if len(names) == 0 {
						return fmt.Errorf("No remote configured")
					}
					target := names[0]
					if _, ok := a.snapshot.Remotes["origin"]; ok {
						target = "origin"
					}
					a.operate("Creating Pull Request…", func(ctx context.Context, repo string) error {
						if err := runCommands(ctx, repo, []string{"push", "--", target, b.Full + ":refs/heads/" + b.Name}); err != nil {
							return err
						}
						a.update(func() { a.openURL(url) })
						return nil
					}, false)
					return nil
				})
			})
		}
	}
	add("branch-archive", "Create Archive", 2, false, func() { a.archive(b.Full) })
	label := "Select in Branches Dropdown"
	if a.selectedBranch(b.Full) {
		label = "Unselect in Branches Dropdown"
	}
	add("branch-select", label, 2, false, func() { a.toggleBranch(b.Full) })
	add("branch-copy", "Copy Branch Name to Clipboard", 3, false, func() { a.copyText(b.Name) })
	return actions
}
func (a *App) tagActions(tag Tag) []contextAction {
	actions := []contextAction{}
	if tag.Annotated {
		actions = append(actions, contextAction{"tag-details", "View Details", 0, false, func() { a.tagDetails(tag) }})
	}
	actions = append(actions, contextAction{"tag-delete", "Delete Tag…", 0, true, func() { a.deleteTagForm(tag) }})
	if len(a.snapshot.Remotes) > 0 {
		actions = append(actions, contextAction{"tag-push", "Push Tag…", 0, true, func() { a.tagPushForm(tag) }})
	}
	actions = append(actions, contextAction{"tag-archive", "Create Archive", 1, false, func() { a.archive("refs/tags/" + tag.Name) }}, contextAction{"tag-copy", "Copy Tag Name to Clipboard", 1, false, func() { a.copyText(tag.Name) }})
	return actions
}
func (a *App) branchPullRequestLink(b Branch) string {
	base := "main"
	for _, other := range a.snapshot.Branches {
		if other.Symbolic != "" {
			_, base, _ = strings.Cut(strings.TrimPrefix(other.Symbolic, "refs/remotes/"), "/")
			break
		}
	}
	raw := a.remoteURL()
	head := b.Name
	if b.Remote {
		remote, branch := a.remoteBranchParts(b)
		raw = a.snapshot.Remotes[remote]
		head = branch
	}
	return providerLink(raw, a.config.PrProvider, base, head, "", true)
}
func (a *App) viewIssueAction(ref string) *contextAction {
	matches := issuePattern.FindAllStringSubmatch(ref, -1)
	links := []string{}
	numbers := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		if seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		link := providerLink(a.remoteURL(), a.config.IssueProvider, "", "", match[1], false)
		if link != "" {
			links = append(links, link)
			numbers = append(numbers, "#"+match[1])
		}
	}
	if len(links) == 0 {
		return nil
	}
	title := "View Issue"
	if len(links) > 1 {
		title += "…"
	}
	return &contextAction{"open-issue", title, 1, false, func() {
		if len(links) == 1 {
			a.openURL(links[0])
			return
		}
		choice := selectField("Issue", numbers[0], numbers...)
		a.showActionForm("View Issue", "Select which issue you want to view for this branch:", "View Issue", []*actionField{choice}, func() error {
			for i, n := range numbers {
				if n == choice.Value {
					a.openURL(links[i])
					break
				}
			}
			return nil
		})
	}}
}
func (a *App) referenceMenu(m *ui.Menu, ref string, c Commit) {
	if strings.HasPrefix(ref, "tag:") {
		name := strings.TrimPrefix(strings.TrimPrefix(ref, "tag:"), "refs/tags/")
		tag, ok := a.snapshot.Tags[name]
		if !ok {
			tag = Tag{Name: name, SHA: c.SHA}
		}
		a.tagMenu(m, tag)
		return
	}
	name := baseRef(ref)
	for _, b := range a.snapshot.Branches {
		if b.Name == name || b.Full == ref {
			a.branchMenu(m, b)
			return
		}
	}
}
func (a *App) referencePill(c *ui.Context, ref string, commit Commit, available ...float32) {
	width := float32(260)
	if len(available) > 0 && available[0] > 0 {
		width = available[0]
	}
	colors := a.colors()
	color := colors[commit.Lane%len(colors)]
	local, remote, combined := strings.Cut(ref, "\x1f")
	label := referenceDisplayName(ref)
	current := a.currentReference(ref)
	neutral := ui.Hex("#808080")
	pill := ui.Row(c.Key(ref)).MinHeight(20).MaxWidth(width).Shrink(1).Radius(5).Background(neutral.Alpha(.15)).Border(1, neutral.Alpha(.75)).AlignItems(ui.Center)
	pill.ContextMenu(func(m *ui.Menu) { a.referenceMenu(m, local, commit) })
	tooltip := local
	if combined {
		pill.Label(label)
		tooltip = "Local: " + local + "\nRemote: " + remote + "/" + baseRef(local)
	}
	if current {
		pill.Border(1, color).Current(ui.CurrentItem)
		tooltip = "Current branch (HEAD)\n" + tooltip
	}
	pill.Tooltip(tooltip)
	pill.Children(func() {
		icon := branchIcon
		if strings.HasPrefix(ref, "tag:") {
			icon = tagIcon
		}
		block := ui.Box(c).Size(18, 18).Shrink(0).Background(color).Center()
		if current {
			block.Label("Current branch").Tooltip("Checked-out local branch: " + baseRef(local))
		}
		block.Children(func() { ui.Icon(c, icon).Size(14, 14).TextColor(a.uiTheme.Background) })
		name := strings.TrimPrefix(local, "tag:")
		text := a.referenceText(c, name).MinWidth(0).Padding(1, 5).FontSize(12).TextColor(a.uiTheme.Text).Tooltip(local)
		if current {
			text.Bold()
		}
		if combined {
			remoteRef := "refs/remotes/" + remote + "/" + baseRef(local)
			ui.Row(c.Key("remote")).Height(18).Shrink(0).PaddingX(5).BorderWidth(0, 0, 0, 1).BorderColor(neutral.Alpha(.45)).AlignItems(ui.Center).Children(func() {
				a.referenceText(c, remote).FontSize(12).Italic().TextColor(a.uiTheme.Text).SingleLine()
			}).ContextMenu(func(m *ui.Menu) { a.referenceMenu(m, remoteRef, commit) }).Tooltip("Remote branch: " + remoteRef)
		}
	})
	if pill.DoubleClicked() && !strings.HasPrefix(ref, "tag:") {
		for _, b := range a.snapshot.Branches {
			if b.Name == baseRef(local) {
				if b.Remote {
					a.remoteCheckoutForm(b)
				} else {
					a.checkout(b)
				}
				break
			}
		}
	}
}

func referenceDisplayName(ref string) string {
	local, remote, combined := strings.Cut(ref, "\x1f")
	if combined {
		prefix := remote
		if strings.HasPrefix(local, "refs/heads/") {
			prefix = "refs/remotes/" + remote
		}
		return prefix + "/" + baseRef(local)
	}
	return strings.TrimPrefix(local, "tag:")
}
func referenceSearchText(refs []string) string {
	var text strings.Builder
	for _, ref := range refs {
		text.WriteString(strings.ReplaceAll(ref, "\x1f", " "))
		text.WriteByte(' ')
		text.WriteString(referenceDisplayName(ref))
		text.WriteByte('\n')
	}
	return text.String()
}

func (a *App) currentReference(ref string) bool {
	local, _, _ := strings.Cut(ref, "\x1f")
	if a.snapshot.Tracking.Detached || strings.HasPrefix(local, "tag:") || strings.HasPrefix(local, "refs/remotes/") {
		return false
	}
	for _, branch := range a.snapshot.Branches {
		if branch.Current && !branch.Remote && (local == branch.Name || local == branch.Full) {
			return true
		}
	}
	return false
}

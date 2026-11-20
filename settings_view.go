package main

import (
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

type settingSpec struct {
	Key, Label, Group, Description string
	Reload                         bool
}

var settingSpecs = []settingSpec{
	{"branches", "Local branches", "History", "Include commits reachable from local branches.", true},
	{"remotes", "Remote branches", "History", "Include commits reachable from remote-tracking branches.", true},
	{"tags", "Tags", "History", "Include commits reachable from tags.", true},
	{"first_parent", "First parent only", "History", "Follow the mainline through merges.", true},
	{"include_reflogs", "Include reflogs", "History", "Include commits recorded in reflogs.", true},
	{"only_tags", "Tag-only history", "History", "Show commits reachable only from tags, excluding branches and remotes.", true},
	{"use_mailmap", "Use mailmap", "History", "Resolve author names and emails through .mailmap.", true},
	{"scroll_to_head", "Reveal HEAD on open", "History", "Select and reveal the checked-out commit when a repository opens.", false},
	{"remote_heads", "Remote default-branch references", "References", "Show symbolic references such as origin/HEAD.", true},
	{"full_refs", "Full reference names", "References", "Show refs/heads and refs/remotes prefixes.", true},
	{"combine_refs", "Combine local and remote labels", "References", "Pair a local branch with matching remote labels.", true},
	{"branch_globs", "Branch patterns", "References", "Separate patterns with ;, for example heads/feature/*;heads/main.", false},
	{"dim_non_head_commits", "Dim commits outside HEAD", "Appearance", "Reduce emphasis on commits outside the current ancestry.", true},
	{"show_commit_tooltip", "Commit tooltips", "Appearance", "Show commit metadata when hovering over a row.", false},
	{"graph_style", "Graph connections", "Appearance", "Rounded or angular connections between lanes.", false},
	{"color_preset", "Lane color preset", "Appearance", "Default colors, categorical colors or monochrome.", false},
	{"lane_colors", "Custom lane colors", "Appearance", "Optional hex colors separated with ;, for example #61afef;#98c379.", false},
	{"columns_date", "Date column", "Columns", "Show the selected date format.", false},
	{"columns_author", "Author column", "Columns", "Show commit authors.", false},
	{"columns_commit", "SHA column", "Columns", "Show abbreviated commit hashes.", false},
	{"date_width", "Date column width", "Columns", "Column width in display points.", false},
	{"author_width", "Author column width", "Columns", "Column width in display points.", false},
	{"commit_width", "SHA column width", "Columns", "Column width in display points.", false},
	{"date_commit", "Use committer date", "Columns", "Use the committer date instead of the author date.", false},
	{"date_short", "Short dates", "Columns", "Show the date without time.", false},
	{"relative_dates", "Relative dates", "Columns", "Show elapsed time, such as 3 days ago.", false},
	{"show_detail", "Show details", "Files", "Keep the selected commit's detail panel visible.", false},
	{"file_tree", "File tree", "Files", "Group changed files into collapsible folders.", false},
	{"compact_folders", "Compact folders", "Files", "Combine folders that contain a single child folder.", false},
	{"show_uncommitted", "Working changes", "Files", "Show the working tree's changed files.", true},
	{"include_untracked", "Include untracked files", "Files", "Include new files that have not been staged.", true},
	{"show_stashes", "Show stashes", "Files", "Load stash entries for this repository.", true},
	{"emoji", "Emoji shortcodes", "Messages", "Replace gitmoji and common :shortcodes:.", false},
	{"emoji_mappings", "Custom emoji mappings", "Messages", "Use code:emoji pairs separated with ;, for example shipit:🚢.", false},
	{"markdown", "Markdown messages", "Messages", "Render bold, italic and inline code in commit bodies.", false},
	{"pr_provider", "Pull request URL template", "Messages", "Optional URL with {host}, {owner}, {repo}, {base} and {head}.", false},
	{"issue_provider", "Issue URL template", "Messages", "Optional URL with {host}, {owner}, {repo} and {issue}.", false},
	{"fetch_prune", "Prune remote references", "Remotes", "Remove stale remote-tracking branches when fetching.", false},
	{"fetch_prune_tags", "Prune tags", "Remotes", "Allow fetch to prune local tags absent from the remote.", false},
	{"repo_order", "Repository order", "Workspace", "Sort by name or path, or preserve discovery order.", false},
	{"hidden_actions", "Hidden context-menu actions", "Menus", "Action keys separated with ;, for example revert;reset;branch-delete.", false},
}
var settingsSections = []string{"Appearance", "History", "References", "Columns", "Files", "Messages", "Remotes", "Workspace", "Menus"}

func (a *App) settings(c *ui.Context) {
	justOpened := a.showSettings && !a.settingsWereOpen
	a.settingsWereOpen = a.showSettings
	ui.Modal(c.Key("settings"), &a.showSettings, func() {
		t := c.Theme()
		ui.Column(c).Width(860).Height(570).Padding(18).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Settings").FontSize(18).Bold().Grow(1)
				ui.Text(c, a.repoLabel(a.repo())).FontSize(12).TextColor(t.TextMuted)
				if ui.Button(c, "Done").Clicked() {
					a.showSettings = false
				}
			})
			search := ui.TextInput(c, &a.settingsQuery).FillWidth().Placeholder("Search settings…").Label("Search settings")
			if justOpened {
				search.Focus()
			}
			ui.Row(c).Grow(1).FillWidth().MinHeight(0).Gap(16).Children(func() {
				a.settingsNavigation(c)
				ui.Scroll(c).Grow(1).FillHeight().Children(func() {
					section := a.settingsSection
					if section == "" {
						section = "Appearance"
					}
					if section == "Appearance" || a.settingsQuery != "" {
						a.themeSetting(c)
					}
					matches := 0
					for _, spec := range settingSpecs {
						if a.settingsQuery == "" && spec.Group != section {
							continue
						}
						if a.settingsQuery != "" && !strings.Contains(strings.ToLower(spec.Label+" "+spec.Description+" "+spec.Key), strings.ToLower(a.settingsQuery)) {
							continue
						}
						a.settingRow(c, spec)
						matches++
					}
					if a.settingsQuery != "" && matches == 0 {
						ui.Text(c, "No matching settings").Padding(12).TextColor(t.TextMuted)
					}
					if a.settingsQuery == "" {
						switch section {
						case "Files":
							a.fileSettings(c)
						case "Remotes":
							a.remoteSettings(c)
						case "Workspace":
							a.workspaceSettings(c)
						case "Menus":
							a.menuVisibility(c)
						}
					}
				})
			})
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Settings are saved automatically.").Grow(1).FontSize(11).TextColor(t.TextMuted)
				if ui.Button(c, "Import .gitviz.conf").Disabled(a.repo() == "").Clicked() {
					a.importConfig()
				}
				if ui.Button(c, "Export .gitviz.conf").Disabled(a.repo() == "").Clicked() {
					a.exportConfig()
				}
			})
		})
	})
}

func (a *App) menuVisibility(c *ui.Context) {
	ui.Divider(c)
	ui.Text(c, "Visible context-menu actions").PaddingY(8).Bold()
	actions := [][2]string{
		{"copy-sha", "Copy Commit Hash to Clipboard"}, {"copy-message", "Copy Commit Subject to Clipboard"}, {"checkout", "Checkout"},
		{"create-branch", "Create branch"}, {"create-tag", "Create tag"}, {"cherry-pick", "Cherry pick"}, {"revert", "Revert"}, {"reset", "Reset"},
		{"branch-checkout", "Checkout Branch"}, {"branch-select", "Select / Unselect in Branches Dropdown"}, {"branch-copy", "Copy Branch Name to Clipboard"}, {"branch-rename", "Rename Branch"},
		{"drop", "Drop"}, {"merge", "Merge into current branch"}, {"rebase", "Rebase current branch on this Commit"},
		{"branch-push", "Push Branch"}, {"branch-archive", "Create Archive (Branch)"},
		{"remote-branch-delete", "Delete Remote Branch"}, {"remote-branch-fetch", "Fetch into local branch"}, {"remote-branch-pull", "Pull into current branch"},
		{"tag-details", "View Details (Tag)"}, {"tag-delete", "Delete Tag"}, {"tag-push", "Push Tag"}, {"tag-copy", "Copy Tag Name to Clipboard"}, {"tag-archive", "Create Archive (Tag)"},
		{"branch-delete", "Delete branch"}, {"branch-merge", "Merge branch"}, {"branch-rebase", "Rebase onto branch"},
		{"stash-apply", "Apply stash"}, {"stash-pop", "Pop stash"}, {"stash-drop", "Drop stash"}, {"create-pr", "Create pull request"}, {"open-issue", "Open issue"},
	}
	for _, group := range []string{"Commits", "Branches", "Remote branches", "Tags", "Stashes", "Links"} {
		ui.Column(c).FillWidth().PaddingY(10).BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border.Alpha(.5)).Children(func() {
			ui.Text(c, group).Bold().FontSize(12).Padding(0, 0, 6, 0)
			for _, action := range actions {
				if menuActionGroup(action[0]) != group {
					continue
				}
				key, label := action[0], action[1]
				visible := true
				for _, hidden := range a.config.HiddenActions {
					if hidden == key {
						visible = false
						break
					}
				}
				ui.Row(c).FillWidth().Height(32).AlignItems(ui.Center).Children(func() {
					if ui.Checkbox(c.Key(key), &visible, label).Changed() {
						kept := []string{}
						for _, hidden := range a.config.HiddenActions {
							if hidden != key {
								kept = append(kept, hidden)
							}
						}
						if !visible {
							kept = append(kept, key)
						}
						a.config.HiddenActions = kept
						a.saveConfig(false)
					}
				})
			}
		})
	}
	if ui.Button(c, "Show all actions").Margin(12, 0, 0, 0).Clicked() {
		a.config.HiddenActions = nil
		a.saveConfig(false)
	}
}

func menuActionGroup(key string) string {
	switch {
	case strings.HasPrefix(key, "remote-"):
		return "Remote branches"
	case strings.HasPrefix(key, "branch-"):
		return "Branches"
	case strings.HasPrefix(key, "tag-"):
		return "Tags"
	case strings.HasPrefix(key, "stash-"):
		return "Stashes"
	case key == "create-pr" || key == "open-issue":
		return "Links"
	default:
		return "Commits"
	}
}

func (a *App) themeSetting(c *ui.Context) {
	ui.Column(c).FillWidth().PaddingY(8).Gap(6).Children(func() {
		ui.Text(c, "Color theme").Bold().FontSize(13)
		selected := a.prefs.Theme
		if selected == "" {
			selected = "MyGo Dark"
			if !a.prefs.Dark {
				selected = "MyGo Light"
			}
		}
		if ui.Select(c, &selected, themeNames()).FillWidth().Changed() {
			a.prefs.Theme = selected
			a.setTheme()
			a.persist()
		}
	})
}

func (a *App) settingRow(c *ui.Context, spec settingSpec) {
	fields := configFields(&a.config)
	field := fields[spec.Key]
	ui.Column(c.Key(spec.Key)).FillWidth().PaddingY(9).Gap(4).Children(func() {
		switch {
		case field.Kind() == reflect.Bool:
			if ui.Checkbox(c, field.Addr().Interface().(*bool), spec.Label).Changed() {
				a.saveConfig(spec.Reload)
				a.rebuildFileRows()
			}
		case spec.Key == "graph_style" || spec.Key == "repo_order":
			ui.Text(c, spec.Label).FontSize(13).Bold()
			options := []string{"rounded", "angular"}
			if spec.Key == "repo_order" {
				options = []string{"name", "path", "given"}
			}
			if ui.Select(c, field.Addr().Interface().(*string), options).FillWidth().Changed() {
				a.saveConfig(false)
				a.sortRepos(a.repo())
			}
		case spec.Key == "color_preset":
			ui.Text(c, spec.Label).FontSize(13).Bold()
			options := []string{"Default", "Categorical", "Monochrome"}
			selected := options[a.config.ColorPreset]
			if ui.Select(c, &selected, options).FillWidth().Changed() {
				for i, s := range options {
					if selected == s {
						a.config.ColorPreset = i
					}
				}
				a.saveConfig(false)
			}
		case field.Kind() == reflect.Float64:
			ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, spec.Label).Grow(1).FontSize(13)
				if ui.NumberInput(c, field.Addr().Interface().(*float64), 60, 600, 5).Width(135).Changed() {
					a.saveConfig(false)
				}
			})
		default:
			ui.Text(c, spec.Label).FontSize(13).Bold()
			if a.settingsEdits == nil {
				a.settingsEdits = map[string]string{}
			}
			value, ok := a.settingsEdits[spec.Key]
			if !ok {
				if field.Kind() == reflect.Slice {
					value = strings.Join(field.Interface().([]string), ";")
				} else {
					value = field.String()
				}
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				input := ui.TextInput(c, &value).Grow(1).Label(spec.Label)
				if input.Changed() {
					a.settingsEdits[spec.Key] = value
				}
				if ui.Button(c, "Apply").Clicked() || input.Submitted() {
					if field.Kind() == reflect.Slice {
						field.Set(reflect.ValueOf(splitConfigList(value)))
					} else {
						field.SetString(value)
					}
					delete(a.settingsEdits, spec.Key)
					a.saveConfig(spec.Reload)
				}
			})
		}
		ui.Text(c, spec.Description).FontSize(11).TextColor(c.Theme().TextMuted)
	})
}

func (a *App) fileSettings(c *ui.Context) {
	if ui.Checkbox(c, &a.prefs.InlineDetails, "Expand commit details inline").Changed() {
		a.largeDiffVisible = false
		a.persist()
	}

}

func (a *App) remoteSettings(c *ui.Context) {
	ui.Column(c).FillWidth().PaddingY(12).Gap(6).Children(func() {
		ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Children(func() {
			if ui.Checkbox(c, &a.prefs.AutoFetchOnSwitch, "Automatically fetch when switching repositories").Changed() {
				a.persist()
			}
		})
		ui.Text(c, "Update remote branches and tags in the background.").FontSize(11).TextColor(c.Theme().TextMuted).Margin(0, 0, 0, 24)
	})
	ui.Divider(c).Margin(8, 0)
	ui.Text(c, "Configured remotes").PaddingY(8).Bold()
	names := make([]string, 0, len(a.snapshot.Remotes))
	for name := range a.snapshot.Remotes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		raw := a.snapshot.Remotes[name]
		ui.Column(c.Key(name)).FillWidth().Gap(5).PaddingY(8).Children(func() {
			ui.Text(c, name).Bold()
			ui.Text(c, raw).FontSize(11).TextColor(c.Theme().TextMuted).Selectable()
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, "Fetch").Clicked() {
					args := []string{"fetch", name}
					if a.config.FetchPrune {
						args = append(args, "--prune")
					}
					if a.config.FetchPruneTags {
						args = append(args, "--prune-tags")
					}
					a.runGit("Fetching "+name+"…", args...)
				}
				if ui.Button(c, "Prune").Clicked() {
					a.runGit("Pruning "+name+"…", "remote", "prune", name)
				}
				if ui.Button(c, "Edit URL…").Clicked() {
					a.ask("Edit "+name, "Remote fetch URL.", "Save", func(value string) { a.runGit("Updating remote…", "remote", "set-url", name, value) })
					a.prompt.Input = raw
				}
				if ui.Button(c, "Remove…").Clicked() {
					a.ask("Remove "+name+"?", "Remove this remote and its local tracking references.", "Remove", func(string) { a.runGit("Removing remote…", "remote", "remove", name) })
				}
			})
		})
	}
	if ui.Button(c, "Add remote…").Disabled(a.repo() == "").Clicked() {
		a.ask("Add remote", "Enter a remote name followed by its URL.", "Add", func(value string) {
			fields := strings.Fields(value)
			if len(fields) != 2 || strings.HasPrefix(fields[0], "-") {
				a.errorText = "Enter a remote name and URL"
				return
			}
			a.runGit("Adding remote…", "remote", "add", fields[0], fields[1])
		})
	}
}

func (a *App) workspaceSettings(c *ui.Context) {
	ui.Divider(c)
	ui.Row(c).FillWidth().PaddingY(12).Gap(12).Children(func() {
		ui.Text(c, "Repository discovery depth").Grow(1)
		depth := float64(a.prefs.DiscoveryDepth)
		if ui.NumberInput(c, &depth, 0, 32, 1).Width(130).Changed() {
			a.prefs.DiscoveryDepth = int(depth)
			a.persist()
		}
	})
	if ui.Button(c, "Rescan repositories").Clicked() {
		a.discoverRoots(a.roots)
	}
	ui.Text(c, "Repository switch shortcut").PaddingY(8).Bold()
	value, ok := a.settingsEdits["shortcut"]
	if !ok {
		value = a.prefs.RepositoryShortcut
	}
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		if ui.TextInput(c, &value).Grow(1).Label("Repository switch shortcut").Changed() {
			if a.settingsEdits == nil {
				a.settingsEdits = map[string]string{}
			}
			a.settingsEdits["shortcut"] = value
		}
		if ui.Button(c, "Apply shortcut").Clicked() {
			mods, key := shortcutParts(value)
			if mods == 0 || key == ui.KeyUnknown {
				a.errorText = "Use a shortcut such as cmd-p, cmd-shift-p, or ctrl-alt-b"
				return
			}
			a.prefs.RepositoryShortcut = strings.ToLower(value)
			a.persist()
			if a.win != nil {
				a.installMenu()
			}
		}
	})
	ui.Row(c).FillWidth().Gap(8).PaddingY(12).Children(func() {
		if ui.Button(c, "Add repository…").Clicked() {
			a.openRepository(false)
		}
		if ui.Button(c, "Remove current repository").Disabled(a.repo() == "").Clicked() {
			a.removeRepository()
		}
	})
	for _, root := range a.roots {
		ui.Text(c, filepath.Clean(root)).FontSize(11).TextColor(c.Theme().TextMuted)
	}
}

func shortcutParts(value string) (ui.Modifiers, ui.Key) {
	var mods ui.Modifiers
	var key ui.Key = ui.KeyUnknown
	for _, part := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == '+' || r == '-' }) {
		switch part {
		case "cmd", "command", "super", "meta":
			mods |= ui.Cmd
		case "ctrl", "control":
			mods |= ui.Ctrl
		case "alt", "option":
			mods |= ui.Alt
		case "shift":
			mods |= ui.Shift
		default:
			key = shortcutKeys[part]
		}
	}
	return mods, key
}

var shortcutKeys = map[string]ui.Key{"p": ui.KeyP, "o": ui.KeyO, "b": ui.KeyB, "r": ui.KeyR, "f": ui.KeyF, "g": ui.KeyG, "l": ui.KeyL, "d": ui.KeyD, "t": ui.KeyT, "h": ui.KeyH, "s": ui.KeyS, "k": ui.KeyK, "j": ui.KeyJ, "n": ui.KeyN, "m": ui.KeyM, "a": ui.KeyA, "e": ui.KeyE, "q": ui.KeyQ, "w": ui.KeyW, "x": ui.KeyX, "y": ui.KeyY, "z": ui.KeyZ, "u": ui.KeyU, "i": ui.KeyI, "c": ui.KeyC, "v": ui.KeyV, "comma": ui.KeyComma, ",": ui.KeyComma}

func nativeShortcut(value string) string {
	parts := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == '+' || r == '-' })
	for i, part := range parts {
		switch part {
		case "cmd", "command", "super", "meta":
			parts[i] = "CmdOrCtrl"
		case "ctrl", "control":
			parts[i] = "Control"
		case "alt", "option":
			parts[i] = "Alt"
		case "shift":
			parts[i] = "Shift"
		default:
			parts[i] = strings.ToUpper(part)
		}
	}
	return strings.Join(parts, "+")
}
func baseRef(ref string) string {
	ref, _, _ = strings.Cut(ref, "\x1f")
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(ref, "refs/heads/"), "refs/remotes/"), "tag:")
}

var issuePattern = regexp.MustCompile(`#([0-9]+)`)

func (a *App) removeRepository() {
	repo := a.repo()
	if repo == "" {
		return
	}
	a.prefs.Excluded = append(a.prefs.Excluded, repo)
	a.repos = append(a.repos[:a.active], a.repos[a.active+1:]...)
	a.active = min(a.active, len(a.repos)-1)
	a.configRepo = ""
	a.clearDetail()
	a.persist()
	a.load(false)
}

// Use the same configured shortcut as the native menu, in compact macOS notation.
func shortcutLabel(value string) string {
	return strings.NewReplacer("CmdOrCtrl", "⌘", "Control", "⌃", "Alt", "⌥", "Shift", "⇧", "+", "").Replace(nativeShortcut(value))
}

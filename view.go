package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

const commitRowHeight float32 = 28
const commitVertexOffset = commitRowHeight/2 - 2

var laneColors = []ui.Color{ui.Hex("#0085d9"), ui.Hex("#d9008f"), ui.Hex("#00d90a"), ui.Hex("#d98500"), ui.Hex("#a300d9"), ui.Hex("#ff0000"), ui.Hex("#00d9cc"), ui.Hex("#e138e8"), ui.Hex("#85d900"), ui.Hex("#dc5b23"), ui.Hex("#6f24d6"), ui.Hex("#ffcc00")}

func (a *App) view(c *ui.Context) {
	c.SetMenuPresentation(ui.PopoverMenus)
	c.SetTheme(a.uiTheme)
	if !a.modalOpen() {
		if a.isDiffView() && !a.showSearch && c.Shortcut(0, ui.KeyEscape) {
			a.returnToGraph()
		}
		if c.Shortcut(ui.Cmd, ui.KeyW) {
			a.closeViewOrWindow()
		}
		if c.Shortcut(ui.Cmd, ui.KeyO) {
			a.openRepository(false)
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyO) {
			a.openRepository(true)
		}
		if c.Shortcut(shortcutParts(a.prefs.RepositoryShortcut)) {
			a.openRepositoryPicker()
		}
		commandMods, commandKey := ui.Cmd|ui.Shift, ui.KeyP
		if mods, key := shortcutParts(a.prefs.RepositoryShortcut); mods == commandMods && key == commandKey {
			commandMods = ui.Cmd
		}
		if c.Shortcut(commandMods, commandKey) {
			a.showCommands = true
		}
		if c.Shortcut(ui.Cmd, ui.KeyR) {
			a.load(true)
		}
		if c.Shortcut(ui.Cmd, ui.KeyF) {
			a.openSearch()
		}
		if c.Shortcut(ui.Cmd, ui.KeyB) {
			a.showBranches = true
		}
		if c.Shortcut(ui.Cmd, ui.KeyComma) {
			a.showSettings = true
		}
		if c.Shortcut(ui.Cmd, ui.KeyD) {
			a.detailVisible = !a.detailVisible
			if !a.detailVisible {
				a.detailMaximized = false
			}
			a.config.ShowDetail = a.detailVisible
			a.saveConfig(false)
		}
		if c.Shortcut(ui.Cmd, ui.KeyL) {
			a.showStashes = true
		}
		if c.Shortcut(ui.Cmd, ui.KeyT) {
			names := themeNames()
			index := 0
			for i, n := range names {
				if n == a.prefs.Theme {
					index = i
					break
				}
			}
			a.prefs.Theme = names[(index+1)%len(names)]
			a.setTheme()
			a.persist()
		}
		if c.Shortcut(ui.Cmd, ui.KeyG) {
			a.moveFindMatch(1)
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyG) {
			a.moveFindMatch(-1)
		}
	}
	if a.searchDiff && !a.isDiffView() {
		a.restoreGraphSearch()
	}
	if a.showSearch && a.searchDiff {
		a.rebuildDiffSearch()
	}
	if a.query != a.lastQuery {
		a.rebuildVisible()
	}
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		a.toolbar(c)
		if a.repo() == "" {
			a.welcome(c)
		} else if a.prefs.InlineDetails && a.largeDiffVisible && a.detailVisible && a.revision != "" {
			a.largeDiff(c)
		} else if a.detailVisible && a.revision != "" && (!a.prefs.InlineDetails || strings.HasPrefix(a.revision, "stash@{")) {
			if a.detailMaximized {
				a.detail(c)
			} else {
				ui.SplitVerticalPercent(c, &a.detailPercent, func() { a.graph(c) }, func() { a.detail(c) }).Grow(1).FillWidth().MinHeight(0).SplitReset(40)
			}
		} else {
			a.graph(c)
		}
		a.footer(c)
	})
	a.searchBar(c)
	a.repositoryPicker(c)
	a.branchPicker(c)
	a.stashPicker(c)
	a.settings(c)
	a.commands(c)
	a.inputDialog(c)
	a.syncDialog(c)
	a.actionDialog(c)
	if a.errorText != "" {
		open := true
		ui.Modal(c.Key("error-modal"), &open, func() {
			ui.Column(c).Width(580).Gap(14).Padding(22).Children(func() {
				ui.Text(c, "Git operation failed").FontSize(17).Bold().TextColor(t.Danger)
				ui.Text(c, a.errorText).MaxWidth(530).Selectable()
				if ui.Button(c, "Close error").Clicked() {
					a.errorText = ""
				}
			})
		})
		if !open {
			a.errorText = ""
		}
	}
}

func (a *App) toolbar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Height(54).Shrink(0).PaddingX(14).Gap(8).AlignItems(ui.Center).Background(t.Surface).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		name := "Open a repository"
		if a.repo() != "" {
			name = a.repoLabel(a.repo())
		}
		a.repositoryAnchor = selectorButton(c.Key("repository-picker"), name, "Switch repository", 215, folderIcon, shortcutLabel(a.prefs.RepositoryShortcut)).Tooltip(a.repo() + "\nSwitch repository · " + shortcutLabel(a.prefs.RepositoryShortcut))
		if a.repositoryAnchor.Clicked() {
			if a.showRepos {
				a.showRepos = false
			} else {
				a.openRepositoryPicker()
			}
		}
		if a.repo() != "" {
			label := "All branches"
			if a.filterRef != "" {
				label = strings.TrimPrefix(strings.TrimPrefix(a.filterRef, "refs/heads/"), "refs/remotes/")
				if refs := strings.Split(a.filterRef, "\x1e"); len(refs) > 1 {
					label = fmt.Sprintf("%d branches", len(refs))
				}
			}
			a.branchesAnchor = selectorButton(c.Key("branch-picker"), label, "Select branches", 185, branchIcon, "⌘B").Tooltip(label + "\nSelect branches · ⌘B")
			if a.branchesAnchor.Clicked() {
				a.showBranches = !a.showBranches
			}
			if ui.Checkbox(c, &a.config.Remotes, "Remotes").Changed() {
				a.saveConfig(true)
			}
		}
		ui.Box(c).Grow(1)
		if a.loading {
			ui.Spinner(c).Size(15, 15)
		}
		if toolbarKeyButton(c, "Find", "⌘F", nil).Tooltip("Find · ⌘F").Clicked() {
			if a.showSearch {
				a.closeSearch()
			} else {
				a.openSearch()
			}
		}
		ui.MenuButton(c, "Actions", func(m *ui.Menu) { a.repoMenu(m) })
		if ui.Button(c, "").Label("Fetch remotes").Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() { ui.Text(c, "Fetch").FontSize(13); ui.Icon(c, downIcon).Size(14, 14) })
		}).Disabled(a.repo() == "" || a.busy[a.repo()] != "").Clicked() {
			a.fetch()
		}
		if toolbarKeyButton(c, "Refresh graph", "⌘R", refreshIcon).Tooltip("Refresh · ⌘R").Disabled(a.repo() == "").Clicked() {
			a.load(true)
		}
		if toolbarKeyButton(c, "Settings", "⌘,", settingsIcon).Tooltip("Settings · ⌘,").Clicked() {
			a.showSettings = true
		}
	})
}

func (a *App) closeSearch() {
	if a.searchDiff {
		a.diffQuery = ""
		a.showSearch = false
		a.searchFocused = false
		a.rebuildDiffSearch()
		return
	}
	a.showSearch = false
	a.searchFocused = false
	a.query = ""
	a.rebuildVisible()
}

func (a *App) searchBar(c *ui.Context) {
	if !a.showSearch {
		a.searchFocused = false
		return
	}
	t := c.Theme()
	query := &a.query
	placeholder, inputLabel := "Message, author, SHA or reference…", "Search commits"
	if a.searchDiff {
		query = &a.diffQuery
		placeholder, inputLabel = "Find in this file…", "Search file diff"
	}
	ui.Row(c.Key("floating-search")).Absolute().Top(62).Right(14).Width(660).Height(48).PaddingX(10).Gap(10).AlignItems(ui.Center).Background(t.Surface).Border(1, t.Border).Radius(8).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, .3)).Children(func() {
		input := ui.TextInput(c, query).Grow(1).Basis(0).MinWidth(0).Placeholder(placeholder).Label(inputLabel)
		if !a.searchFocused {
			input.Focus()
			a.searchFocused = true
		}
		if input.Changed() {
			if a.searchDiff {
				a.rebuildDiffSearch()
			} else {
				a.rebuildVisible()
			}
		}
		input.OnTextKey(func(key ui.TextKey) bool {
			if key.Key == ui.KeyEnter && (key.Mods == 0 || key.Mods == ui.Shift) {
				delta := 1
				if key.Mods == ui.Shift {
					delta = -1
				}
				a.moveFindMatch(delta)
				return true
			}
			if key.Key == ui.KeyEscape && key.Mods == 0 {
				a.closeSearch()
				return true
			}
			return false
		})
		label := "Type to search"
		color := t.TextMuted
		count, cursor := len(a.searchMatches), a.matchCursor
		if a.searchDiff {
			count, cursor = len(a.diffMatches), a.diffMatchCursor
		}
		if *query != "" {
			label = fmt.Sprintf("%d matches", count)
			if count == 0 {
				color = t.Danger
			} else {
				label = fmt.Sprintf("%d / %d", cursor+1, count)
			}
		}
		ui.Text(c, label).Width(100).Shrink(0).SingleLine().TextAlign(ui.Center).FontSize(12).TextColor(color)
		ui.Text(c, "⌘G next").Width(50).Shrink(0).SingleLine().FontSize(11).TextColor(t.TextMuted)
		if iconButton(c, "Close search", closeIcon).Clicked() {
			a.closeSearch()
		}
	})
}

func (a *App) graph(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		graphWidth := float32(max(a.snapshot.Lanes, 2)*18 + 22)
		graphColors := a.colors()
		angular := a.config.GraphStyle == "angular"
		ui.Row(c).FillWidth().Height(31).Shrink(0).PaddingX(10).AlignItems(ui.Center).Background(t.Background).BorderWidth(1, 0, 1, 0).BorderColor(t.Border).Children(func() {
			header := func(label string, width float32) {
				box := ui.Row(c).Height(30).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Justify(ui.Center).AlignItems(ui.Center)
				if width > 0 {
					box.Width(width).Shrink(0)
				} else {
					box.Grow(1).MinWidth(0)
				}
				box.Children(func() { ui.Text(c, label).FontSize(13).Bold() })
			}
			header("Graph", graphWidth)
			header("Description", 0)
			if a.config.ColumnsDate {
				header("Date", float32(a.config.DateWidth))
			}
			if a.config.ColumnsAuthor {
				header("Author", float32(a.config.AuthorWidth))
			}
			if a.config.ColumnsCommit {
				header("Commit", float32(a.config.CommitWidth))
			}
		})
		if a.config.ShowUncommitted && len(a.snapshot.Files) > 0 {
			ui.Row(c).FillWidth().Height(commitRowHeight).Shrink(0).PaddingX(10).AlignItems(ui.Center).OnClick(func() { a.selectWorktree() }).Children(func() {
				ui.Box(c).Width(graphWidth).Height(commitRowHeight).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
					x := r.X + 12
					if len(a.snapshot.Commits) > 0 && a.snapshot.Commits[0].SHA == a.snapshot.Head {
						p.Line(x, r.Y+commitVertexOffset, x, r.Y+commitRowHeight, 2, t.TextMuted)
					}
					p.Fill(ui.Rect{X: x - 5, Y: r.Y + commitVertexOffset - 5, W: 10, H: 10}, t.TextMuted, 5)
					p.Fill(ui.Rect{X: x - 3, Y: r.Y + commitVertexOffset - 3, W: 6, H: 6}, t.Background, 3)
				})
				ui.Textf(c, "Uncommitted Changes (%d)", len(a.snapshot.Files)).FontSize(13).Bold()
			})
		}
		if a.prefs.InlineDetails && a.detailVisible && a.revision == "worktree" {
			ui.Box(c).FillWidth().Height(320).Shrink(0).Children(func() { a.inlineCommitDetails(c) })
		}

		if len(a.visible) == 0 {
			ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
				if a.loading {
					ui.Spinner(c)
					ui.Text(c, "Loading repository…").TextColor(t.TextMuted)
				} else {
					label := "No commits to display"
					if a.query != "" {
						label = "No matching commits"
					}
					ui.Text(c, label).FontSize(17).Bold()
					ui.Text(c, "Try another search or show all branches.").TextColor(t.TextMuted)
					if ui.Button(c, "Show all commits").Clicked() {
						a.query = ""
						a.filterRef = ""
						a.load(false)
					}
				}
			})
		} else {
			visible, commits := a.visible, a.snapshot.Commits
			a.list.Selected = &a.listSelected
			a.list.Key = func(i int) any {
				if i >= 0 && i < len(visible) {
					return commits[visible[i]].SHA
				}
				return i
			}
			a.list.Label = func(i int) string {
				commit := commits[visible[i]]
				return commit.Subject + " · " + shortSHA(commit.SHA)
			}
			ui.Themed(c, a.listTheme).Grow(1).FillWidth().MinHeight(0).Children(func() {
				rowActivated := false
				expandedRevision := ""
				if a.prefs.InlineDetails && a.detailVisible {
					expandedRevision = a.revision
				}
				list := ui.List(c.Key("commit-list"), &a.list, len(visible), func(i int) {
					index := visible[i]
					commit := commits[index]
					expanded := expandedRevision == commit.SHA
					card := ui.Themed(c, a.uiTheme).FillWidth().Background(ui.Transparent)
					if expanded {
						card.Background(t.Surface)
					}
					card.Children(func() {
						row := ui.Row(c).FillWidth().MinHeight(commitRowHeight).PaddingX(10).Gap(0).AlignItems(ui.Start)
						if index == a.selected {
							row.Background(a.listTheme.Accent)
						}
						if a.searchMatched[index] {
							color := t.Warning
							if commit.SHA == a.searchCurrent {
								color = t.Accent
							}
							if index != a.selected {
								row.Background(color.Alpha(.09))
							}
							row.DrawOver(func(p *ui.Painter, r ui.Rect) { p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 2, H: r.H}, color, 0) })
						}
						row.Children(func() {
							ui.Box(c).Width(graphWidth).Shrink(0).AlignSelf(ui.Stretch).Draw(func(p *ui.Painter, r ui.Rect) {
								paintCommit(p, r, commit, angular, graphColors)
							})
							description := ui.Row(c).Grow(1).MinWidth(0).Padding(2, 4, 6, 4).Gap(4).Wrap().AlignItems(ui.Center)
							description.Children(func() {
								if commit.SHA == a.snapshot.Head && a.snapshot.Head != "" {
									color := graphColors[commit.Lane%len(graphColors)]
									ui.Box(c).Size(10, 10).Shrink(0).Label("HEAD commit").Draw(func(p *ui.Painter, r ui.Rect) {
										p.Fill(r, color, 5)
										p.Fill(ui.Rect{X: r.X + 2, Y: r.Y + 2, W: r.W - 4, H: r.H - 4}, t.Background, 3)
									})
								}
								for _, ref := range commit.Refs {
									a.referencePill(c, ref, commit, description.Bounds().W-8)
								}
								ui.Row(c).Grow(1).MinWidth(100).Height(20).AlignItems(ui.Center).Children(func() {
									subject := a.graphSearchText(c, a.messageText(commit.Subject)).Grow(1).MinWidth(0).SingleLine().FontSize(13)
									if a.config.DimNonHeadCommits && !a.snapshot.Ancestors[commit.SHA] {
										subject.TextColor(t.TextMuted)
									}
								})
							})
							if a.config.ColumnsDate {
								commitColumnText(c, a.dateText(commit), float32(a.config.DateWidth), false)
							}
							if a.config.ColumnsAuthor {
								commitColumnText(c, commit.Author, float32(a.config.AuthorWidth), false)
							}
							if a.config.ColumnsCommit {
								commitColumnText(c, shortSHA(commit.SHA), float32(a.config.CommitWidth), true)
							}
						})
						if a.config.ShowCommitTooltip {
							row.Tooltip(commit.Subject + "\n" + commit.Author + " · " + commit.Time.Format("2006-01-02 15:04") + "\n" + commit.SHA + "\n" + strings.Join(commit.Refs, ", "))
						}
						row.ContextMenu(func(m *ui.Menu) { a.commitMenu(m, commit) })
						row.OnClick(func() { rowActivated = true; a.toggleCommit(index) })
						if expanded {
							ui.Row(c).FillWidth().Height(320).PaddingX(10).AlignItems(ui.Stretch).Children(func() {
								ui.Box(c).Width(graphWidth).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) { paintContinuation(p, r, commit, graphColors) })
								ui.Box(c).Grow(1).MinWidth(0).Children(func() { a.inlineCommitDetails(c) })
							})
						}
					})
				})
				list.Fill().MinHeight(0)
				if a.focusCommitList {
					list.Focus()
					a.focusCommitList = false
				}
				if (list.Changed() || list.Submitted()) && !rowActivated && a.listSelected >= 0 && a.listSelected < len(visible) {
					a.openCommit(visible[a.listSelected])
				}
			})
		}
		if a.snapshot.More {
			ui.Row(c).FillWidth().Height(32).Justify(ui.Center).Children(func() {
				if ui.Button(c, "Load 500 more commits").Disabled(a.loading).Clicked() {
					a.loadMore()
				}
			})
		}
	})
}

func commitColumnText(c *ui.Context, value string, width float32, mono bool) {
	ui.Row(c).Width(width).Shrink(0).AlignSelf(ui.Stretch).Padding(0, 0, 4, 0).AlignItems(ui.Center).Children(func() {
		text := ui.Text(c, value).FillWidth().MinWidth(0).SingleLine().FontSize(13).TextColor(c.Theme().Text).TextAlign(ui.Center)
		if mono {
			text.Font("Menlo")
		}
	})
}

func paintCommit(p *ui.Painter, r ui.Rect, c Commit, angular bool, laneColors []ui.Color) {
	x := func(l int) float32 { return r.X + 12 + float32(l)*18 }
	y := r.Y + commitVertexOffset
	color := laneColors[c.Lane%len(laneColors)]
	for _, l := range c.Through {
		p.Line(x(l), r.Y, x(l), r.Y+r.H, 2, laneColors[l%len(laneColors)])
	}
	if c.Top {
		p.Line(x(c.Lane), r.Y, x(c.Lane), y, 2, color)
	}
	if c.Bottom {
		p.Line(x(c.Lane), y, x(c.Lane), r.Y+r.H, 2, color)
	}
	for _, l := range c.Incoming {
		path := new(ui.Path).MoveTo(x(l), r.Y)
		if angular {
			path.LineTo(x(c.Lane), y)
		} else {
			path.QuadTo(x(l), y, x(c.Lane), y)
		}
		p.StrokePath(path, 2, laneColors[l%len(laneColors)])
	}
	for _, l := range c.Outgoing {
		path := new(ui.Path).MoveTo(x(c.Lane), y)
		if angular {
			path.LineTo(x(l), r.Y+commitRowHeight)
		} else {
			path.QuadTo(x(l), y, x(l), r.Y+commitRowHeight)
		}
		p.StrokePath(path, 2, laneColors[l%len(laneColors)])
		if r.H > commitRowHeight {
			p.Line(x(l), r.Y+commitRowHeight, x(l), r.Y+r.H, 2, laneColors[l%len(laneColors)])
		}
	}
	p.Fill(ui.Rect{X: x(c.Lane) - 4, Y: y - 4, W: 8, H: 8}, color, 4)
}

func (a *App) detail(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().MinHeight(0).Background(t.Surface).Label("Commit details").Children(func() {
		ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).Children(func() {
			title := "Uncommitted changes"
			if a.selectedCommit != nil {
				title = a.messageText(a.selectedCommit.Subject)
			}
			ui.Text(c, title).Grow(1).MinWidth(0).SingleLine().Bold().FontSize(13).Tooltip(title)
			if a.compare != "" {
				ui.Text(c, "vs "+shortSHA(a.compare)).FontSize(11).TextColor(t.Accent)
				if ui.Button(c, "End comparison").Height(24).FontSize(11).Clicked() {
					a.compare = ""
					a.loadFiles()
				}
			}
			label, icon := "Expand details", expandIcon
			if a.detailMaximized {
				label, icon = "Restore graph", collapseIcon
			}
			if iconButton(c, label, icon).Height(24).Width(24).Tooltip(label).Clicked() {
				a.detailMaximized = !a.detailMaximized
			}
			if iconButton(c, "Close details", closeIcon).Height(24).Width(24).Clicked() {
				a.detailMaximized = false
				a.config.ShowDetail = false
				a.saveConfig(false)
				a.focusCommitList = true
			}
		})
		if commit := a.selectedCommit; commit != nil {
			ui.Row(c).FillWidth().Height(26).Shrink(0).PaddingX(12).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, shortSHA(commit.SHA)).Font("Menlo").FontSize(11).TextColor(t.Accent)
				if commit.Author != "" {
					ui.Text(c, commit.Author).MinWidth(0).SingleLine().FontSize(11).TextColor(t.TextMuted).Tooltip(commit.Author + " <" + commit.Email + ">")
				}
				if !commit.Time.IsZero() {
					ui.Text(c, commit.Time.Format("2006-01-02 15:04 MST")).FontSize(11).TextColor(t.TextMuted)
				}
				ui.Box(c).Grow(1)
				if commit.Body != "" {
					label := "Show more"
					if a.messageExpanded {
						label = "Show less"
					}
					if ui.Button(c, label).Height(22).PaddingX(6).FontSize(11).Label("Toggle commit message").Clicked() {
						a.messageExpanded = !a.messageExpanded
					}
				}
				if ui.Button(c, "Copy SHA").Height(22).PaddingX(6).FontSize(11).Clicked() {
					a.copyText(commit.SHA)
				}
			})
			if commit.Body != "" {
				if a.messageExpanded {
					ui.Scroll(c).FillWidth().MaxHeight(110).Children(func() { a.messageElement(c, commit.Body).FillWidth().Padding(2, 12, 4, 12).FontSize(12).Selectable() })
				} else {
					preview := compactMessage(commit.Body)
					ui.Box(c).FillWidth().Height(36).Shrink(0).PaddingX(12).Clip().Children(func() { a.messageElement(c, preview).FillWidth().MaxLines(2).FontSize(12).Selectable() })
				}
			}
		}
		a.filesToolbar(c)
		ui.Split(c, &a.fileWidth, func() {
			ui.Column(c).Fill().MinHeight(0).Children(func() {
				if a.detailLoading {
					ui.Spinner(c)
				}
				a.fileListView(c)
			})
		}, func() { a.diff(c) }).Grow(1).FillWidth().MinHeight(0).SplitReset(240)
	})
}

func (a *App) diff(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().MinHeight(0).Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Height(28).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			path := "Diff"
			if a.selectedFile >= 0 && a.selectedFile < len(a.files) {
				path = a.files[a.selectedFile].Path
			}
			ui.Text(c, path).Grow(1).MinWidth(0).SingleLine().FontSize(11).Tooltip(path).Label("Diff file path")
			if a.imageDiff != nil {
				label := "Show text diff"
				if a.imageDiffText {
					label = "Show image diff"
				}
				if ui.Button(c, label).Height(22).FontSize(10).PaddingX(5).Clicked() {
					a.imageDiffText = !a.imageDiffText
					if !a.imageDiffText && a.searchDiff {
						a.showSearch = false
						a.searchFocused = false
					}
				}
				if !a.imageDiffText {
					return
				}
			}
			label := "Show headers"
			if a.diffMetadata {
				label = "Hide headers"
			}
			if ui.Button(c, label).Height(22).FontSize(10).PaddingX(5).Clicked() {
				a.diffMetadata = !a.diffMetadata
				a.diffScroll = ui.ScrollState{}
				a.diffScrollTarget = nil
			}
		})
		if a.diffLoading {
			ui.Column(c).Fill().Center().Gap(10).Children(func() { ui.Spinner(c); ui.Text(c, "Loading diff…").TextColor(t.TextMuted) })
			return
		}
		if a.imageDiff != nil && !a.imageDiffText {
			a.imageDiffView(c)
			return
		}
		if len(a.diffLines) == 0 {
			ui.Column(c).Fill().Center().Gap(6).Children(func() {
				ui.Text(c, "Select a file to inspect its diff").TextColor(t.TextMuted)
				ui.Text(c, "Changes load in the background").FontSize(11).TextColor(t.TextMuted)
			})
			return
		}
		lines := a.displayedDiffLines()
		if a.showSearch && a.searchDiff {
			a.rebuildDiffSearch()
		}
		// One selection scope keeps drag, keyboard and Copy working across lines.
		// All paragraphs remain available while dragging beyond the viewport.
		selectionKey := fmt.Sprintf("diff-text:%s:%s:%d:%t", a.repo(), a.revision, a.selectedFile, a.diffMetadata)
		ui.Scroll(c.Key(selectionKey)).Fill().TrackScroll(&a.diffScroll).Selectable().Label("Diff viewport").Children(func() {
			for i, line := range lines {
				color, bg := t.Text, ui.Color{}
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					color = t.Success
					bg = t.Success.Alpha(.07)
				}
				if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					color = t.Danger
					bg = t.Danger.Alpha(.07)
				}
				if strings.HasPrefix(line, "@@") {
					color = t.Accent
					bg = t.Accent.Alpha(.08)
				}
				row := ui.Row(c.Key(i)).FillWidth().MinHeight(20).Shrink(0).PaddingX(12).Gap(12).Background(bg)
				if a.diffScrollTarget != nil && *a.diffScrollTarget == i {
					row.ScrollIntoView()
				}
				row.Children(func() {
					ui.Textf(c, "%d", i+1).Width(36).Font("Menlo").FontSize(11).TextColor(t.TextMuted).Unselectable()
					if spans := a.diffSearchSpans(i, line); len(spans) > 0 {
						ui.RichText(c, spans...).Grow(1).MinWidth(0).Font("Menlo").FontSize(12).TextColor(color)
					} else {
						ui.Text(c, line).Grow(1).MinWidth(0).Font("Menlo").FontSize(12).TextColor(color)
					}
				})
			}
		})
		a.diffScrollTarget = nil
	})
}

func (a *App) footer(c *ui.Context) {
	t := c.Theme()
	tracking := a.snapshot.Tracking
	ui.Row(c).FillWidth().Height(28).Shrink(0).PaddingX(14).Gap(16).AlignItems(ui.Center).Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		if tracking.Branch != "" {
			ui.Icon(c, branchIcon).Size(13, 13).TextColor(t.TextMuted)
			text := tracking.Branch
			if tracking.Upstream != "" {
				text += "  ·  " + tracking.Upstream
				if tracking.Gone {
					text += "  ·  Unavailable"
				} else if tracking.Ahead == 0 && tracking.Behind == 0 {
					if a.remoteState().fresh(time.Now()) && a.remoteState().Branches+a.remoteState().Tags == 0 {
						text += "  ·  Up to date"
					} else {
						text += "  ·  Local refs match"
					}
				} else {
					text += fmt.Sprintf("  ↑%d ↓%d", tracking.Ahead, tracking.Behind)
				}
			} else if !tracking.Detached {
				text += "  ·  No upstream"
			}
			ui.Text(c, text).MaxWidth(440).SingleLine().FontSize(11).Tooltip("↑ commits ahead · ↓ commits behind\nCompared with locally fetched remote references; Fetch updates them.")
		}
		status := a.busy[a.repo()]
		if status != "" {
			ui.Spinner(c).Size(12, 12)
			ui.Text(c, status).Grow(1).MinWidth(0).SingleLine().FontSize(11).TextColor(t.Accent).Tooltip(status)
		} else if failure := a.autoFetchErrors[a.repo()]; failure != "" {
			ui.Text(c, "Fetch failed · Showing local history").Grow(1).MinWidth(0).SingleLine().FontSize(11).TextColor(t.Warning).Tooltip(failure)
		} else {
			text := "Select a commit to inspect changes"
			if a.loading {
				text = "Loading repository…"
			} else if a.selectedCommit != nil {
				text = shortSHA(a.selectedCommit.SHA)
			}
			ui.Text(c, text).Grow(1).MinWidth(0).SingleLine().FontSize(11).TextColor(t.TextMuted)
		}
		a.remoteStatusView(c)
		ui.Textf(c, "%d commits", len(a.snapshot.Commits)).Shrink(0).FontSize(11).TextColor(t.TextMuted)
	})
}

func (a *App) welcome(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).FillWidth().Center().Gap(14).Children(func() {
		ui.Text(c, "gitviz").FontSize(36).Bold()
		ui.Text(c, "Your repositories. One clear graph.").FontSize(15).TextColor(t.TextMuted)
		ui.Row(c).Gap(10).Children(func() {
			if ui.PrimaryButton(c, "Open repository…").Clicked() {
				a.openRepository(false)
			}
			if ui.Button(c, "Open workspace…").Clicked() {
				a.openRepository(true)
			}
		})
		if a.loading {
			ui.Spinner(c)
			ui.Text(c, "Discovering repositories…").TextColor(t.TextMuted)
		}
		ui.Text(c, "⌘O open  ·  ⌘P switch repository  ·  ⌘F find commits").FontSize(12).TextColor(t.TextMuted)
	})
}

func (a *App) stashPicker(c *ui.Context) {
	ui.Modal(c.Key("stashes"), &a.showStashes, func() {
		ui.Column(c).Width(620).Gap(12).Padding(18).Children(func() {
			ui.Text(c, "Stashes").FontSize(18).Bold()
			if len(a.snapshot.Stashes) == 0 {
				ui.Text(c, "No stashed changes").TextColor(c.Theme().TextMuted)
			}
			ui.Scroll(c).FillWidth().MaxHeight(360).Children(func() {
				for _, stash := range a.snapshot.Stashes {
					ui.Row(c.Key(stash.Ref)).FillWidth().Gap(8).Children(func() {
						if ui.Button(c, stash.Ref+"  "+stash.Message).Grow(1).Clicked() {
							a.selectStash(stash)
						}
						ui.MenuButton(c, "…", func(m *ui.Menu) {
							if a.action(m, "stash-apply", "Apply") {
								a.runGit("Applying stash…", "stash", "apply", stash.Ref)
							}
							if a.action(m, "stash-pop", "Pop…") {
								a.ask("Pop "+stash.Ref+"?", "Apply the stash and remove it if successful.", "Pop", func(string) { a.runGit("Popping stash…", "stash", "pop", stash.Ref) })
							}
							if a.action(m, "stash-drop", "Drop…") {
								a.ask("Drop "+stash.Ref+"?", "Type drop to permanently remove this stash.", "Drop", func(v string) {
									if v != "drop" {
										a.errorText = "Type drop to confirm"
										return
									}
									a.runGit("Dropping stash…", "stash", "drop", stash.Ref)
								})
							}
						})
					})
				}
			})
			if ui.PrimaryButton(c, "Stash working changes…").Clicked() {
				a.ask("Stash changes", "Includes untracked files. Enter an optional message.", "Stash", func(message string) { a.runGit("Stashing changes…", "stash", "push", "-u", "-m", message) })
			}
		})
	})
}

func (a *App) commands(c *ui.Context) {
	justOpened := a.showCommands && !a.commandsWereOpen
	if justOpened {
		a.commandQuery = a.commandInitialQuery
		a.commandInitialQuery = ""
		a.commandCursor = 0
	}
	ui.DialogBase(c.Key("commands"), &a.showCommands, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4)).Justify(ui.Start).Padding(120, 0, 0, 0)
		panel.Width(560).Background(c.Theme().Surface).Radius(8).Border(1, c.Theme().Border).Clip().Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				commands := []struct {
					label string
					run   func()
				}{
					{"Open repository…", func() { a.openRepository(false) }},
					{"Open workspace…", func() { a.openRepository(true) }},
					{"Fetch all remotes", a.fetch},
					{"Pull", func() { a.pull(false) }},
					{"Pull Rebase", func() { a.pull(true) }},
					{"Push current branch", func() { a.runGit("Pushing…", "push") }},
					{"View working changes", a.selectWorktree},
					{"Show stashes", func() { a.showStashes = true }},
					{"Refresh graph", func() { a.load(true) }},
				}
				filtered := func(query string) []int {
					var out []int
					for i, cmd := range commands {
						if strings.Contains(strings.ToLower(cmd.label), strings.ToLower(strings.TrimSpace(query))) {
							out = append(out, i)
						}
					}
					return out
				}
				run := func() {
					indices := filtered(a.commandQuery)
					if len(indices) > 0 {
						a.commandCursor = max(0, min(a.commandCursor, len(indices)-1))
						a.showCommands = false
						commands[indices[a.commandCursor]].run()
					}
				}
				input := ui.TextInputBase(c, &a.commandQuery).FillWidth().Height(46).PaddingX(12).Placeholder("Search commands…").Label("Search commands")
				if justOpened {
					input.Focus()
				}
				if input.Changed() {
					a.commandCursor = 0
				}
				input.OnTextKey(func(key ui.TextKey) bool {
					if key.Mods != 0 {
						return false
					}
					a.commandQuery = key.Text()
					switch key.Key {
					case ui.KeyDown:
						a.commandCursor = min(max(0, len(filtered(a.commandQuery))-1), a.commandCursor+1)
						return true
					case ui.KeyUp:
						a.commandCursor = max(0, a.commandCursor-1)
						return true
					case ui.KeyEnter:
						run()
						return true
					}
					return false
				})
				indices := filtered(a.commandQuery)
				if len(indices) == 0 {
					ui.Text(c, "No matching commands").Padding(16).TextColor(c.Theme().TextMuted)
				}
				ui.Scroll(c).FillWidth().MaxHeight(384).Children(func() {
					for position, index := range indices {
						cmd := commands[index]
						row := ui.ButtonBase(c.Key(cmd.label)).FillWidth().Height(32).PaddingX(12).Justify(ui.Start).Label(cmd.label)
						if position == a.commandCursor {
							row.Background(a.listTheme.Accent)
						}
						row.Children(func() { ui.Text(c, cmd.label).SingleLine().FontSize(13) })
						if row.Clicked() {
							a.showCommands = false
							cmd.run()
						}
					}
				})
				ui.Text(c, "↑↓ choose · Enter run · Esc close").FontSize(11).TextColor(c.Theme().TextMuted).Padding(12)
			})
		})
	})
	a.commandsWereOpen = a.showCommands
}

func (a *App) inputDialog(c *ui.Context) {
	justOpened := a.promptOpen && !a.promptWasOpen
	a.promptWasOpen = a.promptOpen
	if a.prompt == nil {
		return
	}
	ui.Modal(c.Key("input-dialog"), &a.promptOpen, func() {
		p := a.prompt
		ui.Column(c).Width(520).Gap(14).Padding(22).Children(func() {
			ui.Text(c, p.Title).FontSize(17).Bold()
			ui.Text(c, p.Description).TextColor(c.Theme().TextMuted)
			input := ui.TextInput(c, &p.Input).FillWidth().Label("Confirmation or value")
			if justOpened {
				input.Focus()
			}
			ui.Row(c).Gap(10).Justify(ui.End).FillWidth().Children(func() {
				if ui.Button(c, "Cancel").Clicked() {
					a.promptOpen = false
				}
				if ui.PrimaryButton(c, p.Confirm).Clicked() || input.Submitted() {
					a.promptOpen = false
					a.prompt = nil
					p.Run(p.Input)
				}
			})
		})
	})
	if !a.promptOpen {
		a.prompt = nil
	}
}

func (a *App) syncDialog(c *ui.Context) {
	if a.syncPrompt == nil {
		return
	}
	ui.Modal(c.Key("checkout-sync"), &a.syncOpen, func() {
		t := a.syncPrompt.Tracking
		ui.Column(c).Width(550).Gap(14).Padding(22).Children(func() {
			ui.Text(c, "Update "+t.Branch+"?").FontSize(18).Bold()
			ui.Textf(c, "Checked out %s. Compared with %s, this branch is %d commits ahead and %d behind.", t.Branch, t.Upstream, t.Ahead, t.Behind)
			ui.Text(c, "Pull merges upstream changes. Pull Rebase replays local commits on top of upstream.").TextColor(c.Theme().TextMuted)
			ui.Row(c).FillWidth().Gap(10).Justify(ui.End).Children(func() {
				if ui.Button(c, "Not now").Clicked() {
					a.syncOpen = false
				}
				if ui.Button(c, "Pull").Clicked() {
					a.pull(false)
				}
				if ui.PrimaryButton(c, "Pull Rebase").Clicked() {
					a.pull(true)
				}
			})
		})
	})
	if !a.syncOpen {
		a.syncPrompt = nil
	}
}

func (a *App) repoMenu(m *ui.Menu) {
	if m.Item("Open repository…").Chosen() {
		a.openRepository(false)
	}
	if m.Item("Open workspace…").Chosen() {
		a.openRepository(true)
	}
	if m.Item("Save workspace…").Chosen() {
		a.saveWorkspace()
	}
	m.Separator()
	if m.Item("Pull").Disabled(a.repo() == "").Chosen() {
		a.pull(false)
	}
	if m.Item("Pull Rebase").Disabled(a.repo() == "").Chosen() {
		a.pull(true)
	}
	if m.Item("Push").Disabled(a.repo() == "").Chosen() {
		a.runGit("Pushing…", "push")
	}
	if m.Item("Stashes…").Chosen() {
		a.showStashes = true
	}
	if m.Item("Create branch…").Disabled(a.repo() == "").Chosen() {
		a.ask("Create branch", "Enter a branch name. It will start at HEAD.", "Create", func(name string) {
			if strings.TrimSpace(name) == "" {
				a.errorText = "Enter a branch name"
				return
			}
			a.runGit("Creating branch…", "branch", "--", name)
		})
	}
	if a.action(m, "create-pr", "Create pull request") {
		a.openURL(a.pullRequestLink())
	}
	if m.Item("Show working changes").Chosen() {
		a.selectWorktree()
	}
}

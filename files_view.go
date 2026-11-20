package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) filesToolbar(c *ui.Context) {
	ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border).Children(func() {
		label := "changed files"
		if len(a.files) == 1 {
			label = "changed file"
		}
		ui.Textf(c, "%d %s", len(a.files), label).FontSize(11).TextColor(c.Theme().TextMuted)
		if ui.Button(c, "Tree / List").Height(24).PaddingX(6).FontSize(11).Clicked() {
			a.config.FileTree = !a.config.FileTree
			a.saveConfig(false)
			a.rebuildFileRows()
		}
	})
}

func (a *App) fileListView(c *ui.Context) {
	ui.Themed(c, a.listTheme).Grow(1).FillWidth().MinHeight(0).Background(ui.Transparent).Children(func() { a.fileListContent(c) })
}

func (a *App) fileListContent(c *ui.Context) {
	t := c.Theme()
	a.fileList.Selected = &a.fileRowSelected
	a.fileList.Key = func(i int) any { return a.fileRows[i].Path }
	a.fileList.Label = func(i int) string { return a.fileRows[i].Path + " · changed file" }
	list := ui.List(c.Key("file-list"), &a.fileList, len(a.fileRows), func(i int) {
		row := a.fileRows[i]
		ui.Themed(c, a.uiTheme).FillWidth().Background(ui.Transparent).Children(func() {
			if row.Dir {
				icon := chevronIcon
				if a.collapsed[row.Path] {
					icon = chevronRightIcon
				}
				ui.Row(c).FillWidth().Height(27).PaddingX(12 + float32(row.Depth)*12).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icon).Size(14, 14).Shrink(0).TextColor(t.TextMuted)
					ui.Text(c, row.Name+"/").FontSize(12).TextColor(t.TextMuted)
				})
				return
			}
			f := a.files[row.File]
			color := t.TextMuted
			if strings.ContainsAny(f.Status, "A?") {
				color = t.Success
			} else if strings.Contains(f.Status, "D") {
				color = t.Danger
			}
			inline := a.prefs.InlineDetails && !a.largeDiffVisible
			if inline && strings.HasPrefix(f.Status, "M") {
				color = t.Warning
			}
			ui.Row(c).FillWidth().Height(27).PaddingX(12 + float32(row.Depth)*12).Gap(8).AlignItems(ui.Center).Label("File diff: " + f.Path).Children(func() {
				if inline {
					ui.Icon(c, fileIcon).Size(14, 14).Shrink(0).TextColor(t.TextMuted)
				} else {
					ui.Text(c, strings.TrimSpace(f.Status)).Width(20).Font("Menlo").FontSize(11).TextColor(color)
				}
				if inline {
					ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, filepath.Base(row.Name)).MinWidth(0).Shrink(1).SingleLine().FontSize(13).TextColor(color).Tooltip(f.Path)
						if f.StatsKnown && !f.Binary {
							ui.RichText(c, ui.Span{Text: "("}, ui.Span{Text: fmt.Sprintf("+%d", f.Additions), Color: t.Success}, ui.Span{Text: " | "}, ui.Span{Text: fmt.Sprintf("-%d", f.Deletions), Color: t.Danger}, ui.Span{Text: ")"}).FontSize(13).Shrink(0)
						}
					})
				} else {
					ui.Text(c, filepath.Base(row.Name)).Grow(1).MinWidth(0).SingleLine().FontSize(12).Tooltip(f.Path)
				}
			}).OnClick(func() { a.openFileDiff(row.File) }).ContextMenu(func(m *ui.Menu) {
				if m.Item("Copy path").Chosen() {
					mygo.Clipboard.WriteText(f.Path)
				}
				if m.Item("Open working copy").Chosen() {
					if err := mygo.Shell.OpenPath(filepath.Join(a.repo(), f.Path)); err != nil {
						a.fail(err)
					}
				}
				if a.revision == "worktree" && m.Item("Discard file…").Chosen() {
					a.ask("Discard "+f.Path+"?", "This removes the working copy changes. Type discard to confirm.", "Discard", func(value string) {
						if value != "discard" {
							a.errorText = "Type discard to confirm"
							return
						}
						if f.Untracked {
							a.runGit("Removing untracked file…", "clean", "-f", "--", f.Path)
						} else {
							a.runGit("Discarding file…", "restore", "--source=HEAD", "--staged", "--worktree", "--", f.Path)
						}
					})
				}
			})
		})
	}).Grow(1).FillWidth().MinHeight(0)
	if list.Changed() && a.fileRowSelected >= 0 && a.fileRowSelected < len(a.fileRows) {
		row := a.fileRows[a.fileRowSelected]
		if row.Dir {
			a.collapsed[row.Path] = !a.collapsed[row.Path]
			a.rebuildFileRows()
		} else {
			a.openFileDiff(row.File)
		}
	}
}

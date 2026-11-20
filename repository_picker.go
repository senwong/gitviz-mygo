package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) filteredRepositories() []int {
	var indices []int
	query := strings.ToLower(strings.TrimSpace(a.repoQuery))
	for i, path := range a.repos {
		if strings.Contains(strings.ToLower(a.repoLabel(path)), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (a *App) openRepositoryPicker() {
	a.repoQuery = ""
	a.repoCursor = max(0, a.active)
	a.repoList = ui.ListState{}
	a.showRepos = true
}

func (a *App) switchRepositoryQueryToCommands() bool {
	query := strings.TrimLeft(a.repoQuery, " \t")
	if !strings.HasPrefix(query, ">") {
		return false
	}
	a.showRepos = false
	a.showCommands = true
	a.commandQuery = strings.TrimSpace(strings.TrimPrefix(query, ">"))
	a.commandInitialQuery = a.commandQuery
	a.commandCursor = 0
	a.commandsWereOpen = false
	return true
}

func (a *App) moveRepositoryCursor(delta int) {
	indices := a.filteredRepositories()
	if len(indices) == 0 {
		a.repoCursor = 0
		return
	}
	a.repoCursor = max(0, min(len(indices)-1, a.repoCursor+delta))
	a.repoList.ScrollIntoView(a.repoCursor)
}

func (a *App) confirmRepository() {
	indices := a.filteredRepositories()
	if len(indices) == 0 {
		return
	}
	a.switchRepo(indices[max(0, min(a.repoCursor, len(indices)-1))])
}

func (a *App) repositoryPicker(c *ui.Context) {
	justOpened := a.showRepos && !a.repoPickerWasOpen
	if justOpened {
		a.repoCursor = max(0, a.active)
		a.repoList = ui.ListState{}
		a.repoList.ScrollIntoView(a.repoCursor)
	}
	t := c.Theme()

	ui.PopoverBase(c.Key("repositories"), a.repositoryAnchor, &a.showRepos, func(panel ui.Element) {
		pickerPanel(panel, t, 320).Label("Repository dropdown").Children(func() {
			ui.Row(c).FillWidth().Height(40).Padding(6).Children(func() {
				input := ui.TextInput(c, &a.repoQuery).FillWidth().Height(28).FontSize(12).Placeholder("Search repositories by name…").Label("Search repositories")
				if justOpened {
					input.Focus()
				}
				if input.Changed() {
					a.switchRepositoryQueryToCommands()
					a.repoCursor = 0
					a.repoList = ui.ListState{}
				}
				input.OnTextKey(func(key ui.TextKey) bool {
					if key.Mods != 0 {
						return false
					}
					switch key.Key {
					case ui.KeyDown:
						a.repoQuery = key.Text()
						a.moveRepositoryCursor(1)
						return true
					case ui.KeyUp:
						a.repoQuery = key.Text()
						a.moveRepositoryCursor(-1)
						return true
					case ui.KeyEnter:
						a.repoQuery = key.Text()
						a.confirmRepository()
						return true
					}
					return false
				})
			})
			indices := a.filteredRepositories()
			a.repoCursor = max(0, min(a.repoCursor, len(indices)-1))
			if len(indices) == 0 {
				message := "No repositories found in the current workspace"
				if a.repoQuery != "" {
					message = fmt.Sprintf("No repositories match ‘%s’", a.repoQuery)
				}
				ui.Text(c, message).FillWidth().SingleLine().Padding(12).FontSize(12).TextColor(t.TextMuted)
			} else {
				a.repoList.Key = func(position int) any { return a.repos[indices[position]] }
				ui.List(c.Key("repository-results"), &a.repoList, len(indices), func(position int) {
					index := indices[position]
					path := a.repos[index]
					row := ui.ButtonBase(c.Key(path)).FillWidth().Height(28).PaddingX(8).Radius(0).Label("Repository: " + a.repoLabel(path)).Tooltip(a.repoLabel(path))
					color := pickerRowColor(row, t, a.listTheme, position == a.repoCursor)
					row.Children(func() {
						pickerCheck(c, path == a.repo(), color)
						ui.Text(c, a.repoLabel(path)).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(color)
					})
					if row.Clicked() {
						a.switchRepo(index)
					}
				}).FillWidth().Height(float32(min(12, len(indices))) * 28)
			}
			ui.Row(c).FillWidth().Height(32).PaddingX(8).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
				ui.Text(c, "↑↓ · Enter · Esc").Grow(1).FontSize(10).TextColor(t.TextMuted)
				ui.MenuButton(c, "Manage…", func(menu *ui.Menu) {
					if menu.Item("Add repository…").Chosen() {
						a.showRepos = false
						a.openRepository(false)
					}
					if menu.Item("Open workspace…").Chosen() {
						a.showRepos = false
						a.openRepository(true)
					}
					if menu.Item("Save workspace…").Chosen() {
						a.saveWorkspace()
					}
				}).Height(24).FontSize(11)
			})
		})
	})

	a.repoPickerWasOpen = a.showRepos
}

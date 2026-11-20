package main

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func pickerPanel(panel ui.Element, t *ui.Theme, width float32) ui.Element {
	return panel.Width(width).Margin(4, 0, 0, 0).Background(t.Surface).Border(1, t.Border).Radius(5).Clip().Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, .3))
}

func pickerRowColor(row ui.Element, t, selection *ui.Theme, keyboard bool) ui.Color {
	if row.Hovered() {
		row.Background(t.Accent)
		return t.AccentText
	}
	if keyboard {
		row.Background(selection.Accent)
	}
	return t.Text
}

func pickerCheck(c *ui.Context, checked bool, color ui.Color) {
	ui.Box(c).Width(20).Height(20).Shrink(0).Center().Children(func() {
		if checked {
			ui.Icon(c, checkIcon).Size(14, 14).TextColor(color)
		}
	})
}

func (a *App) filteredBranches() []Branch {
	var branches []Branch
	query := strings.ToLower(strings.TrimSpace(a.branchQuery))
	for _, b := range a.snapshot.Branches {
		if (b.Symbolic == "" || a.config.RemoteHeads) && (b.Remote || branchMatches(b.Name, a.config.BranchGlobs)) && strings.Contains(strings.ToLower(b.Name), query) {
			branches = append(branches, b)
		}
	}
	return branches
}

func (a *App) applyBranchChoice(position int) {
	branches := a.filteredBranches()
	if position < 0 || position > len(branches) {
		return
	}
	a.filterRef = ""
	if position > 0 {
		a.filterRef = branches[position-1].Full
	}
	a.showBranches = false
	a.load(false)
}

func (a *App) branchPicker(c *ui.Context) {
	if a.repo() == "" {
		a.showBranches = false
		a.branchesWereOpen = false
		return
	}
	justOpened := a.showBranches && !a.branchesWereOpen
	if justOpened {
		a.branchQuery = ""
		a.branchCursor = 0
		a.branchList = ui.ListState{}
	}
	t := c.Theme()
	ui.PopoverBase(c.Key("branches"), a.branchesAnchor, &a.showBranches, func(panel ui.Element) {
		pickerPanel(panel, t, 360).Label("Branch dropdown").Children(func() {
			ui.Row(c).FillWidth().Height(40).Padding(6).Children(func() {
				input := ui.TextInput(c, &a.branchQuery).FillWidth().Height(28).FontSize(12).Placeholder("Filter branches…").Label("Search branches")
				if justOpened {
					input.Focus()
				}
				if input.Changed() {
					a.branchCursor = 0
					if strings.TrimSpace(a.branchQuery) != "" && len(a.filteredBranches()) > 0 {
						a.branchCursor = 1
					}
					a.branchList = ui.ListState{}
				}
				input.OnTextKey(func(key ui.TextKey) bool {
					if key.Mods != 0 {
						return false
					}
					a.branchQuery = key.Text()
					switch key.Key {
					case ui.KeyDown:
						a.branchCursor = min(len(a.filteredBranches()), a.branchCursor+1)
						a.branchList.ScrollIntoView(a.branchCursor)
						return true
					case ui.KeyUp:
						a.branchCursor = max(0, a.branchCursor-1)
						a.branchList.ScrollIntoView(a.branchCursor)
						return true
					case ui.KeyEnter:
						a.applyBranchChoice(a.branchCursor)
						return true
					}
					return false
				})
			})
			branches := a.filteredBranches()
			if justOpened {
				for i, b := range branches {
					if a.selectedBranch(b.Full) {
						a.branchCursor = i + 1
						break
					}
				}
				a.branchList.ScrollIntoView(a.branchCursor)
			}
			a.branchCursor = max(0, min(len(branches), a.branchCursor))
			a.branchList.Key = func(i int) any {
				if i == 0 {
					return "show-all"
				}
				return branches[i-1].Full
			}
			ui.List(c.Key("branch-list"), &a.branchList, len(branches)+1, func(i int) {
				label, checked := "Show All", a.filterRef == ""
				if i > 0 {
					label = branches[i-1].Name
					checked = a.selectedBranch(branches[i-1].Full)
				}
				accessible := "Branch: " + label
				if i == 0 {
					accessible = label
				}
				row := ui.ButtonBase(c.Key(label)).FillWidth().Height(28).PaddingX(8).Radius(0).Label(accessible).Tooltip(label)
				color := pickerRowColor(row, t, a.listTheme, i == a.branchCursor)
				row.Children(func() {
					pickerCheck(c, checked, color)
					text := ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(color)
					if i > 0 && branches[i-1].Current {
						text.Bold()
					}
				})
				if row.Clicked() {
					a.applyBranchChoice(i)
				}
				if i > 0 {
					b := branches[i-1]
					row.ContextMenu(func(m *ui.Menu) { a.branchMenu(m, b) })
				}
			}).FillWidth().Height(float32(min(12, len(branches)+1)) * 28)
			if len(branches) == 0 {
				ui.Text(c, "No matching branches").FillWidth().Padding(12).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Text(c, "↑↓ choose · Enter apply · Esc close").FillWidth().Padding(8).FontSize(10).TextColor(t.TextMuted).BorderWidth(1, 0, 0, 0).BorderColor(t.Border)
		})
	})
	a.branchesWereOpen = a.showBranches
}

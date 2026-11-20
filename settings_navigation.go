package main

import "github.com/egoist/mygo/ui"

var settingsNavigationGroups = []struct {
	Title    string
	Sections []string
}{
	{"Interface", []string{"Appearance", "Columns", "Messages"}},
	{"Repository", []string{"History", "References", "Files", "Remotes"}},
	{"Application", []string{"Workspace", "Menus"}},
}

func settingsSectionIcon(section string) *ui.SVG {
	switch section {
	case "Appearance":
		return settingsAppearanceIcon
	case "History":
		return settingsHistoryIcon
	case "References":
		return branchIcon
	case "Columns":
		return settingsColumnsIcon
	case "Files":
		return folderIcon
	case "Messages":
		return settingsMessagesIcon
	case "Remotes":
		return cloudIcon
	case "Workspace":
		return settingsWorkspaceIcon
	default:
		return settingsMenusIcon
	}
}

func (a *App) settingsNavigation(c *ui.Context) {
	t := c.Theme()
	active := a.settingsSection
	if active == "" {
		active = "Appearance"
	}
	ui.Column(c).Width(172).Shrink(0).FillHeight().Padding(0, 12, 0, 0).Gap(12).BorderWidth(0, 1, 0, 0).BorderColor(t.Border.Alpha(.7)).Role(ui.RoleTabList).Label("Settings categories").FocusGroup(ui.Vertical).Children(func() {
		for _, group := range settingsNavigationGroups {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				ui.Text(c, group.Title).FontSize(10).Bold().TextColor(t.TextMuted).Padding(0, 10, 6, 10)
				for _, section := range group.Sections {
					selected := active == section
					item := ui.ButtonBase(c.Key(section)).FillWidth().Height(32).PaddingX(10).Radius(6).Label(section + " settings").Role(ui.RoleTab).AccessSelected(selected).FocusRing(true)
					color, iconColor := t.TextMuted, t.TextMuted
					if selected {
						color, iconColor = t.Text, t.Accent
						item.Background(t.Accent.Alpha(.10)).DrawOver(func(p *ui.Painter, r ui.Rect) {
							p.Fill(ui.Rect{X: r.X + 1, Y: r.Y + 8, W: 2, H: r.H - 16}, t.Accent, 1)
						})
					} else if item.Hovered() {
						item.Background(t.SurfaceHover)
						color = t.Text
					}
					item.Children(func() {
						ui.Row(c).FillWidth().Gap(9).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, settingsSectionIcon(section)).Size(16, 16).Shrink(0).TextColor(iconColor)
							label := ui.Text(c, section).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(color)
							if selected {
								label.FontWeight(600)
							}
						})
					})
					if item.Clicked() {
						a.settingsSection = section
						a.settingsQuery = ""
					}
				}
			})
		}
	})
}

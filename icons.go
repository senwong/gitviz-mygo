package main

import "github.com/egoist/mygo/ui"

func vectorIcon(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var folderIcon = vectorIcon(`<path d="M20 20H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2Z"/>`)
var branchIcon = vectorIcon(`<circle cx="6" cy="5" r="3"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="19" r="3"/><path d="M6 8v8M18 9a9 9 0 0 1-9 9"/>`)
var chevronIcon = vectorIcon(`<path d="m6 9 6 6 6-6"/>`)
var chevronRightIcon = vectorIcon(`<path d="m9 6 6 6-6 6"/>`)
var refreshIcon = vectorIcon(`<path d="M20 11a8 8 0 1 0-2 7M20 4v7h-7"/>`)
var settingsIcon = vectorIcon(`<path d="m9 3-1 3-3 1-2 3 2 2-1 3 2 3 3-1 2 3h3l1-3 3-1 2-3-2-2 1-3-2-3-3 1-2-3Z"/><circle cx="11.5" cy="11.5" r="3"/>`)
var downIcon = vectorIcon(`<path d="M12 4v16m-6-6 6 6 6-6"/>`)
var closeIcon = vectorIcon(`<path d="m6 6 12 12M6 18 18 6"/>`)
var cloudIcon = vectorIcon(`<path d="M20 16a4 4 0 0 0-1-8 7 7 0 0 0-13-1 5 5 0 0 0-1 10h14"/>`)
var checkIcon = vectorIcon(`<path d="m5 12 4 4L19 6"/>`)

func selectorButton(c *ui.Context, label, accessible string, width float32, icon *ui.SVG, shortcut ...string) ui.Element {
	t := c.Theme()
	button := ui.Button(c, "").Label(accessible).Width(width).Height(32).PaddingX(10).Shrink(0)
	if button.Hovered() {
		button.Border(1, t.Accent)
	}
	return button.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).Size(16, 16).Shrink(0).TextColor(t.TextMuted)
			ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(t.Text)
			if len(shortcut) > 0 && shortcut[0] != "" {
				toolbarShortcut(c, shortcut[0])
			}
			ui.Icon(c, chevronIcon).Size(12, 12).Shrink(0).TextColor(t.TextMuted)
		})
	})
}

func iconButton(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	return ui.Button(c, "").Label(label).Width(32).Height(32).Padding(0).Shrink(0).Children(func() {
		ui.Icon(c, icon).Size(16, 16).Shrink(0).TextColor(c.Theme().TextMuted)
	})
}

var tagIcon = vectorIcon(`<path d="M3 3h8l10 10-8 8L3 11Z"/><circle cx="7.5" cy="7.5" r="1"/>`)

var expandIcon = vectorIcon(`<path d="M8 3H3v5M16 3h5v5M3 16v5h5M21 16v5h-5"/>`)
var collapseIcon = vectorIcon(`<path d="M3 8h5V3M21 8h-5V3M8 21v-5H3M16 21v-5h5"/>`)

var settingsAppearanceIcon = vectorIcon(`<path d="M12 3a9 9 0 1 0 9 9c0-2-2-2-3-2h-1a2 2 0 0 1-2-2c0-1 1-2 1-3s-2-2-4-2Z"/><circle cx="7" cy="12" r=".8"/><circle cx="9" cy="7.5" r=".8"/><circle cx="8.5" cy="16.5" r=".8"/>`)
var settingsHistoryIcon = vectorIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`)
var settingsColumnsIcon = vectorIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16M15 4v16"/>`)
var settingsMessagesIcon = vectorIcon(`<path d="M21 15a3 3 0 0 1-3 3H8l-5 3V6a3 3 0 0 1 3-3h12a3 3 0 0 1 3 3Z"/><path d="M7 8h10M7 12h7"/>`)
var settingsWorkspaceIcon = vectorIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M8 9v11"/>`)
var settingsMenusIcon = vectorIcon(`<path d="M7 6h14M7 12h14M7 18h14"/><circle cx="3" cy="6" r=".6"/><circle cx="3" cy="12" r=".6"/><circle cx="3" cy="18" r=".6"/>`)

var copyIcon = vectorIcon(`<rect x="8" y="8" width="13" height="13" rx="2"/><path d="M16 8V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h3"/>`)
var fileIcon = vectorIcon(`<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6"/>`)

func toolbarShortcut(c *ui.Context, shortcut string) {
	ui.Text(c, shortcut).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine().Shrink(0)
}

func toolbarKeyButton(c *ui.Context, label, shortcut string, icon *ui.SVG) ui.Element {
	return ui.Button(c, "").Label(label).Height(32).PaddingX(8).Shrink(0).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if icon != nil {
				ui.Icon(c, icon).Size(16, 16).Shrink(0).TextColor(c.Theme().TextMuted)
			} else {
				ui.Text(c, label).FontSize(13).SingleLine()
			}
			toolbarShortcut(c, shortcut)
		})
	})
}

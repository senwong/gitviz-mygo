package main

import "github.com/egoist/mygo"

func (a *App) installMenu() {
	action := func(label, shortcut string, run func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: shortcut, Click: func(*mygo.MenuItem, *mygo.Window) {
			if !a.modalOpen() {
				run()
			}
		}}
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			action("Open Repository…", "CmdOrCtrl+O", func() { a.openRepository(false) }),
			action("Open Workspace…", "CmdOrCtrl+Shift+O", func() { a.openRepository(true) }),
			action("Save Workspace…", "", a.saveWorkspace),
			mygo.Separator(),
			action("Switch Repository…", nativeShortcut(a.prefs.RepositoryShortcut), func() { a.openRepositoryPicker(); a.win.Invalidate() }),
			action("Close", "CmdOrCtrl+W", a.closeViewOrWindow),
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "View", Submenu: []*mygo.MenuItem{
			action("Refresh Graph", "CmdOrCtrl+R", func() { a.load(true); a.win.Invalidate() }),
			action("Find…", "CmdOrCtrl+F", func() { a.openSearch(); a.win.Invalidate() }),
			action("Branches…", "CmdOrCtrl+B", func() { a.showBranches = true; a.win.Invalidate() }),
			action("Toggle Details", "CmdOrCtrl+D", func() { a.config.ShowDetail = !a.detailVisible; a.saveConfig(false); a.win.Invalidate() }),
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
	}))
}

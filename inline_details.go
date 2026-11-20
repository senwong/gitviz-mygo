package main

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

func (a *App) openFileDiff(index int) {
	if index < 0 || index >= len(a.files) {
		return
	}
	if a.selectedFile != index || (len(a.diffLines) == 0 && !a.diffLoading) {
		a.selectFile(index)
	}
	if a.prefs.InlineDetails {
		a.largeDiffVisible = true
		if a.showSearch {
			a.openSearch()
		}
	}
}

func (a *App) closeInlineDetails() {
	a.detailVisible = false
	a.config.ShowDetail = false
	a.largeDiffVisible = false
	a.focusCommitList = true
	a.saveConfig(false)
}

func (a *App) inlineCommitDetails(c *ui.Context) {
	ui.Themed(c, a.uiTheme).Fill().Children(func() { a.inlineCommitDetailsContent(c) })
}

func (a *App) inlineCommitDetailsContent(c *ui.Context) {
	t := a.uiTheme
	ui.Row(c).Fill().MinHeight(0).Background(t.Surface).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Label("Inline commit details").Children(func() {
		ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).FillHeight().Padding(10).Children(func() {
			if a.selectedCommit != nil {
				a.commitSummary(c, a.selectedCommit)
			} else {
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "Uncommitted changes").Bold().FontSize(13)
					ui.Text(c, "Select a file to view its diff.").FontSize(13).TextColor(t.TextMuted)
				})
			}
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).FillHeight().MinHeight(0).Padding(6, 4).BorderWidth(0, 0, 0, 1).BorderColor(t.Border).Children(func() {
			if a.detailLoading {
				ui.Spinner(c)
			}
			a.fileListView(c)
		})
		ui.Column(c).Width(30).Shrink(0).FillHeight().Padding(4, 3).Gap(4).Children(func() {
			if detailControl(c, "Close details", closeIcon).Clicked() {
				a.closeInlineDetails()
			}
			if detailControl(c, "Copy SHA", copyIcon).Clicked() {
				a.copyText(a.revision)
			}
			if detailControl(c, "Tree / List", folderIcon).Clicked() {
				a.config.FileTree = !a.config.FileTree
				a.saveConfig(false)
			}
		})
	})
}

func (a *App) returnToGraph() {
	if a.searchDiff {
		a.restoreGraphSearch()
	}
	if !a.prefs.InlineDetails || strings.HasPrefix(a.revision, "stash@{") {
		a.closeInlineDetails()
	}
	a.largeDiffVisible = false
	a.focusCommitList = true
}

func (a *App) largeDiff(c *ui.Context) {
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Background(a.uiTheme.Surface).Children(func() {
		ui.Row(c).FillWidth().Height(34).Shrink(0).PaddingX(10).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "← Back to Git Graph").Label("Back to Git Graph").Height(28).FontSize(12).Background(a.uiTheme.Accent.Alpha(.15)).Border(1, a.uiTheme.Accent.Alpha(.45)).TextColor(a.uiTheme.Accent).Tooltip("Return to Git Graph · Esc").Clicked() {
				a.returnToGraph()
			}
			ui.Text(c, "Esc").FontSize(11).TextColor(a.uiTheme.TextMuted)
			if a.selectedCommit != nil {
				ui.Text(c, a.messageText(a.selectedCommit.Subject)).Grow(1).MinWidth(0).SingleLine().FontSize(12)
			}
		})
		a.filesToolbar(c)
		ui.Split(c, &a.fileWidth, func() { ui.Column(c).Fill().MinHeight(0).Children(func() { a.fileListView(c) }) }, func() { a.diff(c) }).Grow(1).FillWidth().MinHeight(0).SplitReset(240)
	})
}

func paintContinuation(p *ui.Painter, r ui.Rect, commit Commit, colors []ui.Color) {
	lanes := map[int]bool{}
	for _, lane := range commit.Through {
		lanes[lane] = true
	}
	for _, lane := range commit.Outgoing {
		lanes[lane] = true
	}
	if commit.Bottom {
		lanes[commit.Lane] = true
	}
	for lane := range lanes {
		x := r.X + 12 + float32(lane)*18
		p.Line(x, r.Y, x, r.Y+r.H, 2, colors[lane%len(colors)])
	}
}

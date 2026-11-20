package main

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func commitMetadataRow(c *ui.Context, label string, value func()) {
	ui.RichText(c).FillWidth().FontSize(13).FixedLineHeight(18).Selectable().Children(func() {
		ui.Text(c, label+": ").Bold()
		value()
	})
}

func commitPerson(c *ui.Context, name, email string) {
	ui.Text(c, name)
	if email != "" {
		ui.Text(c, " <")
		ui.Link(c, email, "mailto:"+email).Underline()
		ui.Text(c, ">")
	}
}

func formatCommitDate(date time.Time) string {
	return date.Format("Mon Jan 2 2006 15:04:05 GMT-0700 (MST)")
}

func (a *App) openParentCommit(sha string) {
	for i, commit := range a.snapshot.Commits {
		if commit.SHA == sha {
			a.openCommit(i)
			a.list.ScrollIntoView(i)
			return
		}
	}
	a.filterRef = sha
	a.config.ShowDetail = true
	a.load(false)
}

func (a *App) commitSummary(c *ui.Context, commit *Commit) {
	ui.Column(c).FillWidth().Gap(0).Children(func() {
		commitMetadataRow(c, "Commit", func() { ui.Text(c, commit.SHA) })
		commitMetadataRow(c, "Parents", func() {
			if len(commit.Parents) == 0 {
				ui.Text(c, "None")
			}
			for i, parent := range commit.Parents {
				if i > 0 {
					ui.Text(c, ", ")
				}
				if ui.Link(c, parent, "").Underline().Tooltip("View parent commit").Clicked() {
					a.update(func() { a.openParentCommit(parent) })
				}
			}
		})
		commitMetadataRow(c, "Author", func() { commitPerson(c, commit.Author, commit.Email) })
		committed := commit.CommitTime
		if committed.IsZero() {
			committed = commit.Time
		}
		differentDate := !committed.Equal(commit.Time)
		if differentDate {
			commitMetadataRow(c, "Author Date", func() { ui.Text(c, formatCommitDate(commit.Time)) })
		}
		name, email := commit.Committer, commit.CommitterEmail
		if name == "" && email == "" {
			name, email = commit.Author, commit.Email
		}
		commitMetadataRow(c, "Committer", func() { commitPerson(c, name, email) })
		label := "Date"
		if differentDate {
			label = "Committer Date"
		}
		commitMetadataRow(c, label, func() { ui.Text(c, formatCommitDate(committed)) })
		message := strings.TrimSpace(commit.Subject)
		if commit.Body != "" {
			message += "\n\n" + commit.Body
		}
		a.messageElement(c, message).FillWidth().FontSize(13).FixedLineHeight(18).Selectable().Margin(20, 0, 0, 0)
	})
}

func detailControl(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	t := c.Theme()
	button := ui.ButtonBase(c).Size(24, 24).Radius(4).Label(label).Tooltip(label).FocusRing(true)
	color := t.TextMuted
	if button.Hovered() {
		button.Background(t.SurfaceHover)
		color = t.Text
	}
	button.Children(func() { ui.Box(c).Fill().Center().Children(func() { ui.Icon(c, icon).Size(15, 15).TextColor(color) }) })
	return button
}

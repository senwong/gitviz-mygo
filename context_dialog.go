package main

import (
	"github.com/egoist/mygo/ui"
)

type actionField struct {
	Label, Value string
	Options      []string
	Checked      bool
	Check        bool
	Multiple     bool
	Selected     map[string]bool
}
type actionForm struct {
	Title, Description, Confirm, Repo, Error string
	Fields                                   []*actionField
	Run                                      func() error
	Open, Focused                            bool
}

func textField(label, value string) *actionField { return &actionField{Label: label, Value: value} }
func checkField(label string, value bool) *actionField {
	return &actionField{Label: label, Check: true, Checked: value}
}
func selectField(label, value string, options ...string) *actionField {
	return &actionField{Label: label, Value: value, Options: options}
}
func (a *App) showActionForm(title, description, confirm string, fields []*actionField, run func() error) {
	a.contextForm = &actionForm{Title: title, Description: description, Confirm: confirm, Repo: a.repo(), Fields: fields, Run: run, Open: true}
	a.showBranches = false
	a.showCommands = false
	a.showStashes = false
}
func (a *App) submitActionForm(f *actionForm) {
	if a.contextForm != f || !f.Open {
		return
	}
	if f.Repo != a.repo() {
		f.Error = "The repository changed. Close this dialog and try again."
		return
	}
	if a.busy[f.Repo] != "" {
		f.Error = "Wait for the current Git operation to finish."
		return
	}
	if err := f.Run(); err != nil {
		f.Error = err.Error()
		return
	}
	if a.contextForm == f {
		f.Open = false
		a.contextForm = nil
	}
}
func (a *App) actionDialog(c *ui.Context) {
	f := a.contextForm
	if f == nil {
		return
	}
	ui.Modal(c.Key("git-action-dialog"), &f.Open, func() {
		ui.Column(c).Width(580).MaxHeight(530).Padding(20).Gap(14).Children(func() {
			ui.Text(c, f.Title).FontSize(17).Bold()
			ui.Scroll(c).FillWidth().MaxHeight(380).Children(func() {
				ui.Column(c).FillWidth().Gap(12).Children(func() {
					ui.Text(c, f.Description).FillWidth().TextColor(c.Theme().TextMuted)
					for _, field := range f.Fields {
						ui.Column(c.Key(field.Label)).FillWidth().Gap(5).Children(func() {
							switch {
							case field.Check:
								ui.Checkbox(c, &field.Checked, field.Label)
							case field.Multiple:
								ui.Text(c, field.Label).Bold()
								for _, option := range field.Options {
									value := field.Selected[option]
									if ui.Checkbox(c.Key(option), &value, option).Changed() {
										field.Selected[option] = value
									}
								}
							case len(field.Options) > 0:
								ui.Text(c, field.Label).FontSize(12)
								ui.Select(c, &field.Value, field.Options).FillWidth().Label(field.Label)
							default:
								ui.Text(c, field.Label).FontSize(12)
								input := ui.TextInput(c, &field.Value).FillWidth().Label(field.Label)
								if !f.Focused {
									input.Focus()
									f.Focused = true
								}
								if input.Submitted() {
									a.submitActionForm(f)
								}
							}
						})
					}
					if f.Error != "" {
						ui.Text(c, f.Error).FillWidth().TextColor(c.Theme().Danger)
					}
				})
			})
			ui.Row(c).FillWidth().Gap(10).Justify(ui.End).Children(func() {
				if ui.Button(c, "Cancel").Clicked() {
					f.Open = false
				}
				if ui.PrimaryButton(c, f.Confirm).Label("Confirm " + f.Confirm).Disabled(a.busy[f.Repo] != "").Clicked() {
					a.submitActionForm(f)
				}
			})
		})
	})
	if !f.Open && a.contextForm == f {
		a.contextForm = nil
	}
}
func (a *App) modalOpen() bool {
	return a.showRepos || a.showBranches || a.showSettings || a.showCommands || a.showStashes || a.promptOpen || a.syncOpen || a.errorText != "" || a.contextForm != nil
}

package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSettingsNavigationSwitchesSectionsAndSupportsKeyboard(t *testing.T) {
	a := uiFixture()
	a.showSettings = true
	tt := ui.NewTester(a.view, 1380, 900)
	for _, section := range settingsSections {
		a.settingsQuery = "author"
		tt.Frame()
		if err := tt.Click(section + " settings"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if a.settingsSection != section || a.settingsQuery != "" {
			t.Fatalf("navigation did not activate %s", section)
		}
	}
	if err := tt.Click("Appearance settings"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if a.settingsSection != "Columns" {
		t.Fatalf("keyboard did not activate next category: %s", a.settingsSection)
	}
	tt.Key(0, ui.KeyEscape)
	if a.showSettings {
		t.Fatal("Escape did not close settings")
	}
}

func TestSettingsNavigationFitsAndUsesQuietSelection(t *testing.T) {
	for _, theme := range []string{"One Dark Pro", "GitHub Light"} {
		t.Run(theme, func(t *testing.T) {
			a := uiFixture()
			a.prefs.Theme = theme
			a.setTheme()
			a.showSettings = true
			a.settingsSection = "Appearance"
			tt := ui.NewTester(a.view, 1380, 900)
			selected, ok := tt.Find("Appearance settings")
			if !ok || selected.H != 32 {
				t.Fatalf("navigation is not compact: %+v", selected)
			}
			frame := tt.Image()
			point := frame.RGBAAt(int(selected.X+selected.W-10), int(selected.Y+selected.H/2))
			if nearColor(point.R, point.G, point.B, a.uiTheme.Accent) {
				t.Fatal("selected navigation still has a solid accent background")
			}
			for _, section := range settingsSections {
				rect, ok := tt.Find(section + " settings")
				if !ok || rect.X != selected.X || rect.W != selected.W || rect.H != selected.H {
					t.Fatalf("inconsistent category layout: %s %+v", section, rect)
				}
			}
			captureMigrationPreview(t, tt, "settings-navigation-"+theme)
			tt.SetSize(1000, 620)
			menus, ok := tt.Find("Menus settings")
			footer, _ := tt.Find("Import .gitviz.conf")
			if !ok || menus.Y+menus.H > footer.Y || menus.X < 0 || menus.X+menus.W > 1000 {
				t.Fatalf("navigation overflows smallest window: menus=%+v footer=%+v", menus, footer)
			}
			captureMigrationPreview(t, tt, "settings-navigation-minimum-"+theme)
		})
	}
}

package main

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestReferenceTextUsesThemeTextInsideSelectedList(t *testing.T) {
	for _, theme := range []string{"MyGo Dark", "One Dark Pro", "GitHub Light"} {
		for _, selected := range []bool{false, true} {
			t.Run(theme+map[bool]string{false: "/normal", true: "/selected"}[selected], func(t *testing.T) {
				a := uiFixture()
				a.prefs.Theme = theme
				a.setTheme()
				a.snapshot.Commits[0].Refs = []string{"release\x1forigin", "tag:v1"}
				if selected {
					a.listSelected = 0
				}
				tt := ui.NewTester(a.view, 1380, 900)
				for _, label := range []string{"release", "origin"} {
					r, ok := tt.Find(label)
					if !ok {
						t.Fatalf("missing ref %s", label)
					}
					frame := tt.Image()
					accent := a.uiTheme.Text
					pixels := 0
					for y := int(r.Y); y < int(r.Y+r.H); y++ {
						for x := int(r.X); x < int(r.X+r.W); x++ {
							p := frame.RGBAAt(x, y)
							if abs(int(p.R)-int(accent.R)) < 24 && abs(int(p.G)-int(accent.G)) < 24 && abs(int(p.B)-int(accent.B)) < 24 {
								pixels++
							}
						}
					}
					if pixels < 5 {
						t.Fatalf("%s lost readable theme text: only %d text pixels; list selection color must not be used for reference text", label, pixels)
					}
				}
				if selected {
					captureMigrationPreview(t, tt, "reference-style-"+theme)
				}
			})
		}
	}
}
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

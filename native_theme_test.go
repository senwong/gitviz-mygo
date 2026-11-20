package main

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestNativeAppearanceFollowsSelectedTheme(t *testing.T) {
	a := uiFixture()
	for _, test := range []struct {
		name string
		want mygo.ThemeSource
	}{
		{"One Dark Pro", mygo.ThemeDark}, {"GitHub Light", mygo.ThemeLight}, {"MyGo Dark", mygo.ThemeDark}, {"MyGo Light", mygo.ThemeLight}, {"Dracula", mygo.ThemeDark},
	} {
		a.prefs.Theme = test.name
		a.setTheme()
		if got := a.nativeThemeSource(); got != test.want {
			t.Fatalf("%s: native appearance is %s, want %s", test.name, got, test.want)
		}
	}
}

package main

import (
	"github.com/egoist/mygo/ui"
	"math"
)

type themeSpec struct {
	Name                                                                      string
	Dark                                                                      bool
	Background, Surface, Border, Text, Muted, Hover, Selected, Accent, Danger string
}

var themes = []themeSpec{
	{"Shadcn Dark", true, "#09090b", "#111113", "#27272a", "#fafafa", "#a1a1aa", "#27272a", "#27272a", "#fafafa", "#ef4444"},
	{"Shadcn Light", false, "#f7f8fa", "#ffffff", "#e5e7eb", "#09090b", "#71717a", "#f1f5f9", "#eaf2ff", "#2563eb", "#dc2626"},
	{"Dark+", true, "#1e1e1e", "#252526", "#3c3c3c", "#d4d4d4", "#8a8a8a", "#2a2d2e", "#094771", "#4daafc", "#f14c4c"},
	{"One Dark Pro", true, "#282c34", "#21252b", "#3b4048", "#abb2bf", "#929aaa", "#2c313a", "#3e4451", "#61afef", "#e06c75"},
	{"Dracula", true, "#282a36", "#21222c", "#44475a", "#f8f8f2", "#6272a4", "#343746", "#44475a", "#bd93f9", "#ff5555"},
	{"Nord", true, "#2e3440", "#3b4252", "#4c566a", "#d8dee9", "#7b88a1", "#3b4252", "#434c5e", "#88c0d0", "#bf616a"},
	{"Solarized Dark", true, "#002b36", "#073642", "#0d4b56", "#93a1a1", "#586e75", "#0a3a45", "#0d4b56", "#268bd2", "#dc322f"},
	{"GitHub Light", false, "#ffffff", "#f6f8fa", "#d0d7de", "#1f2328", "#656d76", "#eaeef2", "#ddf4ff", "#0969da", "#cf222e"},
	{"One Light", false, "#fafafa", "#f0f0f0", "#d4d4d4", "#383a42", "#a0a1a7", "#e8e8e8", "#d4e0f5", "#4078f2", "#e45649"},
	{"Solarized Light", false, "#fdf6e3", "#f5efdc", "#eee8d5", "#586e75", "#93a1a1", "#eee8d5", "#e4ddc1", "#268bd2", "#dc322f"},
	{"Quiet Light", false, "#ffffff", "#f3f3f3", "#dcdcdc", "#333333", "#888888", "#e8e8e8", "#cde8ff", "#4b83cd", "#d16969"},
	{"Gruvbox Light", false, "#fbf1c7", "#f2e5bc", "#ebdbb2", "#3c3836", "#928374", "#ebdbb2", "#d5c4a1", "#076678", "#9d0006"},
}

func themeNames() []string {
	names := []string{"MyGo Dark", "MyGo Light"}
	for _, s := range themes {
		names = append(names, s.Name)
	}
	return names
}

func namedTheme(name string) (*ui.Theme, ui.Color, bool) {
	for _, s := range themes {
		if s.Name != name {
			continue
		}
		t := ui.LightTheme()
		if s.Dark {
			t = ui.DarkTheme()
		}
		t.Background = ui.Hex(s.Background)
		t.Surface = ui.Hex(s.Surface)
		t.Border = ui.Hex(s.Border)
		t.Text = ui.Hex(s.Text)
		t.TextMuted = ui.Hex(s.Muted)
		t.SurfaceHover = ui.Hex(s.Hover)
		t.Accent = ui.Hex(s.Accent)
		t.Danger = ui.Hex(s.Danger)
		t.InactiveAccent = t.Accent
		return t, ui.Hex(s.Selected), true
	}
	return nil, ui.Transparent, false
}

func colorLuminance(color ui.Color) float64 {
	linear := func(value uint8) float64 {
		v := float64(value) / 255
		if v <= .04045 {
			return v / 12.92
		}
		return math.Pow((v+.055)/1.055, 2.4)
	}
	return .2126*linear(color.R) + .7152*linear(color.G) + .0722*linear(color.B)
}

func accentForeground(background ui.Color) ui.Color {
	luminance := colorLuminance(background)
	if (luminance+.05)/.05 > 1.05/(luminance+.05) {
		return ui.Hex("#000000")
	}
	return ui.Hex("#ffffff")
}

func contrastRatio(a, b ui.Color) float64 {
	first, second := colorLuminance(a), colorLuminance(b)
	return (math.Max(first, second) + .05) / (math.Min(first, second) + .05)
}

// Preserve a theme's hue, moving only as far toward its text color as needed
// to keep small labels readable on the surfaces where they are rendered.
func readableThemeColor(color, toward ui.Color, backgrounds ...ui.Color) ui.Color {
	readable := func(candidate ui.Color) bool {
		for _, background := range backgrounds {
			if contrastRatio(candidate, background) < 4.5 {
				return false
			}
		}
		return true
	}
	if readable(color) {
		return color
	}
	for step := 1; step <= 100; step++ {
		candidate := color.Mix(toward, float32(step)/100)
		if readable(candidate) {
			return candidate
		}
	}
	return toward
}

func polishTheme(t *ui.Theme, selected ui.Color) {
	t.SurfacePressed = t.SurfaceHover.Mix(t.Text, .04)
	backgrounds := []ui.Color{t.Background, t.Surface, t.SurfaceHover, selected}
	t.TextMuted = readableThemeColor(t.TextMuted, t.Text, backgrounds...)
	t.Accent = readableThemeColor(t.Accent, t.Text, t.Background, t.Surface)
	t.Warning = readableThemeColor(t.Warning, t.Text, backgrounds...)
	t.Success = readableThemeColor(t.Success, t.Text, append(backgrounds, t.Background.Mix(t.Success, .07))...)
	t.Danger = readableThemeColor(t.Danger, t.Text, append(backgrounds, t.Background.Mix(t.Danger, .07))...)
	t.AccentText = accentForeground(t.Accent)
	// A shared foreground must remain readable through all primary-button states.
	t.AccentHover = t.Accent.Mix(t.AccentText, .08)
	t.AccentPressed = t.Accent.Mix(t.AccentText, .16)
	away := ui.Hex("#ffffff")
	if t.AccentText == away {
		away = ui.Hex("#000000")
	}
	t.AccentHover = readableThemeColor(t.AccentHover, away, t.AccentText)
	t.AccentPressed = readableThemeColor(t.AccentPressed, away, t.AccentText)
	t.InactiveAccent = t.Accent
	t.Focus = t.Accent.Alpha(.65)
	t.Selection = t.Accent.Alpha(.25)
}

package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestReferenceSearchSpans(t *testing.T) {
	for _, tc := range []struct {
		text, query string
		want        []string
	}{
		{"publish-cn-20261112", "202611", []string{"202611"}},
		{"publish-202611-20261112", "202611", []string{"202611", "202611"}},
		{"Publish-CN-20261112", "  publish-cn  ", []string{"Publish-CN"}},
		{"İ发布-20261112", "202611", []string{"202611"}},
		{"发布-中文-中文", "中文", []string{"中文", "中文"}},
		{"feature/a.b", "a.b", []string{"a.b"}},
		{"feature/axb", "a.b", nil},
		{"publish-cn-20261112", " ", nil},
		{"publish-cn-20261112", "absent", nil},
	} {
		for _, dark := range []bool{true, false} {
			spans := referenceSearchSpans(tc.text, tc.query, dark)
			var rebuilt strings.Builder
			var matches []string
			for _, span := range spans {
				rebuilt.WriteString(span.Text)
				if span.Background.A > 0 {
					matches = append(matches, span.Text)
				}
				if span.Weight != 0 || span.PaddingX != 0 {
					t.Fatal("highlight changes text metrics")
				}
			}
			if rebuilt.String() != tc.text || strings.Join(matches, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("%q query %q: text=%q matches=%v want=%v", tc.text, tc.query, rebuilt.String(), matches, tc.want)
			}
		}
	}
}

func TestTagSearchHighlightsSubstringWithoutMovingLabels(t *testing.T) {
	for _, theme := range []string{"MyGo Dark", "GitHub Light"} {
		t.Run(theme, func(t *testing.T) {
			a := uiFixture()
			a.prefs.Theme = theme
			a.setTheme()
			a.snapshot.Commits[0].Refs = []string{"release\x1forigin", "tag:publish-cn-20261112"}
			a.snapshot.Tags = map[string]Tag{"publish-cn-20261112": {Name: "publish-cn-20261112", Annotated: true}}
			tt := newMenuTester(a, 1380, 900)
			before, ok := tt.Find("publish-cn-20261112")
			if !ok {
				t.Fatal("missing tag")
			}
			tt.Key(ui.Cmd, ui.KeyF)
			tt.Type("202611")
			after, ok := tt.Find("publish-cn-20261112")
			if !ok || before != after {
				t.Fatalf("highlight moved tag: before=%+v after=%+v", before, after)
			}
			background := ui.Hex("#a66b00")
			if !a.uiTheme.Dark {
				background = ui.Hex("#ffe08a")
			}
			frame := tt.Image()
			columns := 0
			for x := int(after.X); x < int(after.X+after.W); x++ {
				for y := int(after.Y); y < int(after.Y+after.H); y++ {
					pixel := frame.RGBAAt(x, y)
					if nearColor(pixel.R, pixel.G, pixel.B, background) {
						columns++
						break
					}
				}
			}
			if columns < 20 || float32(columns) >= after.W*.6 {
				t.Fatalf("expected only matching digits highlighted: colored columns=%d tag width=%v", columns, after.W)
			}
			captureMigrationPreview(t, tt, "tag-search-highlight-"+theme)
			if err := tt.RightClick("publish-cn-20261112"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(tt.Menu(), "|"), "View Details") {
				t.Fatal("highlight lost tag context menu")
			}
			tt.CloseMenu()
			tt.Key(0, ui.KeyEscape)
			if a.query != "" {
				t.Fatal("search did not clear")
			}
			for _, span := range referenceSearchSpans("publish-cn-20261112", a.query, a.uiTheme.Dark) {
				if span.Background.A != 0 {
					t.Fatal("highlight remained after closing search")
				}
			}
		})
	}
}

func TestCommitSubjectSearchHighlightsWithoutMovingGraph(t *testing.T) {
	for _, theme := range themeNames() {
		for _, size := range [][2]int{{1380, 900}, {1000, 620}} {
			a := searchGraphFixture()
			a.prefs.Theme = theme
			a.setTheme()
			subject := "dev-10.44.1-2 PLA 2026 审查员敬称 10.44.1-2 · 中文文案"
			a.snapshot.Commits[0].Subject = subject
			tt := ui.NewTester(a.view, size[0], size[1])
			before, ok := tt.Find(subject)
			if !ok {
				t.Fatal("commit subject missing")
			}
			next, _ := tt.Find(a.snapshot.Commits[1].Subject)
			tt.Key(ui.Cmd, ui.KeyF)
			tt.Type("10.44.1-2")
			after, ok := tt.Find(subject)
			nextAfter, _ := tt.Find(a.snapshot.Commits[1].Subject)
			if !ok || before != after || next != nextAfter {
				t.Fatalf("%s: search moved commit rows: before=%+v after=%+v", theme, before, after)
			}
			background := ui.Hex("#ffe08a")
			if a.uiTheme.Dark {
				background = ui.Hex("#a66b00")
			}
			pixels := 0
			frame := tt.Image()
			for y := int(after.Y); y < int(after.Y+after.H); y++ {
				for x := int(after.X); x < int(after.X+after.W); x++ {
					p := frame.RGBAAt(x, y)
					if nearColor(p.R, p.G, p.B, background) {
						pixels++
					}
				}
			}
			if pixels < 100 {
				t.Fatalf("%s: matching commit text was not highlighted", theme)
			}
			if theme == "Nord" && size[0] == 1380 {
				captureMigrationPreview(t, tt, "commit-subject-search-highlight")
			}
			tt.RightClickAt(after.X+5, after.Y+after.H/2)
			if !strings.Contains(strings.Join(tt.Menu(), "|"), "Copy Commit Hash") {
				t.Fatal("highlight lost commit context menu")
			}
			tt.CloseMenu()
			tt.Key(0, ui.KeyEscape)
			afterClose, _ := tt.Find(subject)
			if afterClose != before || a.query != "" {
				t.Fatal("closing search changed the row")
			}
		}
	}
}

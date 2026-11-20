package main

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestFloatingSearchDoesNotShiftGraph(t *testing.T) {
	a := uiFixture()
	tt := ui.NewTester(a.view, 1380, 900)
	before, ok := tt.Find("A test commit")
	if !ok {
		t.Fatal("missing commit")
	}
	tt.Key(ui.Cmd, ui.KeyF)
	after, ok := tt.Find("A test commit")
	if !ok || before != after {
		t.Fatalf("search shifted graph: before=%+v after=%+v", before, after)
	}
	tt.Type("not present")
	if a.query != "not present" || !tt.HasText("0 matches") {
		t.Fatal("search was not focused or no-match feedback is missing")
	}
	captureMigrationPreview(t, tt, "floating-search-empty")
	tt.Key(0, ui.KeyEscape)
	if a.showSearch || a.query != "" {
		t.Fatal("Escape did not close and clear search")
	}
	restored, _ := tt.Find("A test commit")
	if restored != before {
		t.Fatal("closing search shifted graph")
	}
	tt.SetSize(1000, 620)
	tt.Key(ui.Cmd, ui.KeyF)
	search, ok := tt.Find("Search commits")
	if !ok || search.X < 0 || search.X+search.W > 1000 {
		t.Fatalf("search overflows minimum window: %+v", search)
	}
	captureMigrationPreview(t, tt, "floating-search-minimum")
	if err := tt.Click("Close search"); err != nil {
		t.Fatal(err)
	}
	if a.showSearch {
		t.Fatal("close button failed")
	}
}

func TestSearchInputWidthStaysFixedAcrossQueryStates(t *testing.T) {
	for _, theme := range themeNames() {
		for _, size := range [][2]int{{1380, 900}, {1000, 620}} {
			a := searchGraphFixture()
			a.prefs.Theme = theme
			a.setTheme()
			a.showSearch = true
			tt := ui.NewTester(a.view, size[0], size[1])
			initial, ok := tt.Find("Search commits")
			if !ok {
				t.Fatal("search input missing")
			}
			for _, query := range []string{"needle", "no-matching-commit", "a very long search query that must not resize its input field", ""} {
				tt.Key(ui.Cmd, ui.KeyA)
				tt.Key(0, ui.KeyBackspace)
				tt.Type(query)
				tt.Frame()
				current, ok := tt.Find("Search commits")
				if !ok || current != initial {
					t.Fatalf("%s %dx%d query=%q resized search: before=%+v after=%+v", theme, size[0], size[1], query, initial, current)
				}
			}
		}
	}
}

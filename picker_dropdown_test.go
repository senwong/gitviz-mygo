package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func dropdownFixture() *App {
	a := uiFixture()
	a.repos = []string{"/work/alpha", "/work/beta", "/work/gamma"}
	a.active = 0
	a.snapshot.Branches = []Branch{{Name: "release", Full: "refs/heads/release", Current: true}, {Name: "feature/ui", Full: "refs/heads/feature/ui"}, {Name: "origin/feature/ui", Full: "refs/remotes/origin/feature/ui", Remote: true}}
	return a
}

func checkDropdownAnchor(t *testing.T, tt *ui.Tester, panel, anchor string, width int) {
	t.Helper()
	r, ok := tt.Find(panel)
	b, ok2 := tt.Find(anchor)
	if !ok || !ok2 || abs(int(r.X-b.X)) > 1 || abs(int(r.Y-b.Y-b.H-4)) > 1 || r.X+r.W > float32(width) || r.Y+r.H > 620 {
		t.Fatalf("dropdown is not anchored/in viewport: panel=%+v anchor=%+v", r, b)
	}
}

func checkPickerHover(t *testing.T, tt *ui.Tester, a *App, label string) {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatal("missing picker row", label)
	}
	for _, x := range []float32{r.X + 50, r.X + r.W - 10} {
		tt.Move(x, r.Y+r.H/2)
		tt.Frame()
		pixel := tt.Image().RGBAAt(int(r.X+r.W-10), int(r.Y+r.H/2))
		if !nearColor(pixel.R, pixel.G, pixel.B, a.uiTheme.Accent) {
			t.Fatalf("row %q has no hover at %v: got=%v expected=%v", label, x, pixel, a.uiTheme.Accent)
		}
	}
}

func TestPickerDropdownsAnchorAndHoverInEveryTheme(t *testing.T) {
	for _, theme := range themeNames() {
		for _, width := range []int{1380, 1000} {
			a := dropdownFixture()
			a.prefs.Theme = theme
			a.setTheme()
			tt := ui.NewTester(a.view, width, 620)
			if err := tt.Click("Switch repository"); err != nil {
				t.Fatal(err)
			}
			if !a.showRepos {
				t.Fatal("repository button did not open dropdown")
			}
			checkDropdownAnchor(t, tt, "Repository dropdown", "Switch repository", width)
			if tt.HasText("/work/alpha") || tt.HasText("/work/beta") {
				t.Fatal("repository list still shows paths")
			}
			checkPickerHover(t, tt, a, "Repository: beta")
			if theme == "Nord" {
				captureMigrationPreview(t, tt, fmt.Sprintf("repository-dropdown-hover-%d", width))
			}
			// Clicking the other selector dismisses the first popup and opens its own.
			if err := tt.Click("Select branches"); err != nil {
				t.Fatal(err)
			}
			if a.showRepos || !a.showBranches {
				t.Fatal("selector click did not switch dropdowns")
			}
			checkDropdownAnchor(t, tt, "Branch dropdown", "Select branches", width)
			for _, text := range []string{"Show All", "release", "feature/ui"} {
				if !tt.HasText(text) {
					t.Fatal("missing compact branch option", text)
				}
			}
			if tt.HasText("Checkout") || tt.HasText("Filter") {
				t.Fatal("branch picker still contains action-button rows")
			}
			checkPickerHover(t, tt, a, "Branch: feature/ui")
			if theme == "Nord" {
				captureMigrationPreview(t, tt, fmt.Sprintf("branch-dropdown-hover-%d", width))
			}
			if err := tt.RightClick("Branch: feature/ui"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(tt.Menu(), "|"), "Checkout Branch") {
				t.Fatal("compact branch row lost checkout context action")
			}
			tt.CloseMenu()
			tt.Key(0, ui.KeyEscape)
			if a.showBranches {
				t.Fatal("Escape left branch dropdown open")
			}
			tt.Key(ui.Cmd, ui.KeyP)
			if !a.showRepos {
				t.Fatal("Cmd+P did not open repository dropdown")
			}
			tt.ClickAt(float32(width-20), 560)
			if a.showRepos {
				t.Fatal("outside click left dropdown open")
			}
		}
	}
}

func TestBranchDropdownFiltersByKeyboardWithoutCheckout(t *testing.T) {
	repo := testRepo(t)
	head := testCommit(t, repo, "file.txt", "contents\n", "base")
	testGit(t, repo, "branch", "feature/ui")
	h := newActionHarness(t, repo)
	a := h.a
	tt := ui.NewTester(a.view, 1000, 620)
	tt.Key(ui.Cmd, ui.KeyB)
	if !tt.Focused("Search branches") {
		t.Fatal("branch filter not focused")
	}
	tt.Type("feature/ui")
	tt.Key(0, ui.KeyEnter)
	h.drain()
	tt.Frame()
	if a.showBranches || a.filterRef != "refs/heads/feature/ui" {
		t.Fatalf("branch filter not applied: %q", a.filterRef)
	}
	if actual := strings.TrimSpace(testGit(t, repo, "rev-parse", "HEAD")); actual != head {
		t.Fatal("filter changed HEAD")
	}
	if branch := strings.TrimSpace(testGit(t, repo, "branch", "--show-current")); branch != "main" {
		t.Fatal("branch picker checked out a branch")
	}
	tt.Key(ui.Cmd, ui.KeyB)
	if err := tt.Click("Show All"); err != nil {
		t.Fatal(err)
	}
	h.drain()
	if a.filterRef != "" || a.showBranches {
		t.Fatal("Show All did not clear filter")
	}
}

func TestRepositoryDropdownButtonTogglesAndDoesNotShiftGraph(t *testing.T) {
	a := dropdownFixture()
	tt := ui.NewTester(a.view, 1000, 620)
	before, _ := tt.Find("A test commit")
	if err := tt.Click("Switch repository"); err != nil {
		t.Fatal(err)
	}
	after, _ := tt.Find("A test commit")
	if before != after {
		t.Fatal("dropdown shifted graph")
	}
	if err := tt.Click("Switch repository"); err != nil {
		t.Fatal(err)
	}
	if a.showRepos {
		t.Fatal("clicking open selector did not close it")
	}
}

func TestDropdownsRevealCurrentSelectionInLongLists(t *testing.T) {
	a := dropdownFixture()
	a.repos = nil
	a.snapshot.Branches = nil
	for i := 0; i < 45; i++ {
		name := fmt.Sprintf("repo-%02d", i)
		a.repos = append(a.repos, "/work/"+name)
		a.snapshot.Branches = append(a.snapshot.Branches, Branch{Name: name, Full: "refs/heads/" + name})
	}
	a.active = 37
	a.filterRef = "refs/heads/repo-36"
	tt := ui.NewTester(a.view, 1000, 620)
	tt.Key(ui.Cmd, ui.KeyP)
	r, ok := tt.Find("Repository: repo-37")
	panel, _ := tt.Find("Repository dropdown")
	if !ok || r.Y < panel.Y || r.Y+r.H > panel.Y+panel.H {
		t.Fatal("current repository was not revealed in dropdown")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Key(ui.Cmd, ui.KeyB)
	r, ok = tt.Find("Branch: repo-36")
	panel, _ = tt.Find("Branch dropdown")
	if !ok || r.Y < panel.Y || r.Y+r.H > panel.Y+panel.H {
		t.Fatal("selected branch was not revealed in dropdown")
	}
}

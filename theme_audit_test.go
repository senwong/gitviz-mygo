package main

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type themeAuditScene struct {
	name     string
	setup    func(*App)
	interact func(*testing.T, *ui.Tester)
}
type themeAuditCapture struct{ Theme, Scene, Size, File string }

func themeAuditFixture() *App {
	a := searchGraphFixture()
	a.prefs.InlineDetails = true
	a.post = func(func()) {} // Capture only: never execute queued Git operations.
	a.snapshot.Head = a.snapshot.Commits[0].SHA
	a.snapshot.Ancestors = map[string]bool{a.snapshot.Head: true}
	a.snapshot.Remotes = map[string]string{"origin": "git@example.test:team/gitviz.git", "upstream": "git@example.test:upstream/gitviz.git"}
	a.snapshot.Branches = []Branch{{Name: "release", Full: "refs/heads/release", Current: true, SHA: a.snapshot.Head}, {Name: "feature/ui", Full: "refs/heads/feature/ui", SHA: a.snapshot.Head}, {Name: "origin/feature/ui", Full: "refs/remotes/origin/feature/ui", Remote: true, SHA: a.snapshot.Head}}
	a.snapshot.Commits[0].Refs = []string{"release", "origin/feature/ui", "tag:publish-cn-20261112"}
	a.snapshot.Commits[0].Body = "Complete commit message with 中文描述 and https://example.test/ticket/123\n\nCo-Authored-By: Contributor <contributor@example.test>"
	a.snapshot.Commits[0].Committer = "Committer Name"
	a.snapshot.Commits[0].CommitterEmail = "committer@example.test"
	a.snapshot.Tags = map[string]Tag{"publish-cn-20261112": {Name: "publish-cn-20261112", Annotated: true, SHA: a.snapshot.Head}}
	a.snapshot.Files = []FileChange{{Path: "src/app.go", Status: "M"}, {Path: "new.txt", Status: "?", Untracked: true}}
	a.snapshot.Stashes = []Stash{{Ref: "stash@{0}", Message: "WIP on release: pending changes"}}
	a.repos = []string{"/workspace/gitviz", "/workspace/umu_resource", "/workspace/umu_web_apps/apps/umu_profile"}
	a.rebuildVisible()
	return a
}
func themeAuditDetails(a *App) {
	a.selected = 0
	a.listSelected = 0
	a.detailVisible = true
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.files = []FileChange{{Path: "src/app.go", Status: "M", StatsKnown: true, Additions: 12, Deletions: 3}, {Path: "src/components/新增.tsx", Status: "A", StatsKnown: true, Additions: 24}, {Path: "old.txt", Status: "D", StatsKnown: true, Deletions: 9}, {Path: "assets/icon.png", Status: "M", StatsKnown: true, Binary: true}}
	a.config.FileTree = true
	a.rebuildFileRows()
}
func themeAuditScenes() []themeAuditScene {
	click := func(label string) func(*testing.T, *ui.Tester) {
		return func(t *testing.T, tt *ui.Tester) {
			t.Helper()
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
		}
	}
	right := func(label string) func(*testing.T, *ui.Tester) {
		return func(t *testing.T, tt *ui.Tester) {
			t.Helper()
			if err := tt.RightClick(label); err != nil {
				t.Fatal(err)
			}
		}
	}
	scenes := []themeAuditScene{
		{"graph", func(a *App) {}, nil},
		{"welcome", func(a *App) { a.active = -1; a.repos = nil }, nil},
		{"empty-history", func(a *App) { a.snapshot.Commits = nil; a.rebuildVisible() }, nil},
		{"fetching", func(a *App) { a.busy[a.repo()] = "Fetching remotes…"; a.loading = true }, nil},
		{"repositories", func(a *App) { a.openRepositoryPicker() }, nil},
		{"repositories-empty", func(a *App) { a.openRepositoryPicker() }, func(t *testing.T, tt *ui.Tester) { tt.Type("no-such-repo") }},
		{"commands", func(a *App) { a.showCommands = true }, nil},
		{"commands-empty", func(a *App) { a.showCommands = true }, func(t *testing.T, tt *ui.Tester) { tt.Type("no-such-command") }},
		{"commands-prefix", func(a *App) { a.openRepositoryPicker() }, func(t *testing.T, tt *ui.Tester) { tt.Type("> Fetch") }},
		{"branches", func(a *App) { a.showBranches = true }, nil},
		{"branches-empty", func(a *App) { a.showBranches = true }, func(t *testing.T, tt *ui.Tester) { tt.Type("no-such-branch") }},
		{"stashes", func(a *App) { a.showStashes = true }, nil},
		{"stashes-empty", func(a *App) { a.showStashes = true; a.snapshot.Stashes = nil }, nil},
		{"search-matches", func(a *App) { a.showSearch = true; a.query = "202611"; a.rebuildVisible() }, nil},
		{"search-empty", func(a *App) { a.showSearch = true; a.query = "no-matches"; a.rebuildVisible() }, nil},
		{"inline-details", themeAuditDetails, nil},
		{"inline-details-empty", func(a *App) { themeAuditDetails(a); a.files = nil; a.rebuildFileRows() }, nil},
		{"large-diff", func(a *App) {
			themeAuditDetails(a)
			a.largeDiffVisible = true
			a.selectedFile = 0
			a.diffLines = []string{"diff --git a/src/app.go b/src/app.go", "@@ -1,3 +1,3 @@", " package main", "-old := false", "+updated := true", " // 中文注释"}
		}, nil},
		{"diff-loading", func(a *App) { themeAuditDetails(a); a.largeDiffVisible = true; a.diffLoading = true }, nil},
		{"diff-empty", func(a *App) { themeAuditDetails(a); a.largeDiffVisible = true }, nil},
		{"legacy-details", func(a *App) { themeAuditDetails(a); a.prefs.InlineDetails = false }, nil},
		{"error", func(a *App) {
			a.errorText = "Git operation failed: working tree contains local changes.\nPlease commit or stash your changes."
		}, nil},
		{"prompt", func(a *App) { a.ask("Rename Branch", "Enter the new branch name.", "Rename", func(string) {}) }, nil},
		{"checkout-sync", func(a *App) {
			a.syncPrompt = &SyncPrompt{Repo: a.repo(), Tracking: a.snapshot.Tracking}
			a.syncOpen = true
		}, nil},
		{"commit-menu", func(a *App) {}, right("needle merge")},
		{"branch-menu", func(a *App) { a.showBranches = true }, right("feature/ui")},
		{"remote-menu", func(a *App) {
			a.showBranches = true
			a.snapshot.Commits[0].Refs = []string{"release", "tag:publish-cn-20261112"}
		}, right("origin/feature/ui")},
		{"tag-menu", func(a *App) {}, right("publish-cn-20261112")},
		{"file-menu", themeAuditDetails, right("File diff: src/app.go")},
		{"actions-menu", func(a *App) {}, click("Actions")},
		{"theme-options", func(a *App) { a.showSettings = true; a.settingsSection = "Appearance" }, nil},
	}
	// Interaction states use the same native renderer as ordinary application views.
	scenes = append(scenes,
		themeAuditScene{"primary-hover", func(a *App) { a.active = -1; a.repos = nil }, func(t *testing.T, tt *ui.Tester) {
			r, ok := tt.Find("Open repository…")
			if !ok {
				t.Fatal("open button missing")
			}
			tt.Move(r.X+r.W/2, r.Y+r.H/2)
		}},
		themeAuditScene{"primary-pressed", func(a *App) { a.active = -1; a.repos = nil }, func(t *testing.T, tt *ui.Tester) {
			r, ok := tt.Find("Open repository…")
			if !ok {
				t.Fatal("open button missing")
			}
			tt.Press(r.X+r.W/2, r.Y+r.H/2)
		}},
		themeAuditScene{"reset-options", func(a *App) { a.resetForm(a.snapshot.Commits[0]) }, click("Reset Mode")},
		themeAuditScene{"tag-type-options", func(a *App) { a.createTagForm(a.snapshot.Commits[0]) }, click("Type")},
		themeAuditScene{"settings-search", func(a *App) { a.showSettings = true }, func(t *testing.T, tt *ui.Tester) { tt.Type("branch") }},
		themeAuditScene{"settings-search-empty", func(a *App) { a.showSettings = true }, func(t *testing.T, tt *ui.Tester) { tt.Type("no-such-setting") }},
		themeAuditScene{"stash-menu", func(a *App) { a.showStashes = true }, click("…")},
	)

	scenes = append(scenes,
		themeAuditScene{"worktree-details", func(a *App) { themeAuditDetails(a); a.selectedCommit = nil; a.revision = "worktree"; a.selected = -1 }, nil},
		themeAuditScene{"worktree-diff", func(a *App) {
			themeAuditDetails(a)
			a.selectedCommit = nil
			a.revision = "worktree"
			a.selected = -1
			a.largeDiffVisible = true
			a.selectedFile = 0
			a.diffLines = []string{"@@ -1 +1 @@", "-old content", "+working copy changes"}
		}, nil},
		themeAuditScene{"stash-details", func(a *App) {
			themeAuditDetails(a)
			a.selectedCommit = &Commit{SHA: "stash-sha", Subject: "WIP on release: pending changes"}
			a.revision = "stash@{0}"
			a.selected = -1
		}, nil},
		themeAuditScene{"tag-details", func(a *App) {
			a.showActionForm("Tag Details", "object merge-sha\ntype commit\ntag publish-cn-20261112\ntagger Ada <ada@example.test>\n\nRelease notes with 中文说明", "Close", nil, func() error { return nil })
		}, nil},
		themeAuditScene{"wrapped-references", func(a *App) {
			a.snapshot.Commits[0].Refs = []string{"release", "feature/ui", "origin/feature/ui", "tag:publish-cn-20261112", "tag:publish-com-20261112", "tag:bak_publish-cn-20261112_" + strings.Repeat("long-reference-", 10)}
		}, nil},
		themeAuditScene{"repository-manage", func(a *App) { a.openRepositoryPicker() }, click("Manage…")},
		themeAuditScene{"diff-headers", func(a *App) {
			themeAuditDetails(a)
			a.largeDiffVisible = true
			a.selectedFile = 0
			a.diffMetadata = true
			a.diffLines = []string{"diff --git a/src/app.go b/src/app.go", "index abcd..ef01 100644", "--- a/src/app.go", "+++ b/src/app.go", "@@ -1 +1 @@", "-old content", "+new content"}
		}, nil},
		themeAuditScene{"comparison", func(a *App) { themeAuditDetails(a); a.prefs.InlineDetails = false; a.compare = "base-sha" }, nil},
		themeAuditScene{"commit-tooltip", func(a *App) { a.config.ShowCommitTooltip = true }, func(t *testing.T, tt *ui.Tester) {
			r, ok := tt.Find("needle merge")
			if !ok {
				t.Fatal("commit missing")
			}
			tt.Move(r.X+r.W/2, r.Y+r.H/2)
			tt.Advance(time.Second)
		}},
	)
	for _, section := range settingsSections {
		section := section
		scenes = append(scenes, themeAuditScene{"settings-" + section, func(a *App) { a.showSettings = true; a.settingsSection = section }, nil})
		scenes = append(scenes, themeAuditScene{"settings-" + section + "-bottom", func(a *App) { a.showSettings = true; a.settingsSection = section }, func(t *testing.T, tt *ui.Tester) {
			r, ok := tt.Find("Search settings")
			if !ok {
				t.Fatal("settings input missing")
			}
			tt.Scroll(r.X+r.W-50, r.Y+150, 0, 3000)
		}})
	}
	forms := []struct {
		name  string
		setup func(*App)
	}{
		{"checkout", func(a *App) { a.commitCheckoutForm(a.snapshot.Commits[0]) }},
		{"branch-create", func(a *App) { a.createBranchForm(a.snapshot.Commits[0]) }},
		{"tag-create", func(a *App) { a.createTagForm(a.snapshot.Commits[0]) }},
		{"cherry-pick", func(a *App) { a.cherryPickForm(a.snapshot.Commits[0]) }},
		{"revert", func(a *App) { a.revertForm(a.snapshot.Commits[0]) }},
		{"reset", func(a *App) { a.resetForm(a.snapshot.Commits[0]) }},
		{"merge", func(a *App) { a.mergeForm("feature/ui", "feature/ui", "", "") }},
		{"rebase", func(a *App) { a.rebaseForm("feature/ui", "feature/ui") }},
		{"branch-push", func(a *App) { a.pushBranchForm(a.snapshot.Branches[1]) }},
		{"branch-delete", func(a *App) { a.deleteBranchForm(a.snapshot.Branches[1]) }},
		{"remote-checkout", func(a *App) { a.remoteCheckoutForm(a.snapshot.Branches[2]) }},
		{"tag-push", func(a *App) { a.tagPushForm(a.snapshot.Tags["publish-cn-20261112"]) }},
		{"tag-delete", func(a *App) { a.deleteTagForm(a.snapshot.Tags["publish-cn-20261112"]) }},
		{"tag-push-warning", func(a *App) {
			a.confirmTagPush(&tagPushCheck{Tag: a.snapshot.Tags["publish-cn-20261112"], Missing: []string{"origin"}, Remotes: []string{"origin"}})
		}},
		{"validation-error", func(a *App) {
			a.createBranchForm(a.snapshot.Commits[0])
			a.contextForm.Error = "A branch named release already exists. Choose another name."
		}},
	}
	for _, f := range forms {
		scenes = append(scenes, themeAuditScene{"form-" + f.name, f.setup, nil})
	}
	return scenes
}

// TestThemeScreenAudit is an opt-in capture of every application surface in every theme.
// It never submits a Git action and does not read or write the user's preferences.
func TestThemeScreenAudit(t *testing.T) {
	root := os.Getenv("GITVIZ_THEME_AUDIT_DIR")
	if root == "" {
		t.Skip("set GITVIZ_THEME_AUDIT_DIR for the complete screenshot matrix")
	}
	var manifest []themeAuditCapture
	defer func() {
		data, _ := json.MarshalIndent(manifest, "", "  ")
		if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0644); err != nil {
			t.Error(err)
		}
	}()
	for _, name := range themeNames() {
		for _, size := range []struct {
			name string
			w, h int
		}{{"wide", 1380, 900}, {"minimum", 1000, 620}} {
			for _, scene := range themeAuditScenes() {
				t.Run(name+"/"+size.name+"/"+scene.name, func(t *testing.T) {
					a := themeAuditFixture()
					a.prefs.Theme = name
					a.setTheme()
					scene.setup(a)
					tt := ui.NewTester(a.view, size.w, size.h)
					if scene.name == "theme-options" {
						if err := tt.Click(name); err != nil {
							t.Fatal(err)
						}
					} else if scene.interact != nil {
						scene.interact(t, tt)
					}
					tt.Frame()
					file := filepath.Join(strings.ReplaceAll(name, " ", "-"), size.name, scene.name+".png")
					path := filepath.Join(root, file)
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(path)
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					f.Close()
					if err != nil {
						t.Fatal(err)
					}
					manifest = append(manifest, themeAuditCapture{name, scene.name, size.name, file})
					if scene.name == "large-diff" {
						r, ok := tt.Find("Diff viewport")
						if !ok || r.H < 300 {
							t.Fatalf("diff viewport clipped: %+v", r)
						}
					}
					if a.contextForm != nil {
						if _, ok := tt.Find("Confirm " + a.contextForm.Confirm); !ok {
							t.Fatal("dialog confirm action missing")
						}
					}
				})
			}
		}
	}
	t.Logf("Captured %d theme/screen/size combinations", len(manifest))
}

func TestThemeContrastInventory(t *testing.T) {
	for _, name := range themeNames() {
		a := uiFixture()
		a.prefs.Theme = name
		a.setTheme()
		th := a.uiTheme
		for _, fg := range []struct {
			name  string
			color ui.Color
		}{{"text", th.Text}, {"muted", th.TextMuted}, {"success", th.Success}, {"warning", th.Warning}, {"danger", th.Danger}, {"accent", th.Accent}} {
			for _, bg := range []struct {
				name  string
				color ui.Color
			}{{"background", th.Background}, {"surface", th.Surface}} {
				l1, l2 := colorLuminance(fg.color), colorLuminance(bg.color)
				if l1 < l2 {
					l1, l2 = l2, l1
				}
				ratio := (l1 + .05) / (l2 + .05)
				if ratio < 4.5 {
					t.Errorf("%s %s/%s: %.2f", name, fg.name, bg.name, ratio)
				}
			}
		}
	}
}

func TestEveryThemeButtonStateAndSelectionContrast(t *testing.T) {
	for _, name := range themeNames() {
		a := uiFixture()
		a.prefs.Theme = name
		a.setTheme()
		th := a.uiTheme
		for _, state := range []ui.Color{th.Accent, th.AccentHover, th.AccentPressed, th.InactiveAccent} {
			if contrastRatio(th.AccentText, state) < 4.5 {
				t.Errorf("%s primary state has unreadable text", name)
			}
		}
		for _, fg := range []ui.Color{th.Text, th.TextMuted, th.Success, th.Warning, th.Danger} {
			if contrastRatio(fg, a.listTheme.Accent) < 4.5 {
				t.Errorf("%s selected row text is unreadable", name)
			}
		}
		before := *th
		a.prefs.Theme = "Shadcn Light"
		a.setTheme()
		a.prefs.Theme = name
		a.setTheme()
		if !reflect.DeepEqual(before, *a.uiTheme) {
			t.Errorf("%s changes colors depending on the previous theme", name)
		}
	}
}

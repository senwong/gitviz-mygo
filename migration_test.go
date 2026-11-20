package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEveryLegacySettingHasAnEditorAndRoundTrips(t *testing.T) {
	config := defaultConfig()
	fields := configFields(&config)
	specs := map[string]bool{}
	for _, spec := range settingSpecs {
		if specs[spec.Key] {
			t.Fatalf("duplicate setting %s", spec.Key)
		}
		if !fields[spec.Key].IsValid() {
			t.Fatalf("unknown setting %s", spec.Key)
		}
		specs[spec.Key] = true
	}
	for key, field := range fields {
		if !specs[key] {
			t.Fatalf("legacy setting has no editor: %s", key)
		}
		switch field.Kind() {
		case reflect.Bool:
			field.SetBool(!field.Bool())
		case reflect.Float64:
			field.SetFloat(field.Float() + 12)
		case reflect.Slice:
			field.Set(reflect.ValueOf([]string{"value-one", "value-two"}))
		case reflect.String:
			field.SetString("custom-value")
		case reflect.Int:
			field.SetInt(2)
		}
	}
	config.GraphStyle = "angular"
	config.RepoOrder = "given"
	config.Extra = map[string]string{"future-option": "preserve=this"}
	loaded := parseConfig(serializeConfig(config))
	if !reflect.DeepEqual(config, loaded) {
		t.Fatalf("round trip lost values\n%+v\n%+v", config, loaded)
	}
	if fields := configFields(&loaded); len(fields) != len(settingSpecs) {
		t.Fatal("settings coverage is incomplete")
	}
}

func TestLegacyConfigImportDoesNotModifySourceAndExportPreservesUnknowns(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".gitviz.conf")
	text := "# preserved source\nfile_tree=true\nfirst_parent=true\nauthor_width=177\nfuture-value=keep\n"
	os.WriteFile(path, []byte(text), 0644)
	a := newApp(Preferences{Dark: true, Remotes: true, Detail: true})
	a.repos = []string{repo}
	a.active = 0
	a.prefsFile = filepath.Join(t.TempDir(), "settings.json")
	a.config = a.configForRepo(repo)
	if !a.config.FileTree || !a.config.FirstParent || a.config.AuthorWidth != 177 {
		t.Fatal("legacy values not imported")
	}
	a.config.FileTree = false
	a.saveConfig(false)
	data, _ := os.ReadFile(path)
	if string(data) != text {
		t.Fatal("saving preferences modified the original configuration")
	}
	a.exportConfig()
	data, _ = os.ReadFile(path)
	exported := parseConfig(string(data))
	if exported.FileTree || exported.Extra["future-value"] != "keep" {
		t.Fatalf("export lost values %+v", exported)
	}
}

func TestFileTreeCompactionAndCollapse(t *testing.T) {
	files := []FileChange{{Path: "src/deep/a.txt"}, {Path: "src/deep/b.txt"}, {Path: "README"}}
	rows := buildFileRows(files, true, true, map[string]bool{})
	if len(rows) != 4 || rows[0].Name != "src/deep" || !rows[0].Dir {
		t.Fatalf("compact tree %+v", rows)
	}
	rows = buildFileRows(files, true, true, map[string]bool{"src/deep": true})
	if len(rows) != 2 || rows[1].Path != "README" {
		t.Fatalf("collapse %+v", rows)
	}
	if rows = buildFileRows(files, false, true, nil); len(rows) != 3 || rows[0].Dir {
		t.Fatalf("flat list %+v", rows)
	}
}

func TestQuerySettingsMailmapFirstParentTagOnlyAndStatus(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	testCommit(t, repo, "base", "base", "base")
	testGit(t, repo, "switch", "-c", "feature")
	testCommit(t, repo, "feature", "feature", "feature")
	testGit(t, repo, "switch", "main")
	testCommit(t, repo, "main", "main", "main")
	testGit(t, repo, "merge", "--no-ff", "--no-edit", "feature")
	config := defaultConfig()
	config.FirstParent = true
	commits, _, err := readLogConfig(ctx, repo, 500, config, "refs/heads/main")
	if err != nil || len(commits) != 3 {
		t.Fatalf("first-parent result %d %v", len(commits), err)
	}
	testCommit(t, repo, ".mailmap", "Canonical <canonical@example.test> Gitviz Test <gitviz@example.test>\n", "mailmap")
	config.UseMailmap = true
	commits, _, err = readLogConfig(ctx, repo, 500, config, "")
	if err != nil || commits[0].Author != "Canonical" {
		t.Fatalf("mailmap %+v %v", commits, err)
	}
	os.WriteFile(filepath.Join(repo, "untracked"), []byte("new"), 0644)
	config.IncludeUntracked = false
	config.DimNonHeadCommits = true
	s, err := loadSnapshotConfig(ctx, repo, 500, config, "")
	if err != nil || len(s.Files) != 0 || !s.Ancestors[s.Head] {
		t.Fatalf("status or ancestry %+v %v", s, err)
	}
	testGit(t, repo, "switch", "--orphan", "only-tag")
	testCommit(t, repo, "tag-file", "tag", "tag-only")
	testGit(t, repo, "tag", "special")
	testGit(t, repo, "switch", "main")
	testGit(t, repo, "branch", "-D", "only-tag")
	config.OnlyTags = true
	config.UseMailmap = false
	commits, _, err = readLogConfig(ctx, repo, 500, config, "")
	if err != nil || len(commits) != 1 || commits[0].Subject != "tag-only" {
		t.Fatalf("tag-only %+v %v", commits, err)
	}
}

func TestDisplaySettingsAndProviders(t *testing.T) {
	config := defaultConfig()
	config.CombineRefs = true
	config.FullRefs = true
	refs := decorateRefs([]string{"feature/a", "origin/feature/a", "tag:v1"}, config, []Branch{{Name: "feature/a"}, {Name: "origin/feature/a", Remote: true}})
	if refs[0] != "refs/heads/feature/a\x1forigin" || refs[1] != "tag:refs/tags/v1" {
		t.Fatalf("refs %+v", refs)
	}
	if !branchMatches("feature/deep/a", []string{"heads/feature/*"}) || branchMatches("fix/a", []string{"heads/feature/*"}) {
		t.Fatal("glob semantics mismatch")
	}
	spans := messageSpans("**bold** _italic_ `code`", true)
	bold, italic, code := false, false, false
	for _, s := range spans {
		bold = bold || s.Weight == 700
		italic = italic || s.Italic
		code = code || s.Font == "Menlo"
	}
	if !bold || !italic || !code {
		t.Fatalf("markdown %+v", spans)
	}
	a := uiFixture()
	a.config.EmojiMappings = []string{"shipit:🚢"}
	if got := a.messageText(":shipit: :bug:"); got != "🚢 🐛" {
		t.Fatalf("emoji %q", got)
	}
	link := providerLink("git@gitlab.example.com:team/project.git", "", "release", "feature/test", "", true)
	if !strings.Contains(link, "/-/merge_requests/new") || !strings.Contains(link, "feature%2Ftest") {
		t.Fatalf("provider %q", link)
	}
	if link = providerLink("git@github.com:team/project.git", "https://bugs.test/{repo}/{issue}", "", "", "123", false); link != "https://bugs.test/project/123" {
		t.Fatalf("custom provider %q", link)
	}
	if providerLink("git@github.com:team/project.git", "javascript:bad", "", "", "1", false) != "" {
		t.Fatal("unsafe link accepted")
	}
	mods, key := shortcutParts("cmd-shift-p")
	if mods != ui.Cmd|ui.Shift || key != ui.KeyP {
		t.Fatal("shortcut parsing failed")
	}
	if len(themeNames()) != 14 {
		t.Fatal("not all legacy themes migrated")
	}
}

func TestNativeDetailColumnSettings(t *testing.T) {
	a := uiFixture()
	a.prefsFile = filepath.Join(t.TempDir(), "prefs.json")
	a.selectedCommit = &a.snapshot.Commits[0]
	a.revision = a.selectedCommit.SHA
	a.files = []FileChange{{Path: "src/a.go", Status: "M"}}
	a.rebuildFileRows()
	a.detailVisible = true
	a.showSettings = true
	a.settingsSection = "Columns"
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.Click("Author column"); err != nil {
		t.Fatal(err)
	}
	if a.config.ColumnsAuthor {
		t.Fatal("column switch did not change setting")
	}
	a.showSettings = false
	tt.Frame()
	if tt.HasText("Author") {
		t.Fatal("column setting did not change rendering")
	}
}

func captureMigrationPreview(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("GITVIZ_TEST_ARTIFACT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "gitviz-mygo-"+name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNativeHiddenActionsAndDetailPreference(t *testing.T) {
	a := uiFixture()
	a.config.HiddenActions = []string{"revert"}
	tt := ui.NewTester(a.view, 1380, 900)
	if err := tt.RightClick("A test commit"); err != nil {
		t.Fatal(err)
	}
	labels := strings.Join(tt.Menu(), "|")
	if strings.Contains(labels, "Revert") {
		t.Fatalf("hidden action is visible: %s", labels)
	}
	if !strings.Contains(labels, "Cherry Pick") {
		t.Fatalf("other actions vanished: %s", labels)
	}
	tt.CloseMenu()
	a.config.ShowDetail = false
	a.post = func(func()) {}
	a.selectCommit(0)
	if a.detailVisible {
		t.Fatal("selecting a commit ignored the detail visibility setting")
	}
	if a.cancelDetail != nil {
		a.cancelDetail()
	}
}

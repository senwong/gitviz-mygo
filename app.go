package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type Prompt struct {
	Title, Description, Input, Confirm string
	Run                                func(string)
}

type SyncPrompt struct {
	Repo     string
	Tracking Tracking
}

type App struct {
	remoteStates                                                                 map[string]RemoteStatus
	remoteVersions                                                               map[string]uint64
	repositoryAnchor, branchesAnchor                                             ui.Element
	branchCursor                                                                 int
	searchDiff                                                                   bool
	diffQuery, diffFindQuery, diffFindFile                                       string
	diffFindSource                                                               []string
	diffMatches                                                                  []diffSearchMatch
	diffMatchLines                                                               map[int][]int
	diffMatchCursor                                                              int
	quitting                                                                     bool
	searchMatches                                                                []int
	searchMatched                                                                map[int]bool
	searchCurrent                                                                string
	focusCommitList                                                              bool
	compactDiffSource, compactDiffCache                                          []string
	messageExpanded                                                              bool
	diffMetadata                                                                 bool
	openExternal                                                                 func(string) error
	clipboardWrite                                                               func(string)
	contextForm                                                                  *actionForm
	searchFocused                                                                bool
	branchesWereOpen, settingsWereOpen, promptWasOpen                            bool
	commandQuery                                                                 string
	commandInitialQuery                                                          string
	commandCursor                                                                int
	commandsWereOpen                                                             bool
	repoCursor                                                                   int
	repoList                                                                     ui.ListState
	repoPickerWasOpen                                                            bool
	repoNames                                                                    map[string]string
	settingsEdits                                                                map[string]string
	prefsFile                                                                    string
	config                                                                       RepoConfig
	configRepo                                                                   string
	settingsQuery, settingsSection                                               string
	collapsed                                                                    map[string]bool
	fileRows                                                                     []FileRow
	fileRowSelected                                                              int
	win                                                                          *mygo.Window
	prefs                                                                        Preferences
	roots, repos                                                                 []string
	active                                                                       int
	snapshot                                                                     Snapshot
	loading, detailLoading, diffLoading                                          bool
	loadGen, detailGen, diffGen                                                  uint64
	cancelLoad, cancelDetail, cancelDiff                                         context.CancelFunc
	busy                                                                         map[string]string
	errorText                                                                    string
	query, repoQuery, branchQuery, filterRef                                     string
	lastQuery                                                                    string
	visible                                                                      []int
	selected                                                                     int
	revision                                                                     string
	selectedCommit                                                               *Commit
	files                                                                        []FileChange
	selectedFile                                                                 int
	diffText                                                                     string
	imageDiff                                                                    *imageFileDiff
	imageDiffText                                                                bool
	compare                                                                      string
	list, branchList, fileList                                                   ui.ListState
	diffScroll                                                                   ui.ScrollState
	diffScrollTarget                                                             *int
	listSelected                                                                 int
	showSearch, showRepos, showBranches, showSettings, showCommands, showStashes bool
	detailVisible                                                                bool
	detailMaximized                                                              bool
	largeDiffVisible                                                             bool
	remoteRefreshRepo                                                            string
	autoFetchErrors                                                              map[string]string
	prompt                                                                       *Prompt
	promptOpen                                                                   bool
	syncPrompt                                                                   *SyncPrompt
	syncOpen                                                                     bool
	detailPercent                                                                float32
	fileWidth                                                                    float32
	diffLines                                                                    []string
	matchCursor                                                                  int
	post                                                                         func(func())
	screenshot                                                                   string
	screenshotDone                                                               bool
	listTheme                                                                    *ui.Theme
	uiTheme                                                                      *ui.Theme
}

func newApp(prefs Preferences) *App {
	if prefs.RepositoryShortcut == "" {
		prefs.RepositoryShortcut = "cmd-p"
	}
	a := &App{prefs: prefs, roots: prefs.Roots, busy: map[string]string{}, active: -1, selected: -1, selectedFile: -1, listSelected: -1, detailVisible: prefs.Detail, detailPercent: 40, fileWidth: 240, autoFetchErrors: map[string]string{}}
	a.prefsFile = prefsPath()
	a.config = defaultConfig()
	a.config.Remotes = prefs.Remotes
	a.config.ShowDetail = prefs.Detail
	a.collapsed = map[string]bool{}
	a.setTheme()
	return a
}

func (a *App) setTheme() {
	if a.prefs.Theme == "MyGo Dark" {
		a.prefs.Dark = true
	} else if a.prefs.Theme == "MyGo Light" {
		a.prefs.Dark = false
	}
	if a.prefs.Dark {
		a.uiTheme = ui.DarkTheme()
	} else {
		a.uiTheme = ui.LightTheme()
	}
	if a.prefs.Dark {
		a.uiTheme.Background = ui.Hex("#171717")
		a.uiTheme.Surface = ui.Hex("#1e1e1e")
		a.uiTheme.Border = ui.Hex("#333333")
		a.uiTheme.Text = ui.Hex("#cccccc")
		a.uiTheme.TextMuted = ui.Hex("#858585")
		a.uiTheme.Accent = ui.Hex("#0085d9")
		a.uiTheme.AccentText = ui.Hex("#101723")
	}
	var selectedColor ui.Color
	if named, color, ok := namedTheme(a.prefs.Theme); ok {
		a.uiTheme = named
		a.prefs.Dark = named.Dark
		selectedColor = color
	}
	a.uiTheme.FontSize = 13
	selected := *a.uiTheme
	selected.Radius = 0
	if a.prefs.Dark {
		selected.Accent = ui.Hex("#252525")
	} else {
		selected.Accent = ui.Hex("#dce9fb")
	}
	if selectedColor != ui.Transparent {
		selected.Accent = selectedColor
	}
	selected.Accent = readableThemeColor(selected.Accent, a.uiTheme.Background, a.uiTheme.Text)
	polishTheme(a.uiTheme, selected.Accent)
	selection := selected.Accent
	selected = *a.uiTheme
	selected.Radius = 0
	selected.Accent = selection
	selected.AccentText = a.uiTheme.Text
	selected.InactiveAccent = selected.Accent
	a.listTheme = &selected
	if a.win != nil {
		mygo.Theme.SetSource(a.nativeThemeSource())
		a.syncWindowChrome()
	}
}

func (a *App) repo() string {
	if a.active >= 0 && a.active < len(a.repos) {
		return a.repos[a.active]
	}
	return ""
}
func (a *App) update(fn func()) {
	if a.post != nil {
		a.post(fn)
	} else if a.win != nil {
		a.win.Update(fn)
	}
}
func (a *App) fail(err error) {
	if err != nil {
		a.errorText = err.Error()
	}
}

func (a *App) persist() {
	a.prefs.LastRepository = a.repo()
	a.prefs.Roots = append([]string(nil), a.roots...)
	a.prefs.Detail = a.detailVisible
	if err := savePreferencesAt(a.prefsFile, a.prefs); err != nil {
		a.fail(err)
	}
}

func (a *App) discoverRoots(roots []string) {
	a.loadGen++
	gen := a.loadGen
	if a.cancelLoad != nil {
		a.cancelLoad()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.cancelLoad = cancel
	a.loading = true
	depth := a.prefs.DiscoveryDepth
	go func() {
		defer cancel()
		repos, err := discoverDepth(ctx, roots, depth)
		names := workspaceNames(roots)
		a.update(func() {
			if gen != a.loadGen {
				return
			}
			a.loading = false
			if err != nil {
				a.fail(err)
				return
			}
			previous := a.repo()
			preferred := previous
			if preferred == "" {
				preferred = a.prefs.LastRepository
			}
			filtered := repos[:0]
			for _, repo := range repos {
				excluded := false
				for _, skip := range a.prefs.Excluded {
					if repo == skip {
						excluded = true
						break
					}
				}
				if !excluded {
					filtered = append(filtered, repo)
				}
			}
			repos = filtered
			a.repos = repos
			a.repoNames = names
			a.active = -1
			for i, p := range repos {
				if p == preferred {
					a.active = i
					break
				}
			}
			if a.active < 0 && len(repos) > 0 {
				a.active = 0
			}
			a.roots = append([]string(nil), roots...)
			a.persist()
			if previous != a.repo() {
				a.remoteRefreshRepo = a.repo()
			}
			a.load(false)
		})
	}()
}

func (a *App) switchRepo(index int) {
	if index < 0 || index >= len(a.repos) {
		return
	}
	if a.repo() == a.repos[index] {
		a.showRepos = false
		return
	}
	a.active = index
	a.detailMaximized = false
	a.remoteRefreshRepo = a.repo()
	a.filterRef = ""
	a.contextForm = nil
	a.query = ""
	a.showRepos = false
	a.persist()
	a.load(false)
}

func (a *App) load(preserve bool) { a.loadWithScroll(preserve, false) }

func (a *App) loadWithScroll(preserve, scrollToTop bool) {
	if a.cancelLoad != nil {
		a.cancelLoad()
	}
	if a.cancelDetail != nil {
		a.cancelDetail()
	}
	if a.cancelDiff != nil {
		a.cancelDiff()
	}
	a.loadGen++
	a.detailGen++
	a.diffGen++
	gen := a.loadGen
	a.syncPrompt = nil
	a.syncOpen = false
	repo := a.repo()
	if repo == "" {
		a.updateWindowTitle()
		a.loading = false
		return
	}
	if a.configRepo != repo {
		a.config = a.configForRepo(repo)
		a.configRepo = repo
		a.detailVisible = a.config.ShowDetail
	}
	a.sortRepos(repo)
	selected := a.revision
	if !preserve {
		a.snapshot = Snapshot{}
		a.visible = nil
		a.clearDetail()
		a.selected = -1
		a.list = ui.ListState{}
		a.listSelected = -1
	}
	a.updateWindowTitle()
	a.loading = true
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	a.cancelLoad = cancel
	limit := max(500, len(a.snapshot.Commits))
	config, ref := cloneConfig(a.config), a.filterRef
	go func() {
		defer cancel()
		s, err := loadSnapshotConfig(ctx, repo, limit, config, ref)
		a.update(func() {
			if gen != a.loadGen || repo != a.repo() {
				return
			}
			a.loading = false
			if err != nil {
				a.fail(err)
				return
			}
			a.snapshot = s
			a.updateWindowTitle()

			a.rebuildVisible()
			if preserve && selected != "" {
				a.selectRevision(selected)
			} else if len(s.Commits) > 0 {
				index := 0
				if a.config.ScrollToHead {
					for i, c := range s.Commits {
						if c.SHA == s.Head {
							index = i
							break
						}
					}
				}
				a.selectCommit(index)
				if a.config.ScrollToHead {
					a.list.ScrollIntoView(a.listSelected)
				}
			}
			if scrollToTop {
				a.list.ScrollTo(0, ui.Start)
			}
			a.scheduleCapture()
			if a.remoteRefreshRepo == repo {
				a.remoteRefreshRepo = ""
				if a.prefs.AutoFetchOnSwitch {
					a.autoFetchOnSwitch()
				} else {
					a.checkRemoteUpdates()
				}
			}
		})
	}()
}

func (a *App) loadMore() {
	if a.loading || !a.snapshot.More {
		return
	}
	gen, repo := a.loadGen, a.repo()
	limit := len(a.snapshot.Commits) + 500
	config, ref := cloneConfig(a.config), a.filterRef
	a.loading = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		s, err := loadSnapshotConfig(ctx, repo, limit, config, ref)
		a.update(func() {
			if gen != a.loadGen || repo != a.repo() {
				return
			}
			a.loading = false
			if err != nil {
				a.fail(err)
				return
			}
			a.snapshot = s
			a.updateWindowTitle()

			a.rebuildVisible()
		})
	}()
}

func (a *App) rebuildVisible() {
	changed := a.lastQuery != a.query
	previous := a.searchCurrent
	if changed {
		previous = ""
	}
	a.visible = make([]int, len(a.snapshot.Commits))
	a.searchMatches = nil
	a.searchMatched = map[int]bool{}
	q := strings.ToLower(strings.TrimSpace(a.query))
	a.matchCursor = -1
	for i, commit := range a.snapshot.Commits {
		a.visible[i] = i
		if q == "" {
			continue
		}
		text := commit.Subject + "\n" + commit.Body + "\n" + commit.Author + "\n" + commit.Email + "\n" + commit.SHA + "\n" + referenceSearchText(commit.Refs)
		if q != "" && strings.Contains(strings.ToLower(text), q) {
			if commit.SHA == previous {
				a.matchCursor = len(a.searchMatches)
			}
			a.searchMatches = append(a.searchMatches, i)
			a.searchMatched[i] = true
		}
	}
	a.searchCurrent = ""
	if len(a.searchMatches) > 0 {
		if a.matchCursor < 0 {
			a.matchCursor = 0
		}
		index := a.searchMatches[a.matchCursor]
		a.searchCurrent = a.snapshot.Commits[index].SHA
		if changed {
			a.list.ScrollIntoView(index)
		}
	}
	a.lastQuery = a.query
	a.listSelected = a.selected
	if a.selected < 0 || a.selected >= len(a.snapshot.Commits) {
		a.listSelected = -1
	}
}

func (a *App) moveSearchMatch(delta int) {
	if a.query != a.lastQuery {
		a.rebuildVisible()
	}
	if len(a.searchMatches) == 0 {
		return
	}
	a.matchCursor = (a.matchCursor + delta + len(a.searchMatches)) % len(a.searchMatches)
	index := a.searchMatches[a.matchCursor]
	a.searchCurrent = a.snapshot.Commits[index].SHA
	a.selectCommit(index)
	a.list.ScrollIntoView(index)
}

func (a *App) clearDetail() {
	a.largeDiffVisible = false
	a.detailGen++
	a.diffGen++
	if a.cancelDetail != nil {
		a.cancelDetail()
	}
	if a.cancelDiff != nil {
		a.cancelDiff()
	}
	a.messageExpanded = false
	a.revision = ""
	a.selectedCommit = nil
	a.files = nil
	a.diffText = ""
	a.imageDiff = nil
	a.imageDiffText = false
	a.diffLines = nil
	a.compactDiffSource = nil
	a.compactDiffCache = nil
	a.selectedFile = -1
	a.fileRows = nil
	a.fileList = ui.ListState{}
	a.detailLoading = false
	a.diffLoading = false
	a.compare = ""
}

func (a *App) toggleCommit(index int) {
	if index < 0 || index >= len(a.snapshot.Commits) {
		return
	}
	if a.detailVisible && a.revision == a.snapshot.Commits[index].SHA {
		a.detailMaximized = false
		a.closeInlineDetails()
		return
	}
	a.openCommit(index)
}

func (a *App) openCommit(index int) {
	if index < 0 || index >= len(a.snapshot.Commits) {
		return
	}
	hidden := !a.config.ShowDetail
	a.config.ShowDetail = true
	a.detailVisible = true
	if a.selectedCommit == nil || a.revision != a.snapshot.Commits[index].SHA {
		a.selectCommit(index)
	}
	if hidden {
		a.saveConfig(false)
	}
	a.focusCommitList = true
}

func (a *App) selectCommit(index int) {
	if index < 0 || index >= len(a.snapshot.Commits) {
		return
	}
	compare := a.compare
	a.clearDetail()
	a.compare = compare
	a.selected = index
	a.detailVisible = a.config.ShowDetail
	c := a.snapshot.Commits[index]
	a.selectedCommit = &c
	a.revision = c.SHA
	for i, j := range a.visible {
		if j == index {
			a.listSelected = i
			break
		}
	}
	a.loadFiles()
}

func (a *App) selectRevision(revision string) {
	if revision == "worktree" {
		a.selectWorktree()
		return
	}
	for i, c := range a.snapshot.Commits {
		if c.SHA == revision {
			a.selectCommit(i)
			return
		}
	}
	for _, s := range a.snapshot.Stashes {
		if s.Ref == revision {
			a.selectStash(s)
			return
		}
	}
	a.clearDetail()
	a.selected = -1
}

func (a *App) selectWorktree() {
	a.clearDetail()
	a.selected = -1
	a.listSelected = -1
	a.revision = "worktree"
	a.detailVisible = a.config.ShowDetail
	a.files = append([]FileChange(nil), a.snapshot.Files...)
	a.rebuildFileRows()
}

func (a *App) selectStash(stash Stash) {
	a.clearDetail()
	a.selected = -1
	a.revision = stash.Ref
	a.detailVisible = a.config.ShowDetail
	a.selectedCommit = &Commit{SHA: stash.SHA, Subject: stash.Message}
	a.showStashes = false
	a.loadFiles()
}

func (a *App) loadFiles() {
	if a.cancelDetail != nil {
		a.cancelDetail()
	}
	a.detailGen++
	gen := a.detailGen
	repo, revision, compare := a.repo(), a.revision, a.compare
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.cancelDetail = cancel
	a.detailLoading = true
	go func() {
		defer cancel()
		var files []FileChange
		var err error
		if compare != "" {
			files, err = compareFiles(ctx, repo, compare, revision)
		} else {
			ref := revision
			if strings.HasPrefix(revision, "stash@{") {
				files, err = compareFiles(ctx, repo, revision+"^1", revision)
			} else {
				files, err = changedFiles(ctx, repo, ref)
			}
		}
		a.update(func() {
			if gen != a.detailGen || repo != a.repo() {
				return
			}
			a.detailLoading = false
			if err != nil {
				a.fail(err)
				return
			}
			a.files = files
			a.rebuildFileRows()
			if len(files) > 0 && !a.prefs.InlineDetails {
				a.selectFile(0)
			}
		})
	}()
}

func (a *App) selectFile(index int) {
	if index < 0 || index >= len(a.files) {
		return
	}
	if a.cancelDiff != nil {
		a.cancelDiff()
	}
	a.diffGen++
	gen := a.diffGen
	a.selectedFile = index
	for i, row := range a.fileRows {
		if row.File == index && !row.Dir {
			a.fileRowSelected = i
			break
		}
	}
	a.diffLoading = true
	a.diffLines = nil
	a.compactDiffSource = nil
	a.compactDiffCache = nil
	a.diffText = ""
	a.imageDiff = nil
	a.imageDiffText = false
	repo, revision, compare, file := a.repo(), a.revision, a.compare, a.files[index]
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.cancelDiff = cancel
	go func() {
		defer cancel()
		text, err := fileDiff(ctx, repo, revision, compare, file)
		var preview *imageFileDiff
		if err == nil {
			preview = loadImageDiff(ctx, repo, revision, compare, file)
		}
		a.update(func() {
			if gen != a.diffGen || repo != a.repo() {
				return
			}
			a.diffLoading = false
			if err != nil {
				a.fail(err)
				return
			}
			a.diffText = text
			a.imageDiff = preview
			a.diffLines = strings.Split(text, "\n")
			a.diffScroll = ui.ScrollState{}
			a.diffScrollTarget = nil
		})
	}()
}

func (a *App) operate(label string, fn func(context.Context, string) error, checkout bool) {
	repo := a.repo()
	if repo == "" {
		return
	}
	if a.busy[repo] != "" {
		return
	}
	a.busy[repo] = label
	a.errorText = ""
	a.showBranches = false
	a.showCommands = false
	a.showStashes = false
	a.syncPrompt = nil
	a.syncOpen = false
	a.prompt = nil
	a.promptOpen = false
	if a.cancelLoad != nil {
		a.cancelLoad()
	}
	a.loadGen++
	a.loading = false
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		err := fn(ctx, repo)
		var tracking Tracking
		if err == nil && checkout {
			branches, _, _ := readBranches(ctx, repo)
			tracking = readTracking(ctx, repo, branches)
		}
		a.update(func() {
			delete(a.busy, repo)
			if repo != a.repo() {
				return
			}
			a.load(true)
			if err != nil {
				var check *tagPushCheck
				if errors.As(err, &check) {
					a.confirmTagPush(check)
				} else {
					a.fail(err)
				}
				return
			}
			if checkout && tracking.Upstream != "" && !tracking.Gone && (tracking.Ahead > 0 || tracking.Behind > 0) {
				a.syncPrompt = &SyncPrompt{Repo: repo, Tracking: tracking}
				a.syncOpen = true
			}
		})
	}()
}

func (a *App) checkout(b Branch) {
	a.operate("Checking out "+b.Name+"…", func(ctx context.Context, repo string) error { return checkoutBranch(ctx, repo, b) }, true)
}

func (a *App) pull(rebase bool) {
	sync := a.syncPrompt
	branch := a.snapshot.Tracking.Branch
	if sync != nil {
		if sync.Repo != a.repo() {
			a.syncOpen = false
			a.syncPrompt = nil
			return
		}
		branch = sync.Tracking.Branch
	}
	label := "Pulling…"
	if rebase {
		label = "Pulling with rebase…"
	}
	a.operate(label, func(ctx context.Context, repo string) error { return pullBranch(ctx, repo, branch, rebase) }, false)
}

func (a *App) runGit(label string, args ...string) {
	a.operate(label, func(ctx context.Context, repo string) error { _, err := gitCommand(ctx, repo, args...); return err }, false)
}

func (a *App) ask(title, description, confirm string, run func(string)) {
	a.prompt = &Prompt{Title: title, Description: description, Confirm: confirm, Run: run}
	a.promptOpen = true
}

func (a *App) openRepository(workspace bool) {
	go func() {
		opts := mygo.OpenDialogOptions{Parent: a.win, Directory: !workspace, Multiple: !workspace, Title: "Open repository"}
		if workspace {
			opts.Title = "Open workspace"
			opts.Filters = []mygo.FileFilter{{Name: "Workspace", Extensions: []string{"gitviz-workspace", "code-workspace", "workspace"}}}
		}
		paths, err := mygo.Dialog.Open(opts)
		a.update(func() {
			if err != nil {
				a.fail(err)
				return
			}
			if len(paths) > 0 {
				for _, path := range paths {
					canonical, _ := filepath.EvalSymlinks(path)
					kept := a.prefs.Excluded[:0]
					for _, skip := range a.prefs.Excluded {
						if skip != canonical {
							kept = append(kept, skip)
						}
					}
					a.prefs.Excluded = kept
				}
				roots := append(append([]string(nil), a.roots...), paths...)
				a.discoverRoots(roots)
			}
		})
	}()
}

func (a *App) scheduleCapture() {
	if a.screenshot == "" || a.screenshotDone || a.win == nil {
		return
	}
	a.screenshotDone = true
	win, path := a.win, a.screenshot
	go func() {
		time.Sleep(750 * time.Millisecond)
		data, err := win.CapturePage()
		if err == nil {
			err = os.WriteFile(path, data, 0644)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Println("Screenshot saved:", path)
		}
		mygo.App.Quit()
	}()
}

func (a *App) saveWorkspace() {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, Title: "Save workspace", DefaultPath: "gitviz.gitviz-workspace"})
		a.update(func() {
			if err != nil {
				a.fail(err)
				return
			}
			if path != "" {
				a.fail(os.WriteFile(path, []byte(strings.Join(a.roots, "\n")+"\n"), 0644))
			}
		})
	}()
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
func repoName(path string) string { return filepath.Base(path) }

func (a *App) nativeThemeSource() mygo.ThemeSource {
	if a.uiTheme.Dark {
		return mygo.ThemeDark
	}
	return mygo.ThemeLight
}

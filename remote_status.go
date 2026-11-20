package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// A successful check describes server refs at CheckedAt; it is never inferred
// from the local ahead/behind count or FETCH_HEAD's modification time.
type RemoteStatus struct {
	Checking                        bool
	CheckedAt, FetchedAt            time.Time
	Branches, Tags, RemovedBranches int
	FetchedBranches, FetchedTags    int
	Error                           string
}

func (s RemoteStatus) fresh(now time.Time) bool {
	return !s.Checking && s.Error == "" && !s.CheckedAt.IsZero() && now.Sub(s.CheckedAt) >= 0 && now.Sub(s.CheckedAt) < 5*time.Minute
}

func readRefTips(ctx context.Context, repo string) (map[string]string, error) {
	out, err := gitCommand(ctx, repo, "for-each-ref", "--format=%(refname)%09%(objectname)%09%(symref)")
	if err != nil {
		return nil, err
	}
	refs := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) >= 2 && (len(fields) < 3 || fields[2] == "") {
			refs[fields[0]] = fields[1]
		}
	}
	return refs, nil
}

func matchRefspec(pattern, ref string) (string, bool) {
	prefix, suffix, wild := strings.Cut(pattern, "*")
	if !wild {
		return "", ref == pattern
	}
	if !strings.HasPrefix(ref, prefix) || !strings.HasSuffix(ref, suffix) || len(ref) < len(prefix)+len(suffix) {
		return "", false
	}
	return ref[len(prefix) : len(ref)-len(suffix)], true
}

func excludedRemoteRef(ref string, specs []string) bool {
	for _, spec := range specs {
		if strings.HasPrefix(spec, "^") {
			if _, ok := matchRefspec(spec[1:], ref); ok {
				return true
			}
		}
	}
	return false
}

func mapFetchRef(ref string, specs []string, reverse bool) string {
	for _, spec := range specs {
		if strings.HasPrefix(spec, "^") {
			continue
		}
		source, destination, ok := strings.Cut(strings.TrimPrefix(spec, "+"), ":")
		if !ok {
			continue
		}
		from, to := source, destination
		if reverse {
			from, to = destination, source
		}
		if wildcard, ok := matchRefspec(from, ref); ok {
			result := strings.Replace(to, "*", wildcard, 1)
			sourceRef := ref
			if reverse {
				sourceRef = result
			}
			if !excludedRemoteRef(sourceRef, specs) {
				return result
			}
		}
	}
	return ""
}

func inspectRemoteUpdates(ctx context.Context, repo string) (RemoteStatus, error) {
	local, err := readRefTips(ctx, repo)
	if err != nil {
		return RemoteStatus{}, err
	}
	remotes := readRemotes(ctx, repo)
	names := remoteNames(remotes)
	branches, tags, removed := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, name := range names {
		raw, err := gitCommand(ctx, repo, "ls-remote", "--refs", "--heads", "--tags", "--", name)
		if err != nil {
			return RemoteStatus{}, fmt.Errorf("%s: %w", name, err)
		}
		specText, _ := gitCommand(ctx, repo, "config", "--get-all", "remote."+name+".fetch")
		specs := strings.Fields(specText)
		advertised := map[string]string{}
		for _, line := range strings.Split(raw, "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				continue
			}
			sha, ref := fields[0], fields[1]
			advertised[ref] = sha
			if strings.HasPrefix(ref, "refs/tags/") {
				if local[ref] != sha {
					tags[ref] = true
				}
			} else if strings.HasPrefix(ref, "refs/heads/") {
				if destination := mapFetchRef(ref, specs, false); destination != "" && local[destination] != sha {
					branches[destination] = true
				}
			}
		}
		for ref := range local {
			if source := mapFetchRef(ref, specs, true); strings.HasPrefix(source, "refs/heads/") && advertised[source] == "" {
				branches[ref] = true
				removed[ref] = true
			}
		}
	}
	return RemoteStatus{CheckedAt: time.Now(), Branches: len(branches), Tags: len(tags), RemovedBranches: len(removed)}, nil
}

func changedRemoteRefs(before, after map[string]string) (int, int) {
	changed := map[string]bool{}
	for ref, sha := range after {
		if before[ref] != sha {
			changed[ref] = true
		}
	}
	for ref := range before {
		if after[ref] == "" {
			changed[ref] = true
		}
	}
	branches, tags := 0, 0
	for ref := range changed {
		if strings.HasPrefix(ref, "refs/remotes/") {
			branches++
		}
		if strings.HasPrefix(ref, "refs/tags/") {
			tags++
		}
	}
	return branches, tags
}

func (a *App) remoteState() RemoteStatus { return a.remoteStates[a.repo()] }

func (a *App) beginRemoteCheck(repo string) uint64 {
	if a.remoteStates == nil {
		a.remoteStates = map[string]RemoteStatus{}
	}
	if a.remoteVersions == nil {
		a.remoteVersions = map[string]uint64{}
	}
	a.remoteVersions[repo]++
	status := a.remoteStates[repo]
	status.Checking = true
	status.Error = ""
	a.remoteStates[repo] = status
	return a.remoteVersions[repo]
}

func (a *App) checkRemoteUpdates() {
	repo := a.repo()
	if repo == "" || len(a.snapshot.Remotes) == 0 || a.busy[repo] != "" || a.remoteState().Checking {
		return
	}
	version := a.beginRemoteCheck(repo)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		status, err := inspectRemoteUpdates(ctx, repo)
		a.update(func() {
			if a.remoteVersions[repo] != version {
				return
			}
			previous := a.remoteStates[repo]
			status.FetchedAt = previous.FetchedAt
			status.FetchedBranches = previous.FetchedBranches
			status.FetchedTags = previous.FetchedTags
			if err != nil {
				status = previous
				status.Checking = false
				status.Error = err.Error()
			}
			a.remoteStates[repo] = status
		})
	}()
}

func (a *App) fetchRemoteUpdates(automatic bool, config RepoConfig) {
	repo := a.repo()
	if repo == "" || a.busy[repo] != "" {
		return
	}
	version := a.beginRemoteCheck(repo)
	delete(a.autoFetchErrors, repo)
	a.busy[repo] = "Fetching remotes…"
	a.showBranches = false
	a.showCommands = false
	a.showStashes = false
	args := fetchArguments(config)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		before, _ := readRefTips(ctx, repo)
		_, fetchErr := gitCommand(ctx, repo, args...)
		after, _ := readRefTips(ctx, repo)
		status, checkErr := inspectRemoteUpdates(ctx, repo)
		if checkErr != nil {
			status.Error = checkErr.Error()
		}
		if fetchErr == nil {
			status.FetchedAt = time.Now()
			status.FetchedBranches, status.FetchedTags = changedRemoteRefs(before, after)
		}
		a.update(func() {
			delete(a.busy, repo)
			if a.remoteVersions[repo] != version {
				return
			}
			previous := a.remoteStates[repo]
			if status.FetchedAt.IsZero() {
				status.FetchedAt = previous.FetchedAt
				status.FetchedBranches = previous.FetchedBranches
				status.FetchedTags = previous.FetchedTags
			}
			if status.CheckedAt.IsZero() {
				status.CheckedAt = previous.CheckedAt
			}
			a.remoteStates[repo] = status
			if fetchErr != nil {
				a.autoFetchErrors[repo] = fetchErr.Error()
			} else {
				delete(a.autoFetchErrors, repo)
			}
			if repo != a.repo() {
				return
			}
			if fetchErr == nil {
				a.loadWithScroll(true, automatic)
			} else if !automatic {
				a.fail(fetchErr)
			}
		})
	}()
}

func (s RemoteStatus) tooltip() string {
	lines := []string{}
	if s.CheckedAt.IsZero() {
		lines = append(lines, "Remote branches and tags have not been verified yet.")
	} else {
		lines = append(lines, "Remote check: "+s.CheckedAt.Format("2006-01-02 15:04:05"))
	}
	lines = append(lines, fmt.Sprintf("Branch updates: %d · Tag updates: %d", s.Branches, s.Tags))
	if s.RemovedBranches > 0 {
		lines = append(lines, fmt.Sprintf("%d remote branch(es) were removed; prune updates the local tracking list.", s.RemovedBranches))
	}
	if !s.FetchedAt.IsZero() {
		lines = append(lines, "Last successful fetch: "+s.FetchedAt.Format("2006-01-02 15:04:05"), fmt.Sprintf("Fetched changes: %d branch refs · %d tags", s.FetchedBranches, s.FetchedTags))
	}
	if s.Error != "" {
		lines = append(lines, s.Error)
	}
	return strings.Join(lines, "\n")
}

func (a *App) remoteStatusView(c *ui.Context) {
	if a.repo() == "" {
		return
	}
	t := c.Theme()
	status := a.remoteState()
	ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Label("Remote freshness").Children(func() {
		label, color := "Remote not checked", t.Warning
		noRemotes := !a.loading && len(a.snapshot.Remotes) == 0
		switch {
		case a.loading || status.Checking:
			label, color = "Checking remote…", t.TextMuted
			ui.Spinner(c).Size(11, 11)
		case noRemotes:
			label, color = "No remotes", t.TextMuted
		case status.Error != "":
			label = "Remote check failed"
		case status.Branches+status.Tags > 0:
			label = fmt.Sprintf("Fetch needed · %d branches · %d tags", status.Branches, status.Tags)
		case status.fresh(time.Now()):
			label, color = "Checked "+status.CheckedAt.Format("15:04")+" · Remote refs current", t.Success
			if status.FetchedBranches+status.FetchedTags > 0 {
				label = fmt.Sprintf("Fetched %s · %d branches · %d tags", status.FetchedAt.Format("15:04"), status.FetchedBranches, status.FetchedTags)
			}
		case !status.CheckedAt.IsZero():
			label = "Recheck needed · " + status.CheckedAt.Format("15:04")
		}
		ui.Text(c, label).MaxWidth(250).SingleLine().FontSize(11).TextColor(color).Tooltip(status.tooltip())
		if noRemotes {
			return
		}
		if ui.Button(c, "Check").Label("Check remote updates").Height(20).PaddingX(5).FontSize(10).Disabled(status.Checking || a.loading || a.busy[a.repo()] != "").Tooltip("Check remote branches and tags without downloading commits").Clicked() {
			a.checkRemoteUpdates()
		}
		if status.Branches+status.Tags > 0 {
			action := "Fetch"
			if status.RemovedBranches == status.Branches && status.Tags == 0 {
				action = "Prune"
			}
			if ui.Button(c, action).Label("Fetch remote updates").Height(20).PaddingX(5).FontSize(10).Disabled(status.Checking || a.busy[a.repo()] != "").Clicked() {
				config := a.config
				if action == "Prune" {
					config.FetchPrune = true
				}
				a.fetchRemoteUpdates(false, config)
			}
		}
	})
}

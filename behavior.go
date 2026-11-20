package main

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func logArguments(config RepoConfig, limit int, ref string) []string {
	author, email := "%an", "%ae"
	committer, committerEmail := "%cn", "%ce"
	if config.UseMailmap {
		author, email = "%aN", "%aE"
		committer, committerEmail = "%cN", "%cE"
	}
	args := []string{"log", "--topo-order", "--no-decorate", "--format=%H%x00%P%x00" + author + "%x00" + email + "%x00%at%x00%s%x00%b%x00%ct%x00" + committer + "%x00" + committerEmail + "%x00", "-n", fmt.Sprint(limit + 1)}
	if config.FirstParent {
		args = append(args, "--first-parent")
	}
	if config.IncludeReflogs {
		args = append(args, "--reflog")
	}
	if ref != "" {
		args = append(args, strings.Split(ref, "\x1e")...)
	} else if config.OnlyTags {
		args = append(args, "--tags", "--not", "--branches", "--remotes")
	} else {
		if config.Branches {
			args = append(args, "--branches")
		}
		if config.Remotes {
			args = append(args, "--remotes")
		}
		if config.Tags {
			args = append(args, "--tags")
		}
		if config.RemoteHeads {
			args = append(args, "--glob=refs/remotes/*/HEAD")
		}
		args = append(args, "HEAD")
	}
	return append(args, "--")
}

func decorateRefs(refs []string, config RepoConfig, branchSets ...[]Branch) []string {
	role := map[string]bool{}
	for _, branches := range branchSets {
		for _, b := range branches {
			role[b.Name] = b.Remote
		}
	}

	result := append([]string(nil), refs...)
	if config.CombineRefs {
		skip := map[string]bool{}
		paired := map[string]string{}
		for _, remote := range refs {
			if strings.HasPrefix(remote, "tag:") {
				continue
			}
			prefix, local, ok := strings.Cut(remote, "/")
			if !ok {
				continue
			}
			for _, r := range refs {
				if r == local {
					if paired[r] == "" || prefix == "origin" {
						paired[r] = prefix
					}
					skip[remote] = true
				}
			}
		}
		result = nil
		for _, r := range refs {
			if skip[r] {
				continue
			}
			label := r
			if remote := paired[r]; remote != "" {
				label += "\x1f" + remote
			}
			result = append(result, label)
		}
	}
	if config.FullRefs {
		for i, r := range result {
			if strings.HasPrefix(r, "tag:") {
				result[i] = "tag:refs/tags/" + strings.TrimPrefix(r, "tag:")
			} else {
				base, _, _ := strings.Cut(r, "\x1f")
				remote := false
				for _, candidate := range refs {
					if candidate == base && role[candidate] {
						remote = true
						break
					}
				}
				if remote {
					result[i] = "refs/remotes/" + r
				} else {
					result[i] = "refs/heads/" + r
				}
			}
		}
	}
	return result
}

func readRemotes(ctx context.Context, repo string) map[string]string {
	result := map[string]string{}
	out, _ := gitCommand(ctx, repo, "config", "--get-regexp", "^remote\\..*\\.url$")
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, " ")
		if !ok {
			key, value, ok = strings.Cut(line, "\t")
		}
		if ok {
			result[strings.TrimSuffix(strings.TrimPrefix(key, "remote."), ".url")] = strings.TrimSpace(value)
		}
	}
	return result
}

func fetchArguments(config RepoConfig) []string {
	args := []string{"fetch", "--all", "--tags"}
	if config.FetchPrune {
		args = append(args, "--prune")
	}
	if config.FetchPruneTags {
		args = append(args, "--prune-tags")
	}
	return args
}

func (a *App) fetch() { a.fetchRemoteUpdates(false, a.config) }

func (a *App) autoFetchOnSwitch() {
	if !a.prefs.AutoFetchOnSwitch || a.repo() == "" || len(a.snapshot.Remotes) == 0 {
		return
	}
	a.fetchRemoteUpdates(true, a.config)
}

func (a *App) sortRepos(active string) {
	if a.config.RepoOrder != "given" {
		sort.SliceStable(a.repos, func(i, j int) bool {
			if a.config.RepoOrder == "path" {
				return a.repos[i] < a.repos[j]
			}
			left, right := strings.ToLower(a.repoLabel(a.repos[i])), strings.ToLower(a.repoLabel(a.repos[j]))
			if left == right {
				return a.repos[i] < a.repos[j]
			}
			return left < right
		})
	}
	for i, p := range a.repos {
		if p == active {
			a.active = i
			return
		}
	}
}

func branchMatches(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		expression := "^" + strings.ReplaceAll(strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*"), "\\?", ".") + "$"
		for _, candidate := range []string{name, "heads/" + name, "remotes/" + name, "refs/heads/" + name, "refs/remotes/" + name} {
			if match, _ := regexp.MatchString(expression, candidate); match {
				return true
			}
		}
	}
	return false
}

func (a *App) dateText(commit Commit) string {
	t := commit.Time
	if a.config.DateCommit && !commit.CommitTime.IsZero() {
		t = commit.CommitTime
	}
	if a.config.RelativeDates {
		return relativeDate(t, time.Now())
	}
	if a.config.DateShort {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:04")
}
func relativeDate(t, now time.Time) string {
	delta := now.Sub(t)
	if delta < 0 {
		return "in the future"
	}
	if delta < time.Minute {
		return "just now"
	}
	units := []struct {
		d    time.Duration
		name string
	}{{365 * 24 * time.Hour, "year"}, {30 * 24 * time.Hour, "month"}, {7 * 24 * time.Hour, "week"}, {24 * time.Hour, "day"}, {time.Hour, "hour"}, {time.Minute, "minute"}}
	for _, u := range units {
		if delta >= u.d {
			n := int(delta / u.d)
			plural := ""
			if n != 1 {
				plural = "s"
			}
			return fmt.Sprintf("%d %s%s ago", n, u.name, plural)
		}
	}
	return "just now"
}

func (a *App) colors() []ui.Color {
	if len(a.config.LaneColors) > 0 {
		var colors []ui.Color
		for _, s := range a.config.LaneColors {
			if match, _ := regexp.MatchString("^#[0-9a-fA-F]{6}$", s); match {
				colors = append(colors, ui.Hex(s))
			}
		}
		if len(colors) > 0 {
			return colors
		}
	}
	if a.config.ColorPreset == 1 {
		return []ui.Color{ui.Hex("#1f77b4"), ui.Hex("#ff7f0e"), ui.Hex("#2ca02c"), ui.Hex("#d62728"), ui.Hex("#9467bd"), ui.Hex("#8c564b"), ui.Hex("#e377c2"), ui.Hex("#7f7f7f")}
	}
	if a.config.ColorPreset == 2 {
		return []ui.Color{ui.Hex("#808080")}
	}
	return laneColors
}

func (a *App) action(m *ui.Menu, key, label string, disabled ...bool) bool {
	for _, hidden := range a.config.HiddenActions {
		if hidden == key {
			return false
		}
	}
	blocked := a.busy[a.repo()] != ""
	for _, v := range disabled {
		blocked = blocked || v
	}
	return m.Item(label).Disabled(blocked).Chosen()
}

type remoteParts struct{ Host, Owner, Repo string }

func parseRemote(raw string) (remoteParts, bool) {
	if !strings.Contains(raw, "://") {
		if host, path, ok := strings.Cut(raw, ":"); ok {
			if _, name, ok := strings.Cut(host, "@"); ok {
				host = name
			}
			raw = "ssh://" + host + "/" + path
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return remoteParts{}, false
	}
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return remoteParts{}, false
	}
	return remoteParts{u.Hostname(), strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1]}, true
}
func providerLink(raw, template, base, head, issue string, pull bool) string {
	r, ok := parseRemote(raw)
	if !ok {
		return ""
	}
	if template == "" {
		switch {
		case strings.Contains(r.Host, "github"):
			if pull {
				template = "https://{host}/{owner}/{repo}/compare/{base}...{head}?expand=1"
			} else {
				template = "https://{host}/{owner}/{repo}/issues/{issue}"
			}
		case strings.Contains(r.Host, "gitlab"):
			if pull {
				template = "https://{host}/{owner}/{repo}/-/merge_requests/new?merge_request[source_branch]={head}&merge_request[target_branch]={base}"
			} else {
				template = "https://{host}/{owner}/{repo}/-/issues/{issue}"
			}
		case strings.Contains(r.Host, "bitbucket"):
			if pull {
				template = "https://{host}/{owner}/{repo}/pull-requests/new?source={head}&dest={base}"
			} else {
				template = "https://{host}/{owner}/{repo}/issues/{issue}"
			}
		default:
			return ""
		}
	}
	owner := strings.Split(r.Owner, "/")
	for i := range owner {
		owner[i] = url.PathEscape(owner[i])
	}
	values := map[string]string{"host": r.Host, "owner": strings.Join(owner, "/"), "repo": url.PathEscape(r.Repo), "base": url.QueryEscape(base), "head": url.QueryEscape(head), "issue": url.PathEscape(issue)}
	for key, value := range values {
		template = strings.ReplaceAll(template, "{"+key+"}", value)
	}
	u, err := url.Parse(template)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	return template
}
func (a *App) remoteURL() string {
	remote, _, _ := strings.Cut(a.snapshot.Tracking.Upstream, "/")
	if raw := a.snapshot.Remotes[remote]; raw != "" {
		return raw
	}
	if raw := a.snapshot.Remotes["origin"]; raw != "" {
		return raw
	}
	keys := make([]string, 0, len(a.snapshot.Remotes))
	for k := range a.snapshot.Remotes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return a.snapshot.Remotes[keys[0]]
	}
	return ""
}
func (a *App) pullRequestLink() string {
	base := "main"
	for _, b := range a.snapshot.Branches {
		if b.Symbolic != "" {
			base = strings.TrimPrefix(b.Symbolic, "refs/remotes/")
			_, base, _ = strings.Cut(base, "/")
			break
		}
	}
	return providerLink(a.remoteURL(), a.config.PrProvider, base, a.snapshot.Tracking.Branch, "", true)
}
func (a *App) openURL(link string) {
	if link != "" {
		if a.openExternal != nil {
			a.fail(a.openExternal(link))
		} else {
			a.fail(mygo.Shell.OpenExternal(link))
		}
	}
}

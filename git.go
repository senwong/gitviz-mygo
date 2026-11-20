package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	SHA, Subject, Body, Author, Email string
	Committer, CommitterEmail         string
	Parents, Refs                     []string
	Time                              time.Time
	CommitTime                        time.Time
	Lane                              int
	Through, Incoming, Outgoing       []int
	Top, Bottom                       bool
}

type Branch struct {
	Name, Full, SHA, Upstream, Symbolic string
	Remote, Current                     bool
}

type Tracking struct {
	Branch, Upstream string
	Ahead, Behind    int
	Gone, Detached   bool
}

type FileChange struct {
	Path, OldPath, Status string
	Untracked             bool
	Additions, Deletions  int
	StatsKnown, Binary    bool
}

type Stash struct{ Ref, SHA, Message string }

type Tag struct {
	Name, SHA string
	Annotated bool
}

type Snapshot struct {
	Tags      map[string]Tag
	Head      string
	Ancestors map[string]bool
	Remotes   map[string]string
	Commits   []Commit
	Branches  []Branch
	Files     []FileChange
	Stashes   []Stash
	Tracking  Tracking
	Lanes     int
	More      bool
}

func gitCommand(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo, "--no-pager", "-c", "color.ui=false", "-c", "core.quotepath=false"}, args...)...)
	cmd.Env = append(commandEnv(), "GIT_TERMINAL_PROMPT=0", "GIT_MERGE_AUTOEDIT=no")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return string(output), nil
}

func repositoryRoot(ctx context.Context, path string) (string, error) {
	root, err := gitCommand(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(root)), nil
}

func readTracking(ctx context.Context, repo string, branches []Branch) Tracking {
	name, _ := gitCommand(ctx, repo, "symbolic-ref", "--quiet", "--short", "HEAD")
	t := Tracking{Branch: strings.TrimSpace(name)}
	if t.Branch == "" {
		sha, _ := gitCommand(ctx, repo, "rev-parse", "--short", "HEAD")
		t.Branch, t.Detached = "Detached "+strings.TrimSpace(sha), true
		return t
	}
	for _, b := range branches {
		if b.Current {
			t.Upstream = b.Upstream
			break
		}
	}
	if t.Upstream == "" {
		return t
	}
	counts, err := gitCommand(ctx, repo, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		t.Gone = true
		return t
	}
	fields := strings.Fields(counts)
	if len(fields) == 2 {
		t.Ahead, _ = strconv.Atoi(fields[0])
		t.Behind, _ = strconv.Atoi(fields[1])
	}
	return t
}

func readBranches(ctx context.Context, repo string) ([]Branch, map[string][]string, error) {
	out, err := gitCommand(ctx, repo, "for-each-ref", "--sort=refname", "--format=%(refname)%00%(objectname)%00%(*objectname)%00%(symref)%00%(upstream:short)%00%(HEAD)")
	if err != nil {
		return nil, nil, err
	}
	var branches []Branch
	refs := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Split(line, "\x00")
		if len(f) != 6 {
			continue
		}
		full, sha := f[0], f[1]
		if f[2] != "" {
			sha = f[2]
		}
		if strings.HasPrefix(full, "refs/tags/") {
			refs[sha] = append(refs[sha], "tag:"+strings.TrimPrefix(full, "refs/tags/"))
			continue
		}
		remote := strings.HasPrefix(full, "refs/remotes/")
		if !remote && !strings.HasPrefix(full, "refs/heads/") {
			continue
		}
		name := strings.TrimPrefix(strings.TrimPrefix(full, "refs/heads/"), "refs/remotes/")
		b := Branch{Name: name, Full: full, SHA: sha, Remote: remote, Symbolic: f[3], Upstream: f[4], Current: f[5] == "*"}
		branches = append(branches, b)
		if b.Symbolic == "" {
			refs[sha] = append(refs[sha], name)
		}
	}
	return branches, refs, nil
}

func readLog(ctx context.Context, repo string, limit int, remote bool, ref string) ([]Commit, bool, error) {
	config := defaultConfig()
	config.Remotes = remote
	return readLogConfig(ctx, repo, limit, config, ref)
}

func readLogConfig(ctx context.Context, repo string, limit int, config RepoConfig, ref string) ([]Commit, bool, error) {
	args := logArguments(config, limit, ref)
	out, err := gitCommand(ctx, repo, args...)
	if err != nil {
		if _, headErr := gitCommand(ctx, repo, "rev-parse", "--verify", "HEAD"); headErr != nil {
			if _, repoErr := gitCommand(ctx, repo, "rev-parse", "--git-dir"); repoErr == nil {
				return nil, false, nil
			}
		}
		return nil, false, err
	}
	fields := strings.Split(out, "\x00")
	commits := make([]Commit, 0, min(limit+1, len(fields)/10))
	for len(fields) >= 10 {
		seconds, _ := strconv.ParseInt(fields[4], 10, 64)
		committed, _ := strconv.ParseInt(fields[7], 10, 64)
		commits = append(commits, Commit{SHA: strings.TrimSpace(fields[0]), Parents: strings.Fields(fields[1]), Author: fields[2], Email: fields[3], Time: time.Unix(seconds, 0), CommitTime: time.Unix(committed, 0), Subject: fields[5], Body: strings.TrimSpace(fields[6]), Committer: fields[8], CommitterEmail: fields[9]})
		fields = fields[10:]
	}
	more := len(commits) > limit
	if more {
		commits = commits[:limit]
	}
	return commits, more, nil
}

func readStatus(ctx context.Context, repo string) ([]FileChange, error) {
	out, err := gitCommand(ctx, repo, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	fields := strings.Split(out, "\x00")
	var files []FileChange
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		file := FileChange{Status: f[:2], Path: f[3:], Untracked: f[:2] == "??"}
		if strings.ContainsAny(f[:2], "RC") && i+1 < len(fields) {
			i++
			file.OldPath = fields[i]
		}
		files = append(files, file)
	}
	return files, nil
}

func readStashes(ctx context.Context, repo string) ([]Stash, error) {
	out, err := gitCommand(ctx, repo, "stash", "list", "--format=%gd%x00%H%x00%gs")
	if err != nil {
		return nil, err
	}
	var result []Stash
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.SplitN(line, "\x00", 3)
		if len(f) == 3 {
			result = append(result, Stash{Ref: f[0], SHA: f[1], Message: f[2]})
		}
	}
	return result, nil
}

func loadSnapshot(ctx context.Context, repo string, limit int, remote bool, ref string) (Snapshot, error) {
	config := defaultConfig()
	config.Remotes = remote
	return loadSnapshotConfig(ctx, repo, limit, config, ref)
}

func loadSnapshotConfig(ctx context.Context, repo string, limit int, config RepoConfig, ref string) (Snapshot, error) {
	var s Snapshot
	branches, refs, err := readBranches(ctx, repo)
	if err != nil {
		return s, err
	}
	s.Branches = branches
	s.Tags, err = readTags(ctx, repo)
	if err != nil {
		return s, err
	}
	s.Commits, s.More, err = readLogConfig(ctx, repo, limit, config, ref)
	if err != nil {
		return s, err
	}
	for i := range s.Commits {
		s.Commits[i].Refs = decorateRefs(refs[s.Commits[i].SHA], config, branches)
		if config.RemoteHeads {
			for _, b := range branches {
				if b.Symbolic != "" && b.SHA == s.Commits[i].SHA {
					label := b.Name
					if config.FullRefs {
						label = b.Full
					}
					s.Commits[i].Refs = append(s.Commits[i].Refs, label)
				}
			}
		}
	}
	s.Lanes = assignLanes(s.Commits)
	s.Tracking = readTracking(ctx, repo, branches)
	head, _ := gitCommand(ctx, repo, "rev-parse", "--verify", "HEAD")
	s.Head = strings.TrimSpace(head)
	s.Ancestors = map[string]bool{}
	if config.DimNonHeadCommits {
		out, _ := gitCommand(ctx, repo, "rev-list", "--max-count=50000", "HEAD")
		for _, sha := range strings.Fields(out) {
			s.Ancestors[sha] = true
		}
	}
	s.Remotes = readRemotes(ctx, repo)
	s.Files, err = readStatus(ctx, repo)
	if err != nil {
		return s, err
	}
	if !config.IncludeUntracked {
		filtered := s.Files[:0]
		for _, f := range s.Files {
			if !f.Untracked {
				filtered = append(filtered, f)
			}
		}
		s.Files = filtered
	}
	if !config.ShowUncommitted {
		s.Files = nil
	}
	if config.ShowStashes {
		s.Stashes, err = readStashes(ctx, repo)
	}
	return s, err
}

func changedFiles(ctx context.Context, repo, revision string) ([]FileChange, error) {
	out, err := gitCommand(ctx, repo, "diff-tree", "--root", "--no-commit-id", "--name-status", "-r", "-M", "--diff-merges=first-parent", "-z", revision, "--")
	if err != nil {
		return nil, err
	}
	f := strings.Split(out, "\x00")
	var files []FileChange
	for i := 0; i+1 < len(f); {
		status, path := strings.TrimSpace(f[i]), f[i+1]
		i += 2
		if status == "" {
			continue
		}
		file := FileChange{Status: status, Path: path}
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i >= len(f) {
				break
			}
			file.OldPath = path
			file.Path = f[i]
			i++
		}
		files = append(files, file)
	}
	return withFileStatistics(ctx, repo, revision, "", files)
}

func fileDiff(ctx context.Context, repo, revision, compare string, file FileChange) (string, error) {
	if !filepath.IsLocal(file.Path) {
		return "", errors.New("invalid repository-relative file path")
	}
	if revision == "worktree" && file.Untracked {
		cmd := exec.CommandContext(ctx, "git", "-C", repo, "diff", "--no-color", "--no-ext-diff", "--no-index", "--", "/dev/null", file.Path)
		out, err := cmd.Output()
		var exit *exec.ExitError
		if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == 1) {
			return "", err
		}
		return string(out), nil
	}
	args := []string{"show", "--format=", "--first-parent", "--no-ext-diff", "--no-color", revision}
	if revision == "worktree" {
		base := "HEAD"
		if _, err := gitCommand(ctx, repo, "rev-parse", "--verify", base); err != nil {
			cmd := exec.CommandContext(ctx, "git", "-C", repo, "hash-object", "-t", "tree", "--stdin")
			cmd.Stdin = strings.NewReader("")
			out, e := cmd.Output()
			if e != nil {
				return "", e
			}
			base = strings.TrimSpace(string(out))
		}
		if compare != "" {
			base = compare
		}
		args = []string{"diff", "--no-ext-diff", "--no-color", base}
	} else if compare != "" {
		args = []string{"diff", "--no-ext-diff", "--no-color", compare, revision}
	} else if strings.HasPrefix(revision, "stash@{") {
		args = []string{"diff", "--no-ext-diff", "--no-color", revision + "^1", revision}
	}
	args = append(args, "--", file.Path)
	if file.OldPath != "" {
		args = append(args, file.OldPath)
	}
	return gitCommand(ctx, repo, args...)
}

func compareFiles(ctx context.Context, repo, from, to string) ([]FileChange, error) {
	args := []string{"diff", "--name-only", "-z", from}
	if to != "worktree" {
		args = append(args, to)
	}
	args = append(args, "--")
	out, err := gitCommand(ctx, repo, args...)
	if err != nil {
		return nil, err
	}
	var result []FileChange
	for _, path := range strings.Split(out, "\x00") {
		if path != "" {
			result = append(result, FileChange{Path: path, Status: "M"})
		}
	}
	return withFileStatistics(ctx, repo, to, from, result)
}

func checkoutBranch(ctx context.Context, repo string, b Branch) error {
	if b.Symbolic != "" {
		return errors.New("select the branch this symbolic reference points to")
	}
	if !b.Remote {
		_, err := gitCommand(ctx, repo, "switch", "--", b.Name)
		return err
	}
	_, name, ok := strings.Cut(b.Name, "/")
	if !ok {
		return errors.New("invalid remote branch")
	}
	if _, err := gitCommand(ctx, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+name); err == nil {
		_, err = gitCommand(ctx, repo, "switch", "--", name)
		return err
	}
	_, err := gitCommand(ctx, repo, "switch", "--track", "-c", name, b.Name)
	return err
}

func pullBranch(ctx context.Context, repo, branch string, rebase bool) error {
	name, err := gitCommand(ctx, repo, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) != branch {
		return errors.New("checked-out branch changed; select it again before pulling")
	}
	args := []string{"-c", "core.editor=true", "pull", "--no-edit"}
	if rebase {
		args = append(args, "--rebase")
	} else {
		args = append(args, "--no-rebase", "--ff")
	}
	_, err = gitCommand(ctx, repo, args...)
	return err
}

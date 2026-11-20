package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Preferences struct {
	LastRepository         string                `json:"last_repository,omitempty"`
	InlineDetails          bool                  `json:"inline_details"`
	AutoFetchOnSwitch      bool                  `json:"auto_fetch_on_switch"`
	AlwaysAcceptCheckout   bool                  `json:"always_accept_checkout,omitempty"`
	PushTagSkipRemoteCheck bool                  `json:"push_tag_skip_remote_check,omitempty"`
	Excluded               []string              `json:"excluded,omitempty"`
	Configs                map[string]RepoConfig `json:"configs,omitempty"`
	Theme                  string                `json:"theme,omitempty"`
	DiscoveryDepth         int                   `json:"discovery_depth"`
	RepositoryShortcut     string                `json:"repository_shortcut"`
	Roots                  []string              `json:"roots"`
	Dark                   bool                  `json:"dark"`
	Remotes                bool                  `json:"remotes"`
	Detail                 bool                  `json:"detail"`
}

func commandEnv() []string { return os.Environ() }

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func prefsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gitviz-mygo", "settings.json")
}

func readPreferences() Preferences {
	p := Preferences{InlineDetails: true, AutoFetchOnSwitch: true, Dark: true, Remotes: true, Detail: true, DiscoveryDepth: 2, RepositoryShortcut: "cmd-p"}
	home, _ := os.UserHomeDir()
	if data, err := os.ReadFile(filepath.Join(home, ".config/gitviz/theme")); err == nil {
		p.Theme = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(home, ".config/gitviz/depth")); err == nil {
		if depth, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			p.DiscoveryDepth = max(0, min(32, depth))
		}
	}
	if data, err := os.ReadFile(filepath.Join(home, ".config/gitviz/repository-shortcut")); err == nil {
		p.RepositoryShortcut = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(prefsPath()); err == nil {
		_ = json.Unmarshal(data, &p)
		return p
	}
	if data, err := os.ReadFile(filepath.Join(home, ".config/gitviz/roots")); err == nil {
		p.Roots = parseWorkspace("roots.gitviz-workspace", string(data))
	}
	return p
}

func savePreferences(p Preferences) error {
	return savePreferencesAt(prefsPath(), p)
}
func savePreferencesAt(path string, p Preferences) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "settings-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func parseWorkspace(path, text string) []string {
	var roots []string
	if strings.HasSuffix(path, ".code-workspace") || strings.HasSuffix(path, ".workspace") {
		var value struct {
			Folders []struct {
				Path string `json:"path"`
			} `json:"folders"`
		}
		if json.Unmarshal([]byte(text), &value) != nil {
			return nil
		}
		for _, f := range value.Folders {
			p := expandHome(f.Path)
			if !filepath.IsAbs(p) {
				p = filepath.Join(filepath.Dir(path), p)
			}
			roots = append(roots, p)
		}
		return roots
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			p := expandHome(line)
			if !filepath.IsAbs(p) {
				p = filepath.Join(filepath.Dir(path), p)
			}
			roots = append(roots, p)
		}
	}
	return roots
}

func discover(ctx context.Context, roots []string) ([]string, error) {
	return discoverDepth(ctx, roots, 2)
}

func discoverDepth(ctx context.Context, roots []string, maxDepth int) ([]string, error) {
	seen := map[string]bool{}
	var repos []string
	var visit func(string, int)
	visit = func(path string, depth int) {
		if ctx.Err() != nil {
			return
		}
		path = expandHome(path)
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		if !info.IsDir() {
			canonical, err := filepath.EvalSymlinks(path)
			if err != nil {
				return
			}
			canonical, _ = filepath.Abs(canonical)
			if seen[canonical] {
				return
			}
			seen[canonical] = true
			if strings.HasSuffix(path, ".gitviz-workspace") || strings.HasSuffix(path, ".code-workspace") || strings.HasSuffix(path, ".workspace") {
				data, err := os.ReadFile(path)
				if err == nil {
					for _, p := range parseWorkspace(path, string(data)) {
						visit(p, 0)
					}
				}
			}
			return
		}
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return
		}
		canonical, _ = filepath.Abs(canonical)
		if seen[canonical] {
			return
		}
		seen[canonical] = true
		if _, err := os.Stat(filepath.Join(canonical, ".git")); err == nil {
			if root, err := repositoryRoot(ctx, canonical); err == nil {
				repos = append(repos, root)
			}
			return
		}
		if depth == 0 {
			if root, err := repositoryRoot(ctx, canonical); err == nil {
				if !seen[root] {
					seen[root] = true
				}
				repos = append(repos, root)
				return
			}
		}
		if depth >= maxDepth {
			return
		}
		entries, _ := os.ReadDir(canonical)
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			switch entry.Name() {
			case "node_modules", "target", "vendor", "Library", "Downloads", "Desktop", "Documents":
				continue
			}
			visit(filepath.Join(canonical, entry.Name()), depth+1)
		}
	}
	for _, root := range roots {
		visit(root, 0)
	}
	unique := repos[:0]
	found := map[string]bool{}
	for _, repo := range repos {
		if !found[repo] {
			found[repo] = true
			unique = append(unique, repo)
		}
	}
	return unique, ctx.Err()
}

func compactStrings(values []string) []string {
	result := values[:0]
	for _, v := range values {
		if len(result) == 0 || result[len(result)-1] != v {
			result = append(result, v)
		}
	}
	return result
}

func workspaceNames(roots []string) map[string]string {
	names := map[string]string{}
	seen := map[string]bool{}
	var read func(string)
	read = func(path string) {
		path = expandHome(path)
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return
		}
		canonical, _ = filepath.Abs(canonical)
		if seen[canonical] {
			return
		}
		seen[canonical] = true
		info, err := os.Stat(canonical)
		if err != nil || info.IsDir() {
			return
		}
		data, err := os.ReadFile(canonical)
		if err != nil {
			return
		}
		var document struct {
			Folders []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"folders"`
		}
		if json.Unmarshal(data, &document) == nil {
			for _, folder := range document.Folders {
				p := expandHome(folder.Path)
				if !filepath.IsAbs(p) {
					p = filepath.Join(filepath.Dir(canonical), p)
				}
				if resolved, err := filepath.EvalSymlinks(p); err == nil && strings.TrimSpace(folder.Name) != "" {
					names[resolved] = folder.Name
				}
			}
		}
		for _, child := range parseWorkspace(canonical, string(data)) {
			read(child)
		}
	}
	for _, root := range roots {
		read(root)
	}
	return names
}

func (a *App) repoLabel(repo string) string {
	if name := a.repoNames[repo]; name != "" {
		return name
	}
	return repoName(repo)
}

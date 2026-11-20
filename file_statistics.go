package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type fileStat struct {
	Additions, Deletions int
	Binary               bool
}

func parseFileStatistics(output string) (map[string]fileStat, error) {
	fields := strings.Split(output, "\x00")
	result := map[string]fileStat{}
	for i := 0; i < len(fields); i++ {
		if fields[i] == "" {
			continue
		}
		columns := strings.SplitN(fields[i], "\t", 3)
		if len(columns) != 3 {
			return nil, fmt.Errorf("invalid Git file statistics")
		}
		path := columns[2]
		if path == "" {
			if i+2 >= len(fields) {
				return nil, fmt.Errorf("incomplete Git rename statistics")
			}
			path = fields[i+2]
			i += 2
		}
		stat := fileStat{Binary: columns[0] == "-" || columns[1] == "-"}
		if !stat.Binary {
			var err error
			if stat.Additions, err = strconv.Atoi(columns[0]); err != nil {
				return nil, err
			}
			if stat.Deletions, err = strconv.Atoi(columns[1]); err != nil {
				return nil, err
			}
		}
		result[path] = stat
	}
	return result, nil
}

func withFileStatistics(ctx context.Context, repo, revision, compare string, files []FileChange) ([]FileChange, error) {
	args := []string{"diff-tree", "--root", "--no-commit-id", "--numstat", "-r", "-M", "--diff-merges=first-parent", "-z", revision, "--"}
	if compare != "" {
		args = []string{"diff", "--numstat", "--no-ext-diff", "-z", compare}
		if revision != "worktree" {
			args = append(args, revision)
		}
		args = append(args, "--")
	}
	output, err := gitCommand(ctx, repo, args...)
	if err != nil {
		return nil, err
	}
	stats, err := parseFileStatistics(output)
	if err != nil {
		return nil, err
	}
	for i := range files {
		if stat, ok := stats[files[i].Path]; ok {
			files[i].Additions = stat.Additions
			files[i].Deletions = stat.Deletions
			files[i].Binary = stat.Binary
			files[i].StatsKnown = true
		}
	}
	return files, nil
}

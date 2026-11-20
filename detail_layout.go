package main

import "strings"

func compactMessage(body string) string {
	lines := []string{}
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
		if len(lines) == 2 {
			break
		}
	}
	return strings.Join(lines, "\n")
}
func compactDiffLines(lines []string) []string {
	result := make([]string, 0, len(lines))
	inHunk := false
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			inHunk = false
			continue
		}
		if strings.HasPrefix(line, "@@") {
			inHunk = true
		}
		if !inHunk && (strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ")) {
			continue
		}
		result = append(result, line)
	}
	return result
}

func (a *App) displayedDiffLines() []string {
	if a.diffMetadata {
		return a.diffLines
	}
	if len(a.diffLines) == 0 {
		return nil
	}
	if len(a.compactDiffSource) != len(a.diffLines) || &a.compactDiffSource[0] != &a.diffLines[0] {
		a.compactDiffCache = compactDiffLines(a.diffLines)
		a.compactDiffSource = a.diffLines
	}
	return a.compactDiffCache
}

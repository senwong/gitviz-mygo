package main

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
)

type diffSearchMatch struct{ Line, Start, End int }

func (a *App) isDiffView() bool {
	return a.detailVisible && a.revision != "" && (a.largeDiffVisible || ((!a.prefs.InlineDetails || strings.HasPrefix(a.revision, "stash@{")) && a.selectedFile >= 0))
}

func (a *App) openSearch() {
	a.searchDiff = a.isDiffView()
	a.showSearch = true
	a.searchFocused = false
	if a.searchDiff {
		if a.imageDiff != nil {
			a.imageDiffText = true
		}
		a.rebuildDiffSearch()
	}
}

func (a *App) restoreGraphSearch() {
	a.searchDiff = false
	a.showSearch = a.query != ""
	a.searchFocused = false
}

func (a *App) closeViewOrWindow() {
	if a.modalOpen() {
		return
	}
	if a.isDiffView() {
		a.returnToGraph()
		if a.win != nil {
			a.win.Invalidate()
		}
		return
	}
	if a.win != nil {
		a.win.Close()
	}
}

func (a *App) moveFindMatch(delta int) {
	if !a.searchDiff {
		if !a.isDiffView() || a.showSearch {
			a.moveSearchMatch(delta)
		}
		return
	}
	if !a.showSearch || !a.searchDiff {
		return
	}
	a.rebuildDiffSearch()
	if len(a.diffMatches) == 0 {
		return
	}
	a.diffMatchCursor = (a.diffMatchCursor + delta + len(a.diffMatches)) % len(a.diffMatches)
	a.scrollToDiffLine(a.diffMatches[a.diffMatchCursor].Line)
}

func (a *App) rebuildDiffSearch() {
	lines := a.displayedDiffLines()
	file := a.repo() + "\x00" + a.revision
	if a.selectedFile >= 0 && a.selectedFile < len(a.files) {
		file += "\x00" + a.files[a.selectedFile].Path
	}
	sameSource := len(lines) == len(a.diffFindSource) && (len(lines) == 0 || &lines[0] == &a.diffFindSource[0])
	if sameSource && file == a.diffFindFile && a.diffQuery == a.diffFindQuery {
		return
	}
	a.diffFindSource = lines
	a.diffFindQuery = a.diffQuery
	a.diffFindFile = file
	a.diffMatches = nil
	a.diffMatchLines = map[int][]int{}
	a.diffMatchCursor = -1
	if a.diffQuery == "" || a.diffLoading {
		return
	}
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(a.diffQuery))
	for i, line := range lines {
		for _, pair := range pattern.FindAllStringIndex(line, -1) {
			a.diffMatchLines[i] = append(a.diffMatchLines[i], len(a.diffMatches))
			a.diffMatches = append(a.diffMatches, diffSearchMatch{i, pair[0], pair[1]})
		}
	}
	if len(a.diffMatches) > 0 {
		a.diffMatchCursor = 0
		a.scrollToDiffLine(a.diffMatches[0].Line)
	}
}

func (a *App) diffSearchSpans(line int, text string) []ui.Span {
	if !a.showSearch || !a.searchDiff || len(a.diffMatchLines[line]) == 0 {
		return nil
	}
	var spans []ui.Span
	at := 0
	for _, index := range a.diffMatchLines[line] {
		match := a.diffMatches[index]
		if match.Start > at {
			spans = append(spans, ui.Span{Text: text[at:match.Start]})
		}
		background, foreground := ui.Hex("#ffe08a"), ui.Hex("#242424")
		if a.uiTheme.Dark {
			background, foreground = ui.Hex("#805800"), ui.Hex("#ffffff")
		}
		if index == a.diffMatchCursor {
			background, foreground = ui.Hex("#ffc857"), ui.Hex("#242424")
		}
		spans = append(spans, ui.Span{Text: text[match.Start:match.End], Background: background, Color: foreground})
		at = match.End
	}
	if at < len(text) {
		spans = append(spans, ui.Span{Text: text[at:]})
	}
	return spans
}

func (a *App) scrollToDiffLine(line int) { a.diffScrollTarget = &line }

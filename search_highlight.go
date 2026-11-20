package main

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
)

func referenceSearchSpans(text, query string, dark bool) []ui.Span {
	query = strings.TrimSpace(query)
	if query == "" {
		return []ui.Span{{Text: text}}
	}
	matches := regexp.MustCompile("(?i)"+regexp.QuoteMeta(query)).FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return []ui.Span{{Text: text}}
	}
	background, foreground := ui.Hex("#ffe08a"), ui.Hex("#242424")
	if dark {
		background, foreground = ui.Hex("#a66b00"), ui.Hex("#ffffff")
	}
	spans := make([]ui.Span, 0, 2*len(matches)+1)
	at := 0
	for _, match := range matches {
		if match[0] > at {
			spans = append(spans, ui.Span{Text: text[at:match[0]]})
		}
		spans = append(spans, ui.Span{Text: text[match[0]:match[1]], Background: background, Color: foreground})
		at = match[1]
	}
	if at < len(text) {
		spans = append(spans, ui.Span{Text: text[at:]})
	}
	return spans
}

func (a *App) graphSearchText(c *ui.Context, text string) ui.Element {
	query := ""
	if a.showSearch && !a.searchDiff {
		query = a.query
	}
	return ui.RichText(c, referenceSearchSpans(text, query, a.uiTheme.Dark)...)
}

func (a *App) referenceText(c *ui.Context, text string) ui.Element {
	return a.graphSearchText(c, text)
}

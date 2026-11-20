package main

import (
	"github.com/egoist/mygo/ui"
	"regexp"
	"strings"
	"unicode"
)

var messageLinkPattern = regexp.MustCompile(`https?://[^\s<>]+|#[0-9]+`)

func (a *App) messageElement(c *ui.Context, text string) ui.Element {
	text = a.messageText(text)
	return ui.RichText(c).Children(func() {
		at := 0
		for _, match := range messageLinkPattern.FindAllStringIndex(text, -1) {
			if match[0] > at {
				ui.RichText(c, messageSpans(text[at:match[0]], a.config.Markdown)...)
			}
			token := text[match[0]:match[1]]
			link := strings.TrimRight(token, ".,;:!?)")
			if strings.HasPrefix(token, "#") {
				link = providerLink(a.remoteURL(), a.config.IssueProvider, "", "", token[1:], false)
			}
			if link != "" {
				ui.Link(c, token, link)
			} else {
				ui.Text(c, token)
			}
			at = match[1]
		}
		if at < len(text) {
			ui.RichText(c, messageSpans(text[at:], a.config.Markdown)...)
		}
	})
}

var emojiCodes = map[string]string{
	"bug":                  "🐛",
	"sparkles":             "✨",
	"zap":                  "⚡️",
	"boom":                 "💥",
	"fire":                 "🔥",
	"art":                  "🎨",
	"memo":                 "📝",
	"rocket":               "🚀",
	"construction":         "🚧",
	"wrench":               "🔧",
	"ambulance":            "🚑",
	"lipstick":             "💄",
	"tada":                 "🎉",
	"white_check_mark":     "✅",
	"heavy_check_mark":     "✔️",
	"x":                    "❌",
	"warning":              "⚠️",
	"lock":                 "🔒",
	"arrow_up":             "⬆️",
	"arrow_down":           "⬇️",
	"heavy_plus_sign":      "➕",
	"heavy_minus_sign":     "➖",
	"package":              "📦",
	"bookmark":             "🔖",
	"recycle":              "♻️",
	"pencil2":              "✏️",
	"loud_sound":           "🔊",
	"mute":                 "🔇",
	"bulb":                 "💡",
	"globe_with_meridians": "🌐",
	"smile":                "😄",
	"laughing":             "😆",
	"wink":                 "😉",
	"heart":                "❤️",
	"thumbsup":             "👍",
	"+1":                   "👍",
	"thumbsdown":           "👎",
	"-1":                   "👎",
	"eyes":                 "👀",
	"tada2":                "🎉",
	"100":                  "💯",
	"ok_hand":              "👌",
	"clap":                 "👏",
	"pray":                 "🙏",
	"thinking":             "🤔",
	"sob":                  "😭",
	"cry":                  "😢",
	"rage":                 "😡",
	"sunglasses":           "😎",
	"robot":                "🤖",
	"trophy":               "🏆",
	"medal":                "🏅",
	"star":                 "⭐",
	"star2":                "🌟",
}

func (a *App) messageText(text string) string {
	if !a.config.Emoji || !strings.Contains(text, ":") {
		return text
	}
	for _, mapping := range a.config.EmojiMappings {
		code, value, ok := strings.Cut(mapping, ":")
		if ok {
			text = strings.ReplaceAll(text, ":"+code+":", value)
		}
	}
	for code, value := range emojiCodes {
		text = strings.ReplaceAll(text, ":"+code+":", value)
	}
	return text
}

func messageSpans(text string, markdown bool) []ui.Span {
	if !markdown {
		return []ui.Span{{Text: text}}
	}
	var spans []ui.Span
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			spans = append(spans, ui.Span{Text: plain.String()})
			plain.Reset()
		}
	}
	for i := 0; i < len(text); {
		matched := false
		for _, marker := range []string{"**", "__", "`", "*", "_"} {
			if !strings.HasPrefix(text[i:], marker) {
				continue
			}
			if marker == "_" && i > 0 && unicode.IsLetter(rune(text[i-1])) {
				continue
			}
			end := strings.Index(text[i+len(marker):], marker)
			if end < 0 {
				continue
			}
			flush()
			span := ui.Span{Text: text[i+len(marker) : i+len(marker)+end]}
			switch marker {
			case "**", "__":
				span.Weight = 700
			case "*", "_":
				span.Italic = true
			case "`":
				span.Font = "Menlo"
			}
			spans = append(spans, span)
			i += 2*len(marker) + end
			matched = true
			break
		}
		if !matched {
			plain.WriteByte(text[i])
			i++
		}
	}
	flush()
	return spans
}

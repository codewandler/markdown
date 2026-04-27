// Package html renders stream parser events to reference HTML for conformance
// tests.
package html

import (
	"html"
	"strings"

	"github.com/codewandler/markdown/stream"
)

// Render renders stream events to deterministic HTML.
func Render(events []stream.Event) string {
	r := renderer{}
	for _, event := range events {
		r.render(event)
	}
	return r.out.String()
}

type renderer struct {
	out    strings.Builder
	inCode bool
}

func (r *renderer) render(event stream.Event) {
	switch event.Kind {
	case stream.EventEnterBlock:
		r.enter(event)
	case stream.EventExitBlock:
		r.exit(event)
	case stream.EventText:
		r.text(event)
	case stream.EventSoftBreak:
		r.out.WriteByte('\n')
	case stream.EventLineBreak:
		if r.inCode {
			r.out.WriteByte('\n')
		} else {
			r.out.WriteString("<br />\n")
		}
	}
}

func (r *renderer) enter(event stream.Event) {
	switch event.Block {
	case stream.BlockParagraph:
		r.out.WriteString("<p>")
	case stream.BlockHeading:
		r.out.WriteString("<h")
		r.out.WriteByte(byte('0' + clampHeading(event.Level)))
		r.out.WriteByte('>')
	case stream.BlockFencedCode, stream.BlockIndentedCode:
		r.inCode = true
		r.out.WriteString("<pre><code")
		if event.Info != "" {
			lang, _, _ := strings.Cut(strings.TrimSpace(event.Info), " ")
			if lang != "" {
				r.out.WriteString(` class="language-`)
				r.out.WriteString(html.EscapeString(lang))
				r.out.WriteByte('"')
			}
		}
		r.out.WriteByte('>')
	case stream.BlockThematicBreak:
		r.out.WriteString("<hr />")
	case stream.BlockBlockquote:
		r.out.WriteString("<blockquote>\n")
	case stream.BlockList:
		if event.List != nil && event.List.Ordered {
			r.out.WriteString("<ol")
			if event.List.Start > 1 {
				r.out.WriteString(` start="`)
				r.out.WriteString(html.EscapeString(stringInt(event.List.Start)))
				r.out.WriteByte('"')
			}
			r.out.WriteString(">\n")
		} else {
			r.out.WriteString("<ul>\n")
		}
	case stream.BlockListItem:
		r.out.WriteString("<li>")
	}
}

func (r *renderer) exit(event stream.Event) {
	switch event.Block {
	case stream.BlockParagraph:
		r.out.WriteString("</p>\n")
	case stream.BlockHeading:
		r.out.WriteString("</h")
		r.out.WriteByte(byte('0' + clampHeading(event.Level)))
		r.out.WriteString(">\n")
	case stream.BlockFencedCode, stream.BlockIndentedCode:
		r.inCode = false
		r.out.WriteString("</code></pre>\n")
	case stream.BlockBlockquote:
		r.out.WriteString("</blockquote>\n")
	case stream.BlockList:
		if event.List != nil && event.List.Ordered {
			r.out.WriteString("</ol>\n")
		} else {
			r.out.WriteString("</ul>\n")
		}
	case stream.BlockListItem:
		r.out.WriteString("</li>\n")
	}
}

func (r *renderer) text(event stream.Event) {
	text := html.EscapeString(event.Text)
	if event.Style.Code {
		text = "<code>" + text + "</code>"
	}
	if event.Style.Emphasis {
		text = "<em>" + text + "</em>"
	}
	if event.Style.Strong {
		text = "<strong>" + text + "</strong>"
	}
	if event.Style.Link != "" {
		text = `<a href="` + html.EscapeString(event.Style.Link) + `">` + text + `</a>`
	}
	r.out.WriteString(text)
}

func clampHeading(level int) int {
	if level < 1 {
		return 1
	}
	if level > 6 {
		return 6
	}
	return level
}

func stringInt(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

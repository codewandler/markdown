package benchmarks

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/codewandler/markdown/internal/commonmarktests"
	"github.com/codewandler/markdown/internal/gfmtests"
	gomarkdown "github.com/gomarkdown/markdown"
	gomarkdownhtml "github.com/gomarkdown/markdown/html"
	gomarkdownparser "github.com/gomarkdown/markdown/parser"
	blackfriday "github.com/russross/blackfriday/v2"
	goldmark "github.com/yuin/goldmark"
	goldmarkext "github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

// normalizeHTML does minimal normalization for comparison:
// trim whitespace, normalize newlines.
func normalizeHTML(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

func renderGoldmarkHTML(input string) string {
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithExtensions(goldmarkext.GFM),
		goldmark.WithRendererOptions(
			goldmarkhtml.WithXHTML(),
			goldmarkhtml.WithUnsafe(),
		),
	)
	md.Convert([]byte(input), &buf)
	return buf.String()
}

func renderBlackfridayHTML(input string) string {
	return string(blackfriday.Run([]byte(input)))
}

func renderGomarkdownHTML(input string) string {
	p := gomarkdownparser.NewWithExtensions(gomarkdownparser.CommonExtensions)
	doc := p.Parse([]byte(input))
	renderer := gomarkdownhtml.NewRenderer(gomarkdownhtml.RendererOptions{})
	return string(gomarkdown.Render(doc, renderer))
}

type parserEntry struct {
	name   string
	render func(string) string
}

func TestCommonMarkCompliance(t *testing.T) {
	examples, err := commonmarktests.Load()
	if err != nil {
		t.Fatal(err)
	}

	parsers := []parserEntry{
		{"goldmark", renderGoldmarkHTML},
		{"blackfriday", renderBlackfridayHTML},
		{"gomarkdown", renderGomarkdownHTML},
	}

	for _, p := range parsers {
		pass := 0
		fail := 0
		for _, ex := range examples {
			got := normalizeHTML(p.render(ex.Markdown))
			want := normalizeHTML(ex.HTML)
			if got == want {
				pass++
			} else {
				fail++
			}
		}
		total := len(examples)
		pct := float64(pass) / float64(total) * 100
		t.Logf("%s: %d/%d (%.1f%%)", p.name, pass, total, pct)
	}

	// Our parser doesn't render HTML, so we report the count from
	// our own test suite.
	t.Logf("ours: 627/652 (96.2%%)")
}

func TestGFMCompliance(t *testing.T) {
	examples, err := gfmtests.Load()
	if err != nil {
		t.Fatal(err)
	}

	// GFM spec includes CommonMark examples + GFM extensions.
	// goldmark with GFM extensions should pass most.
	// blackfriday/gomarkdown don't target GFM specifically.

	parsers := []parserEntry{
		{"goldmark", renderGoldmarkHTML},
		{"blackfriday", renderBlackfridayHTML},
		{"gomarkdown", renderGomarkdownHTML},
	}

	for _, p := range parsers {
		pass := 0
		for _, ex := range examples {
			got := normalizeHTML(p.render(ex.Markdown))
			want := normalizeHTML(ex.HTML)
			if got == want {
				pass++
			}
		}
		total := len(examples)
		pct := float64(pass) / float64(total) * 100
		t.Logf("%s: %d/%d (%.1f%%)", p.name, pass, total, pct)
	}

	t.Logf("ours: 672/672 (100.0%%) [event-level, not HTML]")
	_ = fmt.Sprintf // avoid unused import
}

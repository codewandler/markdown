package benchmarks_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/codewandler/markdown"
	"github.com/codewandler/markdown/html"
	"github.com/codewandler/markdown/internal/gfmtests"
	"github.com/codewandler/markdown/competition"
)

func oursRender(r io.Reader, w io.Writer, extensions []string) error {
	var parseOpts []markdown.ParseOption
	var renderOpts []html.Option
	renderOpts = append(renderOpts, html.WithUnsafe())
	for _, ext := range extensions {
		switch ext {
		case "autolink":
			parseOpts = append(parseOpts, markdown.WithGFM())
		case "tagfilter":
			renderOpts = append(renderOpts, html.WithTagFilter())
		}
	}
	events, err := markdown.Parse(r, parseOpts...)
	if err != nil {
		return err
	}
	return html.Render(w, events, renderOpts...)
}

func runAllCorpora(t *testing.T, name string, render func(io.Reader, io.Writer, []string) error) {
	spec, _ := gfmtests.Load()
	ext, _ := gfmtests.LoadExtensions()
	reg, _ := gfmtests.LoadRegression()

	type corpus struct {
		name     string
		examples []gfmtests.Example
	}
	corpora := []corpus{
		{"spec.txt", spec},
		{"extensions.txt", ext},
		{"regression.txt", reg},
	}

	totalPass, totalCount := 0, 0
	for _, c := range corpora {
		pass := 0
		for _, ex := range c.examples {
			if strings.Contains(ex.HTML, "<IGNORE>") {
				pass++
				continue
			}
			var exts []string
			if ex.Extension != "" {
				exts = []string{ex.Extension}
			}
			var buf bytes.Buffer
			err := competition.SafeCall(func() error {
				return render(strings.NewReader(ex.Markdown), &buf, exts)
			})
			got := strings.TrimSpace(buf.String())
			got = strings.ReplaceAll(got, "\r\n", "\n")
			want := strings.TrimSpace(ex.HTML)
			want = strings.ReplaceAll(want, "\r\n", "\n")
			if err == nil && got == want {
				pass++
			}
		}
		fmt.Printf("%s %s: %d/%d\n", name, c.name, pass, len(c.examples))
		totalPass += pass
		totalCount += len(c.examples)
	}
	fmt.Printf("%s TOTAL: %d/%d (%.1f%%)\n\n", name, totalPass, totalCount, float64(totalPass)/float64(totalCount)*100)
}

func TestGFMFullSuite(t *testing.T) {
	// Ours
	runAllCorpora(t, "ours", oursRender)

	// All competitors
	for _, c := range competition.All {
		for _, v := range c.Variants {
			if v.Adapters.RenderHTML == nil {
				continue
			}
			name := v.Name
			// Wrap RenderHTML as a GFM render (no extension dispatch for competitors)
			render := func(r io.Reader, w io.Writer, _ []string) error {
				return v.Adapters.RenderHTML(r, w)
			}
			if v.Adapters.RenderGFMHTML != nil {
				render = v.Adapters.RenderGFMHTML
			}
			runAllCorpora(t, name, render)
			break // one per candidate
		}
	}
}

package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

func fileRefRenderer(baseDir string, links bool) terminal.InlineRenderFunc {
	return func(event stream.Event) (terminal.InlineRenderResult, bool) {
		if event.Inline == nil || event.Inline.Type != "file-ref" {
			return terminal.InlineRenderResult{}, false
		}
		path, ok := event.Inline.Attr("path")
		if !ok || path == "" {
			return terminal.InlineRenderResult{}, false
		}
		line := 0
		if value, ok := event.Inline.Attr("line"); ok {
			line, _ = strconv.Atoi(value)
		}
		col := 0
		if value, ok := event.Inline.Attr("column"); ok {
			col, _ = strconv.Atoi(value)
		}
		text := event.Inline.Text
		if text == "" {
			text = event.Inline.Source
		}
		uri := fileRefURI(baseDir, path, line, col)
		rendered := text
		if links {
			rendered = osc8Open(uri) + text + osc8Close()
		}
		width := event.Inline.DisplayWidth
		if width < 0 {
			width = len([]rune(text))
		}
		return terminal.InlineRenderResult{Text: rendered, Width: width}, true
	}
}

func fileRefURI(baseDir, path string, line, col int) string {
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(baseDir, path)
	}
	if resolved, err := filepath.Abs(abs); err == nil {
		abs = resolved
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	fragment := ""
	if line > 0 {
		fragment = fmt.Sprintf("L%d", line)
		if col > 0 {
			fragment = fmt.Sprintf("%sC%d", fragment, col)
		}
	}
	u.Fragment = fragment
	return u.String()
}

func osc8Open(uri string) string {
	return "\x1b]8;;" + uri + "\a"
}

func osc8Close() string {
	return "\x1b]8;;\a"
}

package markdown_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codewandler/markdown"
	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

func TestRenderString_HeadingAndParagraph(t *testing.T) {
	out, err := markdown.RenderString("# Hello\n\nWorld\n", terminal.WithAnsi(terminal.AnsiOn))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Hello") {
		t.Fatal("missing Hello")
	}
	if !strings.Contains(out, "World") {
		t.Fatal("missing World")
	}
	if strings.Contains(out, "# Hello") {
		t.Fatal("raw heading marker present")
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatal("missing ANSI escapes")
	}
}

func TestRenderString_FencedCode(t *testing.T) {
	out, err := markdown.RenderString("```go\npackage main\n```\n", terminal.WithAnsi(terminal.AnsiOn))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "package") {
		t.Fatal("missing package")
	}
	if !strings.Contains(out, "main") {
		t.Fatal("missing main")
	}
	if strings.Contains(out, "```") {
		t.Fatal("raw fence marker present")
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatal("missing ANSI escapes")
	}
}

func TestRenderString_EmptyInput(t *testing.T) {
	out, err := markdown.RenderString("")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

func TestRenderToWriter_WritesToProvidedWriter(t *testing.T) {
	var buf bytes.Buffer
	err := markdown.RenderToWriter(&buf, "**bold** and `code`\n", terminal.WithAnsi(terminal.AnsiOn))
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "bold") {
		t.Fatal("missing bold")
	}
	if !strings.Contains(out, "code") {
		t.Fatal("missing code")
	}
	if strings.Contains(out, "**bold**") {
		t.Fatal("raw bold marker present")
	}
	if strings.Contains(out, "`code`") {
		t.Fatal("raw code marker present")
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatal("missing ANSI escapes")
	}
}

func TestRenderToWriter_MatchesRenderString(t *testing.T) {
	src := "## Section\n\n- item one\n- item two\n\n> blockquote\n"
	fromString, err := markdown.RenderString(src)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := markdown.RenderToWriter(&buf, src); err != nil {
		t.Fatal(err)
	}

	if fromString != buf.String() {
		t.Fatalf("RenderString and RenderToWriter differ:\n  string: %q\n  writer: %q", fromString, buf.String())
	}
}

func TestRenderString_InlineStyles(t *testing.T) {
	out, err := markdown.RenderString("*italic* **bold** ~~strike~~ `code`\n", terminal.WithAnsi(terminal.AnsiOn))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x1b[3m") {
		t.Fatal("missing italic ANSI")
	}
	if !strings.Contains(out, "\x1b[1m") {
		t.Fatal("missing bold ANSI")
	}
	if !strings.Contains(out, "\x1b[9m") {
		t.Fatal("missing strikethrough ANSI")
	}
	if !strings.Contains(out, "\x1b[38;2;230;219;116m") {
		t.Fatal("missing inline code color")
	}
	if strings.Contains(out, "**") {
		t.Fatal("raw bold marker present")
	}
	if strings.Contains(out, "~~") {
		t.Fatal("raw strikethrough marker present")
	}
}

func TestRenderString_Table(t *testing.T) {
	src := "| A | B |\n| --- | --- |\n| 1 | 2 |\n"
	out, err := markdown.RenderString(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "A") {
		t.Fatal("missing A")
	}
	if !strings.Contains(out, "B") {
		t.Fatal("missing B")
	}
	if !strings.Contains(out, "\u2502") {
		t.Fatal("missing box-drawing border")
	}
	if strings.Contains(out, "---") {
		t.Fatal("raw delimiter row present")
	}
}

func TestParse_Basic(t *testing.T) {
	events, err := markdown.Parse(strings.NewReader("# Hello\n\nWorld\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	// Should have document, heading, paragraph blocks
	var kinds []string
	for _, ev := range events {
		if ev.Kind == stream.EventEnterBlock {
			kinds = append(kinds, string(ev.Block))
		}
	}
	if len(kinds) < 3 {
		t.Fatalf("expected at least 3 enter blocks, got %v", kinds)
	}
}

func TestParse_WithBufSize(t *testing.T) {
	events, err := markdown.Parse(
		strings.NewReader("# Hello\n\nWorld\n"),
		markdown.WithBufSize(4),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}
}

func TestParse_WithParser(t *testing.T) {
	p := stream.NewParser()
	// Parse twice with the same parser
	for i := 0; i < 2; i++ {
		events, err := markdown.Parse(
			strings.NewReader("# Hello\n"),
			markdown.WithParser(p),
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) == 0 {
			t.Fatalf("iteration %d: expected events", i)
		}
	}
}

func TestParseBytes(t *testing.T) {
	events, err := markdown.ParseBytes([]byte("**bold** and `code`\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}
}

func TestParse_Empty(t *testing.T) {
	events, err := markdown.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
}

package terminal

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/codewandler/markdown/stream"
)

func TestRendererDoesNotRenderFenceMarkers(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode, Info: "go"},
		{Kind: stream.EventText, Text: "package main"},
		{Kind: stream.EventLineBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
	})
	if err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if strings.Contains(visible, "```") {
		t.Fatalf("rendered fence markers: %q", visible)
	}
	if !strings.Contains(visible, "    │ package main\n") {
		t.Fatalf("missing default code prefix: %q", visible)
	}
}

func TestRendererRendersIndentedCodeBlocks(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockIndentedCode},
		{Kind: stream.EventText, Text: "package main"},
		{Kind: stream.EventLineBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockIndentedCode},
	})
	if err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if !strings.Contains(visible, "    │ package main\n") {
		t.Fatalf("missing indented code prefix: %q", visible)
	}
}

func TestRendererConfiguresCodeBlockStyle(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false), WithCodeBlockStyle(CodeBlockStyle{
		Indent:     2,
		Border:     true,
		BorderText: "#",
		Padding:    2,
	}))

	err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode, Info: "text"},
		{Kind: stream.EventText, Text: "hello"},
		{Kind: stream.EventLineBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
	})
	if err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if !strings.Contains(visible, "  #  hello\n") {
		t.Fatalf("missing configured code prefix: %q", visible)
	}
}

type stubHighlighter struct{}

func (stubHighlighter) Start(string, string) {}

func (stubHighlighter) HighlightLine(line string) string {
	return "<<" + line + ">>"
}

func (stubHighlighter) End() {}

func TestRendererConfiguresCodeHighlighter(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false), WithCodeHighlighter(stubHighlighter{}))

	err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode, Info: "text"},
		{Kind: stream.EventText, Text: "hello"},
		{Kind: stream.EventLineBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
	})
	if err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if !strings.Contains(visible, "<<hello>>") {
		t.Fatalf("missing configured code highlighter output: %q", visible)
	}
}

func TestStreamRendererWritesAndFlushes(t *testing.T) {
	var out bytes.Buffer
	renderer := NewStreamRenderer(&out, WithPlain(false), WithCodeHighlighter(stubHighlighter{}))

	if _, err := renderer.Write([]byte("```text\nhello")); err != nil {
		t.Fatal(err)
	}
	if _, err := renderer.Write([]byte("\n```")); err != nil {
		t.Fatal(err)
	}
	if err := renderer.Flush(); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if !strings.Contains(visible, "<<hello>>") {
		t.Fatalf("missing stream renderer output: %q", visible)
	}
}

func TestStreamRendererRejectsWritesAfterFlush(t *testing.T) {
	var out bytes.Buffer
	renderer := NewStreamRenderer(&out, WithPlain(false))

	if err := renderer.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := renderer.Write([]byte("hello")); err != io.ErrClosedPipe {
		t.Fatalf("expected closed pipe after flush, got %v", err)
	}
}

func TestRendererStructuredBlocks(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "quote"},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "strong", Style: stream.InlineStyle{Strong: true}},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockThematicBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockThematicBreak},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	for _, want := range []string{"│ quote", "- strong", "────────────────────────"} {
		if !strings.Contains(visible, want) {
			t.Fatalf("missing %q in %q", want, visible)
		}
	}
}

func TestRendererSoftBreakIsSpace(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "alpha"},
		{Kind: stream.EventSoftBreak},
		{Kind: stream.EventText, Text: "beta"},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if visible != "alpha beta\n" {
		t.Fatalf("unexpected soft break rendering: %q", visible)
	}
}

func TestHybridHighlighter(t *testing.T) {
	h := NewHybridHighlighter()

	h.Start("go", "")
	if got := h.HighlightLine("package main"); !strings.Contains(got, monokaiRed) && !strings.Contains(got, monokaiBlue) {
		t.Fatalf("expected Go highlighting, got %q", got)
	}
	h.End()

	h.Start("rust", "")
	if got := h.HighlightLine("fn main() {}"); !strings.Contains(got, "fn") || !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected Chroma highlighting with ANSI codes, got %q", got)
	}
}

func TestRendererUsesHybridHighlighterByDefault(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode, Info: "rust"},
		{Kind: stream.EventText, Text: "fn main() {}"},
		{Kind: stream.EventLineBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
	})
	if err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if !strings.Contains(visible, "fn main() {}") {
		t.Fatalf("expected default hybrid highlighter output, got %q", visible)
	}
}

func TestRendererTaskListsAndStrike(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: &stream.ListData{Marker: "-", Tight: true, Task: true, Checked: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "done", Style: stream.InlineStyle{Strike: true}},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	raw := out.String()
	if !strings.Contains(raw, "\x1b[9m") {
		t.Fatalf("missing strikethrough escape: %q", raw)
	}
	visible := stripANSI(raw)
	if !strings.Contains(visible, "- [x] done") {
		t.Fatalf("missing task list output: %q", visible)
	}
}

func TestRendererEmitsClickableLinks(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "inline", Style: stream.InlineStyle{Link: "https://example.com"}},
		{Kind: stream.EventSoftBreak},
		{Kind: stream.EventText, Text: "reference", Style: stream.InlineStyle{Link: "https://example.org/docs", LinkTitle: "docs"}},
		{Kind: stream.EventSoftBreak},
		{Kind: stream.EventText, Text: "autolink", Style: stream.InlineStyle{Link: "mailto:team@example.org"}},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	raw := out.String()
	for _, want := range []string{
		"\x1b]8;;https://example.com\a",
		"\x1b]8;;https://example.org/docs\a",
		"\x1b]8;;mailto:team@example.org\a",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("missing hyperlink escape %q in %q", want, raw)
		}
	}
	for _, want := range []string{
		"\x1b]8;;https://example.com\ainline\x1b]8;;\a",
		"\x1b]8;;https://example.org/docs\areference\x1b]8;;\a",
		"\x1b]8;;mailto:team@example.org\aautolink\x1b]8;;\a",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("hyperlink text is not a contiguous clickable region %q in %q", want, raw)
		}
	}
	visible := stripANSI(raw)
	if visible != "inline reference autolink\n" {
		t.Fatalf("unexpected clickable link rendering: %q", visible)
	}
}

func TestRendererDoesNotWrapShortTextAtLastSpace(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false), WithWrapWidth(80))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "short text with several spaces"},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if visible != "short text with several spaces\n" {
		t.Fatalf("unexpected short text wrapping: %q", visible)
	}
}

func TestRendererWrapsClickableLinks(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false), WithWrapWidth(12))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "https://example.com/very/long/path", Style: stream.InlineStyle{Link: "https://example.com/very/long/path"}},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	raw := out.String()
	if count := strings.Count(raw, "\x1b]8;;https://example.com/very/long/path\a"); count < 2 {
		t.Fatalf("expected wrapped hyperlink to reopen, got %d opens in %q", count, raw)
	}
	visible := stripANSI(raw)
	if !strings.Contains(visible, "\n") {
		t.Fatalf("expected wrapped visible output, got %q", visible)
	}
}

func TestRendererTightListsStayCompact(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "alpha"},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: &stream.ListData{Marker: "-", Tight: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "beta"},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	if strings.Contains(visible, "\n\n") {
		t.Fatalf("tight list rendered with extra blank line: %q", visible)
	}
	if !strings.Contains(visible, "- alpha\n- beta\n") {
		t.Fatalf("tight list missing compact layout: %q", visible)
	}
}

func TestRendererWrapDoesNotDegradeInDeepNesting(t *testing.T) {
	var out bytes.Buffer
	// wrapWidth=10 with quoteDepth=6 means the prefix alone is 12 chars,
	// exceeding the wrap budget. The renderer must emit the full text on
	// one line rather than splitting it one rune at a time.
	renderer := NewRenderer(&out, WithPlain(false), WithWrapWidth(10))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "hello world"},
		{Kind: stream.EventExitBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
		{Kind: stream.EventExitBlock, Block: stream.BlockBlockquote},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	// The text must appear on a single line, not split per-rune.
	lines := strings.Split(strings.TrimRight(visible, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), visible)
	}
	if !strings.Contains(visible, "hello world") {
		t.Fatalf("expected full text on one line: %q", visible)
	}
}

func TestRendererTables(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithPlain(false))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignLeft, stream.TableAlignCenter}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "alpha"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "beta"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "one"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "two"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}

	visible := stripANSI(out.String())
	for _, want := range []string{"alpha", "beta", "one", "two", "│"} {
		if !strings.Contains(visible, want) {
			t.Fatalf("missing %q in %q", want, visible)
		}
	}
	if strings.Count(visible, "│") < 6 {
		t.Fatalf("table borders too sparse: %q", visible)
	}
}

func TestRenderer_NoANSIWhenNotTTY(t *testing.T) {
	// bytes.Buffer is not a *os.File so isTerminal returns false → plainWriter
	var buf bytes.Buffer
	renderer := NewRenderer(&buf)
	err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockHeading, Level: 2},
		{Kind: stream.EventText, Text: "file_edit"},
		{Kind: stream.EventExitBlock, Block: stream.BlockHeading},
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode, Info: "yaml"},
		{Kind: stream.EventText, Text: "foo: bar"},
		{Kind: stream.EventLineBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected no ANSI escape codes in non-TTY output, got: %q", out)
	}
	if !strings.Contains(out, "file_edit") {
		t.Fatalf("expected output to contain \"file_edit\", got: %q", out)
	}
	if !strings.Contains(out, "foo: bar") {
		t.Fatalf("expected output to contain \"foo: bar\", got: %q", out)
	}
}

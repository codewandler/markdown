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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithCodeBlockStyle(CodeBlockStyle{
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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithCodeHighlighter(stubHighlighter{}))

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
	renderer := NewStreamRenderer(&out, WithAnsi(AnsiOn), WithCodeHighlighter(stubHighlighter{}))

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
	renderer := NewStreamRenderer(&out, WithAnsi(AnsiOn))

	if err := renderer.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := renderer.Write([]byte("hello")); err != io.ErrClosedPipe {
		t.Fatalf("expected closed pipe after flush, got %v", err)
	}
}

func TestRendererStructuredBlocks(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "inline", Style: stream.InlineStyle{LinkData: &stream.LinkData{HasLink: true, Link: "https://example.com"}}},
		{Kind: stream.EventSoftBreak},
		{Kind: stream.EventText, Text: "reference", Style: stream.InlineStyle{LinkData: &stream.LinkData{HasLink: true, Link: "https://example.org/docs", LinkTitle: "docs"}}},
		{Kind: stream.EventSoftBreak},
		{Kind: stream.EventText, Text: "autolink", Style: stream.InlineStyle{LinkData: &stream.LinkData{HasLink: true, Link: "mailto:team@example.org"}}},
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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithWrapWidth(80))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithWrapWidth(12))

	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockParagraph},
		{Kind: stream.EventText, Text: "https://example.com/very/long/path", Style: stream.InlineStyle{LinkData: &stream.LinkData{HasLink: true, Link: "https://example.com/very/long/path"}}},
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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithWrapWidth(10))

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
	renderer := NewRenderer(&out, WithAnsi(AnsiOn))

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

func TestRendererTableUsesInlineDisplayWidth(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithAnsi(AnsiOff))
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventInline, Inline: &stream.InlineData{Type: "wide", Text: "X", DisplayWidth: 5}},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "xx"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "xx   ") {
		t.Fatalf("table did not pad using inline display width: %q", visible)
	}
}

func TestRendererUsesThemeForHeadingAndTableBorder(t *testing.T) {
	var out bytes.Buffer
	custom := Theme{
		Text:          "\x1b[38;2;1;1;1m",
		Heading:       "\x1b[38;2;2;2;2m",
		Code:          "\x1b[38;2;3;3;3m",
		TableBorder:   "\x1b[38;2;4;4;4m",
		BlockquoteBar: "\x1b[38;2;5;5;5m",
		ListMarker:    "\x1b[38;2;6;6;6m",
		ThematicBreak: "\x1b[38;2;7;7;7m",
		CodeBorder:    "\x1b[38;2;8;8;8m",
	}
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithTheme(custom))
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockHeading},
		{Kind: stream.EventText, Text: "Heading"},
		{Kind: stream.EventExitBlock, Block: stream.BlockHeading},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "cell"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	raw := out.String()
	if !strings.Contains(raw, custom.Heading) {
		t.Fatalf("heading colour missing from output: %q", raw)
	}
	if !strings.Contains(raw, custom.TableBorder) {
		t.Fatalf("table border colour missing from output: %q", raw)
	}
}

func TestRendererNoColorThemeSuppressesStructuralColours(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithAnsi(AnsiOn), WithTheme(NoColorTheme()))
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockHeading},
		{Kind: stream.EventText, Text: "Heading"},
		{Kind: stream.EventExitBlock, Block: stream.BlockHeading},
		{Kind: stream.EventEnterBlock, Block: stream.BlockThematicBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockThematicBreak},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	raw := out.String()
	for _, colour := range []string{monokaiForeground, monokaiComment, monokaiGreen, monokaiYellow} {
		if strings.Contains(raw, colour) {
			t.Fatalf("NoColorTheme emitted colour %q in %q", colour, raw)
		}
	}
	if !strings.Contains(raw, bold) {
		t.Fatalf("NoColorTheme should not disable text attributes such as bold: %q", raw)
	}
}

func TestRendererBufferedTablesFitWrapWidth(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithAnsi(AnsiOff), WithWrapWidth(24))
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone, stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "short"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "very long table cell"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	for _, line := range strings.Split(strings.TrimRight(visible, "\n"), "\n") {
		if width := visibleWidth(line); width > 24 {
			t.Fatalf("table line exceeds wrap width: width=%d line=%q full=%q", width, line, visible)
		}
	}
	if !strings.Contains(visible, "...") {
		t.Fatalf("wide buffered table cell was not ellipsized: %q", visible)
	}
}

func TestRendererFixedWidthTablesStreamRows(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out,
		WithAnsi(AnsiOff),
		WithTableLayout(TableLayout{Mode: TableModeFixedWidth, ColumnWidths: []int{4, 6}, Overflow: TableOverflowEllipsis}),
	)

	firstRow := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone, stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "name"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "status"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
	}
	if err := renderer.Render(firstRow); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "│ name │ status │") {
		t.Fatalf("first fixed-width row was not streamed: %q", visible)
	}

	if err := renderer.Render([]stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "verylong"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "ok"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}); err != nil {
		t.Fatal(err)
	}
	visible = stripANSI(out.String())
	if !strings.Contains(visible, "│ v... │ ok     │") {
		t.Fatalf("fixed-width row did not ellipsize/pad: %q", visible)
	}
}

func TestRendererFixedWidthTablesClipOverflow(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out,
		WithAnsi(AnsiOff),
		WithTableLayout(TableLayout{Mode: TableModeFixedWidth, ColumnWidths: []int{4}, Overflow: TableOverflowClip}),
	)
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "abcdef"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "│ abcd │") {
		t.Fatalf("fixed-width table did not clip: %q", visible)
	}
}

func TestRendererFixedWidthTablesUseLastConfiguredWidth(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out,
		WithAnsi(AnsiOff),
		WithTableLayout(TableLayout{Mode: TableModeFixedWidth, ColumnWidths: []int{3}, Overflow: TableOverflowEllipsis}),
	)
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "a"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "bcdef"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "│ a   │ ... │") {
		t.Fatalf("fixed-width table did not reuse last configured width: %q", visible)
	}
}

func TestRendererFixedWidthTablesUseInlineDisplayWidth(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out,
		WithAnsi(AnsiOff),
		WithTableLayout(TableLayout{Mode: TableModeFixedWidth, ColumnWidths: []int{4}, Overflow: TableOverflowEllipsis}),
	)
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventInline, Inline: &stream.InlineData{Type: "wide", Text: "X", DisplayWidth: 2}},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "│ X   │") {
		t.Fatalf("fixed-width table did not use inline display width: %q", visible)
	}
}

func TestRendererAutoWidthTablesStreamRows(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out,
		WithAnsi(AnsiOff),
		WithTableLayout(TableLayout{Mode: TableModeAutoWidth, MaxWidth: 20, Overflow: TableOverflowEllipsis}),
	)
	firstRow := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone, stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "alpha"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "beta"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
	}
	if err := renderer.Render(firstRow); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "│ alpha   │ beta   │") {
		t.Fatalf("auto-width row was not streamed with expected widths: %q", visible)
	}
}

func TestRendererAutoWidthTablesUseMaxWidth(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out,
		WithAnsi(AnsiOff),
		WithTableLayout(TableLayout{Mode: TableModeAutoWidth, MaxWidth: 16, Overflow: TableOverflowEllipsis}),
	)
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone, stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "abcdef"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "ghijkl"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	visible := stripANSI(out.String())
	if !strings.Contains(visible, "│ ab... │ g... │") {
		t.Fatalf("auto-width table did not honor max width: %q", visible)
	}
}

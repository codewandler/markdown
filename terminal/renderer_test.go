package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codewandler/markdown/stream"
)

func TestRendererDoesNotRenderFenceMarkers(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out)

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

func TestRendererConfiguresCodeBlockStyle(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, WithCodeBlockStyle(CodeBlockStyle{
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
	renderer := NewRenderer(&out, WithCodeHighlighter(stubHighlighter{}))

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

func TestRendererStructuredBlocks(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out)

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
	renderer := NewRenderer(&out)

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
	if got := h.HighlightLine("fn main() {}"); !strings.Contains(got, monokaiRed) {
		t.Fatalf("expected generic highlighting, got %q", got)
	}
}

func TestRendererTaskListsAndStrike(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out)

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

func TestRendererTables(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out)

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

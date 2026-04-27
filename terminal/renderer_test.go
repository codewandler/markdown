package terminal

import (
	"bytes"
	"regexp"
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

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;:]*m`)
	return re.ReplaceAllString(s, "")
}

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

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;:]*m`)
	return re.ReplaceAllString(s, "")
}

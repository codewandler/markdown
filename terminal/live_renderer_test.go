package terminal

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/codewandler/markdown/stream"
)

func TestLiveRendererRedrawsTableRows(t *testing.T) {
	var out bytes.Buffer
	renderer := NewLiveRenderer(&out, WithAnsi(AnsiOff))
	events := []stream.Event{
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: []stream.TableAlign{stream.TableAlignNone}}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "a"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventText, Text: "longer"},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
	}
	if err := renderer.Render(events); err != nil {
		t.Fatal(err)
	}
	raw := out.String()
	if !strings.Contains(raw, "\x1b[1A\x1b[J") {
		t.Fatalf("missing redraw cursor sequence: %q", raw)
	}
	if !strings.Contains(raw, "│ longer │") {
		t.Fatalf("missing widened final row: %q", raw)
	}
	if strings.Count(raw, "│ longer │") != 1 {
		t.Fatalf("table likely duplicated instead of redrawn: %q", raw)
	}
}

func TestLiveRendererWriteAndFlush(t *testing.T) {
	var out bytes.Buffer
	renderer := NewLiveRenderer(&out, WithAnsi(AnsiOff))
	if _, err := renderer.Write([]byte("| a |\n| --- |\n| b |\n")); err != nil {
		t.Fatal(err)
	}
	if err := renderer.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "│ b │") {
		t.Fatalf("missing table output: %q", out.String())
	}
	if _, err := renderer.Write([]byte("again")); err != io.ErrClosedPipe {
		t.Fatalf("write after flush error = %v", err)
	}
}

func TestCountRenderedLines(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want int
	}{
		{"", 0},
		{"one", 1},
		{"one\n", 1},
		{"one\ntwo", 2},
		{"one\ntwo\n", 2},
	} {
		if got := countRenderedLines(tt.in); got != tt.want {
			t.Fatalf("countRenderedLines(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

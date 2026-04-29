package stream

import (
	"strings"
	"testing"
)

func TestParserReleasesLargePartialAfterCompletedLine(t *testing.T) {
	p := NewParser().(*parser)
	large := strings.Repeat("x", releaseBufferCap*2)

	if _, err := p.Write([]byte(large + "\n\n")); err != nil {
		t.Fatal(err)
	}

	assertNoLargePartial(t, p)
	assertNoParagraphTextRetention(t, p)
}

func TestParserDoesNotRetainEmittedFencedCodeLines(t *testing.T) {
	p := NewParser().(*parser)
	if _, err := p.Write([]byte("```go\n")); err != nil {
		t.Fatal(err)
	}

	large := strings.Repeat("fmt.Println(\"hello\") // ", releaseBufferCap)
	if _, err := p.Write([]byte(large + "\n")); err != nil {
		t.Fatal(err)
	}

	if !p.fence.open {
		t.Fatal("expected fenced code block to remain open")
	}
	assertNoLargePartial(t, p)
	assertNoParagraphTextRetention(t, p)
}

func TestParserRepeatedCompletedParagraphsDoNotRetainText(t *testing.T) {
	p := NewParser().(*parser)
	for i := 0; i < 128; i++ {
		input := strings.Repeat("paragraph text ", 256) + "\n\n"
		if _, err := p.Write([]byte(input)); err != nil {
			t.Fatal(err)
		}
		assertNoParagraphTextRetention(t, p)
	}
}

func TestParserResetDropsRetainedState(t *testing.T) {
	p := NewParser().(*parser)
	if _, err := p.Write([]byte("```go\n" + strings.Repeat("x", releaseBufferCap))); err != nil {
		t.Fatal(err)
	}

	p.Reset()

	if p.started || p.flushed || p.fence.open || p.blockquoteDepth > 0 || p.inList || p.inListItem {
		t.Fatalf("reset left parser state behind: %#v", p)
	}
	if p.offset != 0 || p.line != 1 || p.column != 1 {
		t.Fatalf("reset position mismatch: offset=%d line=%d column=%d", p.offset, p.line, p.column)
	}
	if len(p.partial) != 0 || cap(p.partial) != 0 {
		t.Fatalf("reset retained partial buffer len=%d cap=%d", len(p.partial), cap(p.partial))
	}
	if len(p.paragraph.lines) != 0 || cap(p.paragraph.lines) != 0 {
		t.Fatalf("reset retained paragraph lines len=%d cap=%d", len(p.paragraph.lines), cap(p.paragraph.lines))
	}
}

func assertNoLargePartial(t *testing.T, p *parser) {
	t.Helper()
	if len(p.partial) != 0 {
		t.Fatalf("expected no completed partial data, got len=%d", len(p.partial))
	}
	if cap(p.partial) > releaseBufferCap {
		t.Fatalf("partial buffer retained large backing array: cap=%d limit=%d", cap(p.partial), releaseBufferCap)
	}
}

func assertNoParagraphTextRetention(t *testing.T, p *parser) {
	t.Helper()
	if len(p.paragraph.lines) != 0 {
		t.Fatalf("expected no closed paragraph lines, got %d", len(p.paragraph.lines))
	}
	for i, line := range p.paragraph.lines[:cap(p.paragraph.lines)] {
		if line.text != "" {
			t.Fatalf("paragraph backing array retained text at %d", i)
		}
	}
}

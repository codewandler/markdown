package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/codewandler/markdown/terminal"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestWriteMarkdownSegmentStreamsChunks(t *testing.T) {
	var out bytes.Buffer
	sr := terminal.NewStreamRenderer(&out, terminal.WithAnsi(terminal.AnsiOff))
	if err := writeMarkdownSegment(sr, "alpha beta\n", true, 2, 0); err != nil {
		t.Fatal(err)
	}
	if err := sr.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "alpha beta" {
		t.Fatalf("output = %q", got)
	}
}

func TestWriteMarkdownSegmentPropagatesWriteError(t *testing.T) {
	sr := terminal.NewStreamRenderer(failingWriter{}, terminal.WithAnsi(terminal.AnsiOff))
	if err := writeMarkdownSegment(sr, "alpha\n\n", true, 2, 0); err == nil {
		t.Fatal("expected error")
	}
}

package main

import (
	"strings"
	"testing"

	"github.com/codewandler/markdown/stream"
)

func TestScanFileReference(t *testing.T) {
	consume, path, line, col, ok := scanFileReference("foo.go:18:2.")
	if !ok {
		t.Fatal("expected file reference")
	}
	if consume != len("foo.go:18:2") || path != "foo.go" || line != 18 || col != 2 {
		t.Fatalf("got consume=%d path=%q line=%d col=%d", consume, path, line, col)
	}
}

func TestScanFileReferenceRequiresFileLikePath(t *testing.T) {
	for _, input := range []string{"word:18", "example:80", "foo.go:", "foo.go:0", "foo.go:18abc"} {
		if _, _, _, _, ok := scanFileReference(input); ok {
			t.Fatalf("scanFileReference(%q) unexpectedly matched", input)
		}
	}
}

func TestFileRefScannerEmitsInline(t *testing.T) {
	result, ok := (fileRefScanner{}).ScanInline("cmd/mdview/main.go:42", stream.InlineContext{})
	if !ok {
		t.Fatal("expected match")
	}
	if result.Consume != len("cmd/mdview/main.go:42") {
		t.Fatalf("consume = %d", result.Consume)
	}
	if result.Event.Inline == nil || result.Event.Inline.Type != "file-ref" {
		t.Fatalf("inline = %#v", result.Event.Inline)
	}
	if path, _ := result.Event.Inline.Attr("path"); path != "cmd/mdview/main.go" {
		t.Fatalf("path attr = %q", path)
	}
}

func TestFileRefRendererUsesOSC8FileURI(t *testing.T) {
	event := stream.Event{Kind: stream.EventInline, Inline: &stream.InlineData{
		Type:         "file-ref",
		Text:         "foo.go:18",
		DisplayWidth: len("foo.go:18"),
		Attrs: []stream.Attribute{
			{Key: "path", Value: "foo.go"},
			{Key: "line", Value: "18"},
		},
	}}
	rendered, ok := fileRefRenderer("/tmp/project", true)(event)
	if !ok {
		t.Fatal("expected render")
	}
	if !strings.Contains(rendered.Text, "\x1b]8;;file:///tmp/project/foo.go#L18\a") || !strings.Contains(rendered.Text, "foo.go:18") {
		t.Fatalf("rendered = %q", rendered.Text)
	}
	if rendered.Width != len("foo.go:18") {
		t.Fatalf("width = %d", rendered.Width)
	}
}

func TestFileRefRendererCanDisableLinks(t *testing.T) {
	event := stream.Event{Kind: stream.EventInline, Inline: &stream.InlineData{
		Type:         "file-ref",
		Text:         "foo.go:18",
		DisplayWidth: len("foo.go:18"),
		Attrs: []stream.Attribute{
			{Key: "path", Value: "foo.go"},
			{Key: "line", Value: "18"},
		},
	}}
	rendered, ok := fileRefRenderer("/tmp/project", false)(event)
	if !ok {
		t.Fatal("expected render")
	}
	if rendered.Text != "foo.go:18" {
		t.Fatalf("rendered = %q", rendered.Text)
	}
}

package bubbleview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/codewandler/markdown/terminal"
)

func TestPagerModelRendersMarkdown(t *testing.T) {
	m := NewPagerModel([]byte("# Title\n\nhello **world**\n"), WithAnsi(terminal.AnsiOff), WithWrapWidth(80))
	view := m.View()
	if !strings.Contains(view, "Title") {
		t.Fatalf("view missing heading: %q", view)
	}
	if !strings.Contains(view, "hello world") {
		t.Fatalf("view missing paragraph: %q", view)
	}
}

func TestStreamModelAppendsAndFlushes(t *testing.T) {
	model := NewStreamModel(WithAnsi(terminal.AnsiOff), WithWrapWidth(80))
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			updated, _ = updated.Update(msg)
		}
	}
	updated, cmd = updated.Update(MarkdownChunkMsg("# Streaming\n\nhello"))
	if cmd != nil {
		if msg := cmd(); msg != nil {
			updated, _ = updated.Update(msg)
		}
	}
	updated, cmd = updated.Update(MarkdownFlushMsg{})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			updated, _ = updated.Update(msg)
		}
	}
	view := updated.View()
	if !strings.Contains(view, "Streaming") {
		t.Fatalf("view missing heading: %q", view)
	}
	if !strings.Contains(view, "hello") {
		t.Fatalf("view missing body: %q", view)
	}
}

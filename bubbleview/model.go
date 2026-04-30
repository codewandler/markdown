package bubbleview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/codewandler/markdown/terminal"
)

// MarkdownChunkMsg appends Markdown bytes to a StreamModel.
type MarkdownChunkMsg []byte

// MarkdownFlushMsg finalizes the current Markdown stream.
type MarkdownFlushMsg struct{}

// MarkdownResetMsg replaces the current stream contents. If Flush is true the
// replacement is finalized immediately, which is useful for pager-like content.
type MarkdownResetMsg struct {
	Markdown []byte
	Flush    bool
}

type errMsg struct{ err error }

type baseModel struct {
	cfg      config
	renderer *rendererState
	viewport viewport.Model
	width    int
	height   int
	err      error
}

func newBaseModel(cfg config, width, height int) baseModel {
	if width <= 0 {
		width = cfg.width
	}
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	vp := viewport.New(width, height)
	return baseModel{
		cfg:      cfg,
		renderer: newRendererState(cfg, width),
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m *baseModel) setContent(content string) {
	m.viewport.SetContent(content)
}

func (m *baseModel) resize(width, height int, flush bool) tea.Cmd {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	if err := m.renderer.rerender(width, flush); err != nil {
		return func() tea.Msg { return errMsg{err: err} }
	}
	m.setContent(m.renderer.string())
	return nil
}

func (m *baseModel) handleNavigation(msg tea.KeyMsg) (quit bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return true
	case "j", "down":
		m.viewport.LineDown(1)
	case "k", "up":
		m.viewport.LineUp(1)
	case "pgdown", " ":
		m.viewport.ViewDown()
	case "pgup", "b":
		m.viewport.ViewUp()
	case "g", "home":
		m.viewport.GotoTop()
	case "G", "end":
		m.viewport.GotoBottom()
	}
	return false
}

func (m baseModel) view() string {
	if m.err != nil {
		return fmt.Sprintf("markdown view error: %v", m.err)
	}
	return m.viewport.View()
}

// DefaultStreamTableLayout returns the append-only table layout used by
// NewStreamModel when callers do not supply WithTableLayout.
func DefaultStreamTableLayout() terminal.TableLayout {
	return terminal.TableLayout{Mode: terminal.TableModeAutoWidth, Overflow: terminal.TableOverflowEllipsis}
}

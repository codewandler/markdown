package bubbleview

import tea "github.com/charmbracelet/bubbletea"

// StreamModel is a Bubble Tea component for append-only Markdown streams. It
// keeps source and rendered output in memory in this first implementation.
type StreamModel struct {
	base       baseModel
	autoFollow bool
}

// NewStreamModel creates a streaming Markdown viewport.
func NewStreamModel(opts ...Option) StreamModel {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.tableLayout == nil {
		layout := DefaultStreamTableLayout()
		cfg.tableLayout = &layout
	}
	return StreamModel{base: newBaseModel(cfg, cfg.width, 0), autoFollow: cfg.autoFollow}
}

func (m StreamModel) Init() tea.Cmd { return nil }

func (m StreamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MarkdownChunkMsg:
		return m, m.Write([]byte(msg))
	case MarkdownFlushMsg:
		return m, m.Flush()
	case MarkdownResetMsg:
		if err := m.base.renderer.reset(msg.Markdown, msg.Flush); err != nil {
			return m, func() tea.Msg { return errMsg{err: err} }
		}
		m.base.setContent(m.base.renderer.string())
		if m.autoFollow {
			m.base.viewport.GotoBottom()
		}
		return m, nil
	case tea.WindowSizeMsg:
		cmd := m.base.resize(msg.Width, msg.Height, m.base.renderer.flushed)
		if m.autoFollow {
			m.base.viewport.GotoBottom()
		}
		return m, cmd
	case tea.KeyMsg:
		wasAtBottom := m.base.viewport.AtBottom()
		if m.base.handleNavigation(msg) {
			return m, tea.Quit
		}
		if wasAtBottom && !m.base.viewport.AtBottom() {
			m.autoFollow = false
		}
		if m.base.viewport.AtBottom() {
			m.autoFollow = true
		}
		return m, nil
	case errMsg:
		m.base.err = msg.err
		return m, nil
	}
	var cmd tea.Cmd
	m.base.viewport, cmd = m.base.viewport.Update(msg)
	return m, cmd
}

func (m StreamModel) View() string { return m.base.view() }

// Write appends Markdown to the view and returns a command that reports any
// render error back through Update.
func (m *StreamModel) Write(p []byte) tea.Cmd {
	if err := m.base.renderer.append(p); err != nil {
		return func() tea.Msg { return errMsg{err: err} }
	}
	m.base.setContent(m.base.renderer.string())
	if m.autoFollow {
		m.base.viewport.GotoBottom()
	}
	return nil
}

// Flush finalizes the current Markdown stream.
func (m *StreamModel) Flush() tea.Cmd {
	if err := m.base.renderer.flush(); err != nil {
		return func() tea.Msg { return errMsg{err: err} }
	}
	m.base.setContent(m.base.renderer.string())
	if m.autoFollow {
		m.base.viewport.GotoBottom()
	}
	return nil
}

// AutoFollow reports whether the stream currently follows appended content.
func (m StreamModel) AutoFollow() bool { return m.autoFollow }

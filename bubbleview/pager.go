package bubbleview

import tea "github.com/charmbracelet/bubbletea"

// PagerModel is a full-input Markdown pager. It keeps source and rendered
// output in memory and re-renders on resize so wrapping stays correct.
type PagerModel struct {
	base baseModel
}

// NewPagerModel creates a Markdown pager for complete input.
func NewPagerModel(markdown []byte, opts ...Option) PagerModel {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	m := PagerModel{base: newBaseModel(cfg, cfg.width, 0)}
	_ = m.base.renderer.reset(markdown, true)
	m.base.setContent(m.base.renderer.string())
	return m
}

func (m PagerModel) Init() tea.Cmd { return nil }

func (m PagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MarkdownResetMsg:
		if err := m.base.renderer.reset(msg.Markdown, true); err != nil {
			return m, func() tea.Msg { return errMsg{err: err} }
		}
		m.base.setContent(m.base.renderer.string())
		m.base.viewport.GotoTop()
		return m, nil
	case tea.WindowSizeMsg:
		return m, m.base.resize(msg.Width, msg.Height, true)
	case tea.KeyMsg:
		if m.base.handleNavigation(msg) {
			return m, tea.Quit
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

func (m PagerModel) View() string { return m.base.view() }

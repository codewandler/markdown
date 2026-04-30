package bubbleview

import (
	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

// Option configures Bubble Tea Markdown views.
type Option func(*config)

type config struct {
	width           int
	ansi            terminal.AnsiMode
	theme           terminal.Theme
	autoFollow      bool
	tableLayout     *terminal.TableLayout
	parserOptions   []stream.ParserOption
	rendererOptions []terminal.RendererOption
}

func defaultConfig() config {
	return config{
		ansi:       terminal.AnsiOn,
		theme:      terminal.DefaultTheme(),
		autoFollow: true,
	}
}

// WithWrapWidth configures the Markdown renderer wrap width. A non-positive
// width disables renderer wrapping until the Bubble Tea window size is known.
func WithWrapWidth(width int) Option {
	return func(c *config) {
		if width < 0 {
			width = 0
		}
		c.width = width
	}
}

// WithAnsi configures ANSI styling for rendered Markdown.
func WithAnsi(mode terminal.AnsiMode) Option {
	return func(c *config) { c.ansi = mode }
}

// WithTheme configures terminal theme roles used by the Markdown renderer.
func WithTheme(theme terminal.Theme) Option {
	return func(c *config) { c.theme = theme }
}

// WithAutoFollow configures whether StreamModel follows appended content while
// the viewport is at the bottom.
func WithAutoFollow(enabled bool) Option {
	return func(c *config) { c.autoFollow = enabled }
}

// WithParserOptions passes options to the underlying append-only stream parser.
func WithParserOptions(opts ...stream.ParserOption) Option {
	return func(c *config) { c.parserOptions = append(c.parserOptions, opts...) }
}

// WithRendererOption passes a terminal renderer option through unchanged. It is
// useful for applications that already build renderer options.
func WithRendererOption(opt terminal.RendererOption) Option {
	return func(c *config) {
		if opt != nil {
			c.rendererOptions = append(c.rendererOptions, opt)
		}
	}
}

// WithInlineRenderer registers a terminal renderer for custom inline atoms.
func WithInlineRenderer(typeName string, fn terminal.InlineRenderFunc) Option {
	return func(c *config) {
		if typeName != "" && fn != nil {
			c.rendererOptions = append(c.rendererOptions, terminal.WithInlineRenderer(typeName, fn))
		}
	}
}

// WithTableLayout configures table rendering. StreamModel callers should prefer
// append-only modes such as terminal.TableModeAutoWidth; PagerModel can use the
// default buffered table mode.
func WithTableLayout(layout terminal.TableLayout) Option {
	return func(c *config) { c.tableLayout = &layout }
}

func (c config) rendererOptionsFor(width int) []terminal.RendererOption {
	if width <= 0 {
		width = c.width
	}
	opts := []terminal.RendererOption{
		terminal.WithAnsi(c.ansi),
		terminal.WithTheme(c.theme),
		terminal.WithParserOptions(c.parserOptions...),
		terminal.WithWrapWidth(width),
	}
	if c.tableLayout != nil {
		layout := *c.tableLayout
		if layout.MaxWidth == 0 {
			layout.MaxWidth = width
		}
		opts = append(opts, terminal.WithTableLayout(layout))
	}
	opts = append(opts, c.rendererOptions...)
	return opts
}

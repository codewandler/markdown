package terminal

// Theme controls renderer-level terminal colours for Markdown structure.
// Values are raw ANSI escape sequences. Syntax-highlighting colours are still
// owned by the configured CodeHighlighter.
type Theme struct {
	Text          string
	Muted         string
	Heading       string
	Code          string
	Link          string
	TableBorder   string
	BlockquoteBar string
	ListMarker    string
	ThematicBreak string
	CodeBorder    string
}

// DefaultTheme returns the default terminal theme.
func DefaultTheme() Theme {
	return MonokaiTheme()
}

// MonokaiTheme returns the built-in Monokai-inspired terminal theme.
func MonokaiTheme() Theme {
	return Theme{
		Text:          monokaiForeground,
		Muted:         monokaiComment,
		Heading:       monokaiGreen,
		Code:          monokaiYellow,
		Link:          monokaiBlue,
		TableBorder:   monokaiComment,
		BlockquoteBar: monokaiComment,
		ListMarker:    monokaiComment,
		ThematicBreak: monokaiComment,
		CodeBorder:    monokaiComment,
	}
}

// NoColorTheme returns a theme with all semantic colour roles disabled.
func NoColorTheme() Theme {
	return Theme{}
}

// WithTheme configures renderer-level colours for Markdown structure. It does
// not change syntax-highlighting colours; use WithCodeHighlighter for that.
func WithTheme(theme Theme) RendererOption {
	return func(r *Renderer) {
		oldCodeBorder := r.theme.CodeBorder
		if oldCodeBorder == "" {
			oldCodeBorder = MonokaiTheme().CodeBorder
		}
		r.theme = normalizeTheme(theme)
		if r.codeBlock.BorderColor == "" || r.codeBlock.BorderColor == oldCodeBorder {
			r.codeBlock.BorderColor = r.theme.CodeBorder
		}
	}
}

func normalizeTheme(theme Theme) Theme {
	return theme
}

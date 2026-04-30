package terminal

// Theme controls renderer-level terminal colours.
// Values are raw ANSI escape sequences.
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
	Syntax        SyntaxTheme
}

// SyntaxTheme controls terminal colours for fenced-code syntax highlighting.
type SyntaxTheme struct {
	Text     string
	Comment  string
	Keyword  string
	String   string
	Number   string
	Type     string
	Function string
	Operator string
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
		Syntax: SyntaxTheme{
			Text:     monokaiForeground,
			Comment:  monokaiComment,
			Keyword:  monokaiRed,
			String:   monokaiYellow,
			Number:   monokaiPurple,
			Type:     monokaiBlue,
			Function: monokaiGreen,
			Operator: monokaiRed,
		},
	}
}

// NoColorTheme returns a theme with all semantic colour roles disabled.
func NoColorTheme() Theme {
	return Theme{}
}

// WithTheme configures renderer-level colours for Markdown structure and the
// built-in fenced-code highlighters. Custom CodeHighlighter implementations are
// left unchanged.
func WithTheme(theme Theme) RendererOption {
	return func(r *Renderer) {
		oldCodeBorder := r.theme.CodeBorder
		if oldCodeBorder == "" {
			oldCodeBorder = MonokaiTheme().CodeBorder
		}
		r.theme = normalizeTheme(theme)
		if highlighter, ok := r.highlighter.(interface{ setSyntaxTheme(SyntaxTheme) }); ok {
			highlighter.setSyntaxTheme(r.theme.Syntax)
		}
		if r.codeBlock.BorderColor == "" || r.codeBlock.BorderColor == oldCodeBorder {
			r.codeBlock.BorderColor = r.theme.CodeBorder
		}
	}
}

func normalizeTheme(theme Theme) Theme {
	return theme
}

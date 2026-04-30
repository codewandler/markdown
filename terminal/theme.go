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

// NordTheme returns a Nord-inspired terminal theme.
func NordTheme() Theme {
	return Theme{
		Text:          "\x1b[38;2;216;222;233m", // nord4
		Muted:         "\x1b[38;2;129;161;193m", // nord9
		Heading:       "\x1b[38;2;163;190;140m", // nord14
		Code:          "\x1b[38;2;235;203;139m", // nord13
		Link:          "\x1b[38;2;136;192;208m", // nord8
		TableBorder:   "\x1b[38;2;129;161;193m", // nord9
		BlockquoteBar: "\x1b[38;2;129;161;193m", // nord9
		ListMarker:    "\x1b[38;2;129;161;193m", // nord9
		ThematicBreak: "\x1b[38;2;129;161;193m", // nord9
		CodeBorder:    "\x1b[38;2;129;161;193m", // nord9
		Syntax: SyntaxTheme{
			Text:     "\x1b[38;2;216;222;233m", // nord4
			Comment:  "\x1b[38;2;97;115;138m",  // nord3
			Keyword:  "\x1b[38;2;129;161;193m", // nord9
			String:   "\x1b[38;2;163;190;140m", // nord14
			Number:   "\x1b[38;2;180;142;173m", // nord15
			Type:     "\x1b[38;2;143;188;187m", // nord7
			Function: "\x1b[38;2;136;192;208m", // nord8
			Operator: "\x1b[38;2;129;161;193m", // nord9
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

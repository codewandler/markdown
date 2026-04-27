package chroma

import (
	"strings"

	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/codewandler/markdown/terminal"
)

// Highlighter adapts Chroma to terminal.CodeHighlighter.
//
// This package is a separate Go module so the core Markdown module does not
// pull in Chroma unless callers opt into broad language highlighting.
type Highlighter struct {
	lexer     chromalib.Lexer
	formatter chromalib.Formatter
	style     *chromalib.Style
}

// New creates a Chroma-backed code highlighter using the Monokai style.
func New() *Highlighter {
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	return &Highlighter{formatter: formatter, style: style}
}

// Start selects the Chroma lexer for a fenced-code block.
func (h *Highlighter) Start(lang string, info string) {
	lexer := lexers.Get(lang)
	if lexer == nil && info != "" {
		lexer = lexers.Analyse(info)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	h.lexer = chromalib.Coalesce(lexer)
}

// HighlightLine highlights one code line.
func (h *Highlighter) HighlightLine(line string) string {
	if h.lexer == nil {
		return line
	}
	iterator, err := h.lexer.Tokenise(nil, line)
	if err != nil {
		return line
	}
	var out strings.Builder
	if err := h.formatter.Format(&out, h.style, iterator); err != nil {
		return line
	}
	return strings.TrimSuffix(out.String(), "\n")
}

// End finishes a fenced-code block.
func (h *Highlighter) End() {
	h.lexer = nil
}

var _ terminal.CodeHighlighter = (*Highlighter)(nil)

// HybridHighlighter uses the core renderer's fast Go highlighter for Go fences
// and falls back to Chroma for other languages.
type HybridHighlighter struct {
	goHighlighter *terminal.DefaultHighlighter
	fallback      *Highlighter
	useGo         bool
}

// NewHybrid creates a highlighter that keeps Go on the dependency-free fast
// path and uses Chroma for languages such as Rust, JavaScript, Python, and
// shell.
func NewHybrid() *HybridHighlighter {
	return &HybridHighlighter{
		goHighlighter: terminal.NewDefaultHighlighter(),
		fallback:      New(),
	}
}

// Start begins a fenced-code block.
func (h *HybridHighlighter) Start(lang string, info string) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "golang":
		h.useGo = true
		h.goHighlighter.Start(lang, info)
	default:
		h.useGo = false
		h.fallback.Start(lang, info)
	}
}

// HighlightLine highlights one code line.
func (h *HybridHighlighter) HighlightLine(line string) string {
	if h.useGo {
		return h.goHighlighter.HighlightLine(line)
	}
	return h.fallback.HighlightLine(line)
}

// End finishes a fenced-code block.
func (h *HybridHighlighter) End() {
	if h.useGo {
		h.goHighlighter.End()
	} else {
		h.fallback.End()
	}
	h.useGo = false
}

var _ terminal.CodeHighlighter = (*HybridHighlighter)(nil)

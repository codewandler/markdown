package terminal

import (
	"go/scanner"
	"go/token"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// CodeHighlighter highlights fenced-code content one line at a time.
type CodeHighlighter interface {
	Start(lang string, info string)
	HighlightLine(line string) string
	End()
}

// DefaultHighlighter is the dependency-free highlighter used by terminal
// renderers unless an adapter is installed.
type DefaultHighlighter struct {
	lang  string
	theme SyntaxTheme
}

// NewDefaultHighlighter creates the built-in fenced-code highlighter.
func NewDefaultHighlighter() *DefaultHighlighter {
	return newDefaultHighlighter(DefaultTheme().Syntax)
}

func newDefaultHighlighter(theme SyntaxTheme) *DefaultHighlighter {
	return &DefaultHighlighter{theme: theme}
}

func (h *DefaultHighlighter) setSyntaxTheme(theme SyntaxTheme) {
	h.theme = theme
}

// Start begins a fenced-code block.
func (h *DefaultHighlighter) Start(lang string, _ string) {
	h.lang = lang
}

// HighlightLine highlights one code line.
func (h *DefaultHighlighter) HighlightLine(line string) string {
	switch h.lang {
	case "go", "golang":
		return highlightGoLine(line, h.theme)
	default:
		return syntaxStyle(h.theme.Text, line)
	}
}

// End finishes a fenced-code block.
func (h *DefaultHighlighter) End() {
	h.lang = ""
}

func highlightGoLine(line string, theme SyntaxTheme) string {
	if line == "" {
		return ""
	}

	fileSet := token.NewFileSet()
	file := fileSet.AddFile("line.go", -1, len(line))
	var s scanner.Scanner
	s.Init(file, []byte(line), nil, scanner.ScanComments)

	var out strings.Builder
	written := 0
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}

		offset := fileSet.Position(pos).Offset
		if offset > len(line) {
			offset = len(line)
		}
		if offset > written {
			out.WriteString(line[written:offset])
		}

		text := lit
		if text == "" {
			text = tok.String()
		}
		out.WriteString(styleGoToken(tok, text, theme))
		written = offset + len(text)
	}
	if written < len(line) {
		out.WriteString(line[written:])
	}
	return out.String()
}

func styleGoToken(tok token.Token, text string, theme SyntaxTheme) string {
	style := theme.Text
	switch {
	case tok.IsKeyword():
		style = theme.Keyword
	case tok == token.IDENT:
		style = styleGoIdent(text, theme)
	case tok == token.STRING || tok == token.CHAR:
		style = theme.String
	case tok == token.INT || tok == token.FLOAT || tok == token.IMAG:
		style = theme.Number
	case tok == token.COMMENT:
		style = theme.Comment
	case tok.IsOperator():
		style = theme.Operator
	}
	return syntaxStyle(style, text)
}

func styleGoIdent(ident string, theme SyntaxTheme) string {
	switch ident {
	case "any", "bool", "byte", "comparable", "complex64", "complex128",
		"error", "float32", "float64", "int", "int8", "int16", "int32",
		"int64", "rune", "string", "uint", "uint8", "uint16", "uint32",
		"uint64", "uintptr":
		return theme.Type
	case "append", "cap", "clear", "close", "complex", "copy", "delete",
		"imag", "len", "make", "max", "min", "new", "panic", "print",
		"println", "real", "recover":
		return theme.Function
	case "nil", "true", "false", "iota":
		return theme.Number
	default:
		if first, _ := utf8.DecodeRuneInString(ident); first >= 'A' && first <= 'Z' {
			return theme.Type
		}
		return theme.Text
	}
}

// HybridHighlighter uses the stdlib Go AST highlighter for Go code and
// Chroma for all other languages.
type HybridHighlighter struct {
	goHighlighter *DefaultHighlighter
	lang          string
	theme         SyntaxTheme
	buf           strings.Builder // accumulates lines for Chroma batch highlight
}

// NewHybridHighlighter creates a hybrid fenced-code highlighter.
func NewHybridHighlighter() *HybridHighlighter {
	return newHybridHighlighter(DefaultTheme().Syntax)
}

func newHybridHighlighter(theme SyntaxTheme) *HybridHighlighter {
	return &HybridHighlighter{goHighlighter: newDefaultHighlighter(theme), theme: theme}
}

func (h *HybridHighlighter) setSyntaxTheme(theme SyntaxTheme) {
	h.theme = theme
	h.goHighlighter.setSyntaxTheme(theme)
}

// Start begins a fenced-code block.
func (h *HybridHighlighter) Start(lang string, info string) {
	h.lang = strings.ToLower(strings.TrimSpace(lang))
	if h.lang == "" {
		h.lang = strings.ToLower(firstWord(info))
	}
	h.buf.Reset()
	if h.lang == "go" || h.lang == "golang" {
		h.goHighlighter.Start(lang, info)
	}
}

// HighlightLine highlights one code line using Chroma for non-Go languages.
func (h *HybridHighlighter) HighlightLine(line string) string {
	if h.lang == "go" || h.lang == "golang" {
		return h.goHighlighter.HighlightLine(line)
	}
	return chromaHighlightLine(line, h.lang, h.theme)
}

// End finishes a fenced-code block.
func (h *HybridHighlighter) End() {
	h.lang = ""
	h.buf.Reset()
	h.goHighlighter.End()
}

var _ CodeHighlighter = (*HybridHighlighter)(nil)

// chromaHighlightLine highlights a single line using Chroma.
// Falls back to themed text if the language is unknown or highlighting fails.
func chromaHighlightLine(line, lang string, theme SyntaxTheme) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return syntaxStyle(theme.Text, line)
	}
	return renderChromaTokens(iterator, theme)
}

func renderChromaTokens(iterator chroma.Iterator, theme SyntaxTheme) string {
	var out strings.Builder
	for token := iterator(); token != chroma.EOF; token = iterator() {
		out.WriteString(syntaxStyle(chromaTokenStyle(token.Type, theme), token.Value))
	}
	if out.Len() == 0 {
		return ""
	}
	return strings.TrimRight(out.String(), "\n")
}

func chromaTokenStyle(tokenType chroma.TokenType, theme SyntaxTheme) string {
	switch {
	case tokenType.InCategory(chroma.Comment):
		return theme.Comment
	case tokenType.InCategory(chroma.Keyword):
		if tokenType == chroma.KeywordType {
			return theme.Type
		}
		return theme.Keyword
	case tokenType.InSubCategory(chroma.LiteralString):
		return theme.String
	case tokenType.InSubCategory(chroma.LiteralNumber):
		return theme.Number
	case tokenType.InCategory(chroma.Operator):
		return theme.Operator
	case tokenType == chroma.NameFunction || tokenType.InSubCategory(chroma.NameFunction) || tokenType == chroma.NameBuiltin:
		return theme.Function
	case tokenType == chroma.NameClass || tokenType == chroma.NameException || tokenType == chroma.NameTag || tokenType == chroma.NameBuiltinPseudo:
		return theme.Type
	default:
		return theme.Text
	}
}

func syntaxStyle(style, text string) string {
	if text == "" || style == "" {
		return text
	}
	return style + text + reset
}

func firstWord(text string) string {
	for i := 0; i < len(text); i++ {
		if text[i] == ' ' || text[i] == '\t' {
			return text[:i]
		}
	}
	return text
}

// PlainHighlighter is a no-op CodeHighlighter that returns lines unchanged.
// Used when the renderer is in plain (non-TTY) mode.
type PlainHighlighter struct{}

func NewPlainHighlighter() *PlainHighlighter              { return &PlainHighlighter{} }
func (PlainHighlighter) Start(_ string, _ string)         {}
func (PlainHighlighter) HighlightLine(line string) string { return line }
func (PlainHighlighter) End()                             {}

var _ CodeHighlighter = (*PlainHighlighter)(nil)

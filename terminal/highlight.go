package terminal

import (
	"go/scanner"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
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
	lang string
}

// NewDefaultHighlighter creates the built-in fenced-code highlighter.
func NewDefaultHighlighter() *DefaultHighlighter {
	return &DefaultHighlighter{}
}

// Start begins a fenced-code block.
func (h *DefaultHighlighter) Start(lang string, _ string) {
	h.lang = lang
}

// HighlightLine highlights one code line.
func (h *DefaultHighlighter) HighlightLine(line string) string {
	switch h.lang {
	case "go", "golang":
		return highlightGoLine(line)
	default:
		return monokaiForeground + line + reset
	}
}

// End finishes a fenced-code block.
func (h *DefaultHighlighter) End() {
	h.lang = ""
}

func highlightGoLine(line string) string {
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
		out.WriteString(styleGoToken(tok, text))
		written = offset + len(text)
	}
	if written < len(line) {
		out.WriteString(line[written:])
	}
	return out.String()
}

func styleGoToken(tok token.Token, text string) string {
	style := monokaiForeground
	switch {
	case tok.IsKeyword():
		style = monokaiRed
	case tok == token.IDENT:
		style = styleGoIdent(text)
	case tok == token.STRING || tok == token.CHAR:
		style = monokaiYellow
	case tok == token.INT || tok == token.FLOAT || tok == token.IMAG:
		style = monokaiPurple
	case tok == token.COMMENT:
		style = monokaiComment
	case tok.IsOperator():
		style = monokaiRed
	}
	return style + text + reset
}

func styleGoIdent(ident string) string {
	switch ident {
	case "any", "bool", "byte", "comparable", "complex64", "complex128",
		"error", "float32", "float64", "int", "int8", "int16", "int32",
		"int64", "rune", "string", "uint", "uint8", "uint16", "uint32",
		"uint64", "uintptr":
		return monokaiBlue
	case "append", "cap", "clear", "close", "complex", "copy", "delete",
		"imag", "len", "make", "max", "min", "new", "panic", "print",
		"println", "real", "recover":
		return monokaiGreen
	case "nil", "true", "false", "iota":
		return monokaiPurple
	default:
		if first, _ := utf8.DecodeRuneInString(ident); first >= 'A' && first <= 'Z' {
			return monokaiBlue
		}
		return monokaiForeground
	}
}

// HybridHighlighter uses the stdlib Go AST highlighter for Go code and
// Chroma for all other languages.
type HybridHighlighter struct {
	goHighlighter *DefaultHighlighter
	lang          string
	buf           strings.Builder // accumulates lines for Chroma batch highlight
}

// NewHybridHighlighter creates a hybrid fenced-code highlighter.
func NewHybridHighlighter() *HybridHighlighter {
	return &HybridHighlighter{goHighlighter: NewDefaultHighlighter()}
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
	return chromaHighlightLine(line, h.lang)
}

// End finishes a fenced-code block.
func (h *HybridHighlighter) End() {
	h.lang = ""
	h.buf.Reset()
	h.goHighlighter.End()
}

var _ CodeHighlighter = (*HybridHighlighter)(nil)

// chromaHighlightLine highlights a single line using Chroma.
// Falls back to plain monochrome if the language is unknown or highlighting fails.
func chromaHighlightLine(line, lang string) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		return monokaiForeground + line + reset
	}
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return monokaiForeground + line + reset
	}
	var buf strings.Builder
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return monokaiForeground + line + reset
	}
	// Chroma terminal16m appends a newline; strip it so the renderer controls line endings.
	return strings.TrimRight(buf.String(), "\n")
}

func highlightGenericLine(line string, lang string) string {
	commentPrefix := genericCommentPrefix(lang)
	keywords := genericKeywords()
	var out strings.Builder
	for i := 0; i < len(line); {
		if commentPrefix != "" && strings.HasPrefix(line[i:], commentPrefix) {
			out.WriteString(monokaiComment)
			out.WriteString(line[i:])
			out.WriteString(reset)
			return out.String()
		}
		if isQuoteByte(line[i]) {
			end := scanQuotedSegment(line, i)
			out.WriteString(monokaiYellow)
			out.WriteString(line[i:end])
			out.WriteString(reset)
			i = end
			continue
		}
		if isDigit(line[i]) {
			end := i + 1
			for end < len(line) && (isDigit(line[end]) || line[end] == '_' || line[end] == '.' || line[end] == 'x' || line[end] == 'X' || line[end] == 'b' || line[end] == 'o' || isHexDigit(line[end])) {
				end++
			}
			out.WriteString(monokaiPurple)
			out.WriteString(line[i:end])
			out.WriteString(reset)
			i = end
			continue
		}
		if isIdentStart(line[i]) {
			end := i + 1
			for end < len(line) && isIdentPart(line[end]) {
				end++
			}
			word := line[i:end]
			if _, ok := keywords[word]; ok {
				out.WriteString(monokaiRed)
				out.WriteString(word)
				out.WriteString(reset)
			} else if first, _ := utf8.DecodeRuneInString(word); unicode.IsUpper(first) {
				out.WriteString(monokaiBlue)
				out.WriteString(word)
				out.WriteString(reset)
			} else {
				out.WriteString(monokaiForeground)
				out.WriteString(word)
				out.WriteString(reset)
			}
			i = end
			continue
		}
		out.WriteByte(line[i])
		i++
	}
	if out.Len() == 0 {
		return monokaiForeground + line + reset
	}
	return out.String()
}

func scanQuotedSegment(line string, start int) int {
	quote := line[start]
	for i := start + 1; i < len(line); i++ {
		if line[i] == '\\' {
			i++
			continue
		}
		if line[i] == quote {
			return i + 1
		}
	}
	return len(line)
}

func genericCommentPrefix(lang string) string {
	switch lang {
	case "python", "py", "bash", "sh", "zsh", "shell", "sql":
		return "#"
	default:
		return "//"
	}
}

func genericKeywords() map[string]struct{} {
	words := []string{
		"fn", "let", "mut", "pub", "impl", "struct", "enum", "trait", "match", "use",
		"mod", "crate", "self", "super", "const", "static", "ref", "return", "if",
		"else", "while", "for", "loop", "break", "continue", "async", "await",
		"function", "class", "import", "export", "default", "extends",
		"new", "this", "switch", "case", "try", "catch", "finally", "throw",
		"def", "lambda", "yield", "with", "pass", "raise", "from", "as",
		"true", "false", "null", "nil", "None", "undefined", "var", "do",
		"end", "then", "fi", "select", "insert", "update", "delete", "create",
	}
	out := make(map[string]struct{}, len(words))
	for _, word := range words {
		out[word] = struct{}{}
	}
	return out
}

func isQuoteByte(c byte) bool {
	return c == '"' || c == '\'' || c == '`'
}

func isIdentStart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c))
}

func isIdentPart(c byte) bool {
	return c == '_' || c == '-' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func firstWord(text string) string {
	for i := 0; i < len(text); i++ {
		if text[i] == ' ' || text[i] == '\t' {
			return text[:i]
		}
	}
	return text
}

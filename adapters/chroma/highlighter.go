package chroma

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/codewandler/markdown/terminal"
)

const (
	reset             = "\x1b[0m"
	monokaiForeground = "\x1b[38;2;248;248;242m"
	monokaiComment    = "\x1b[38;2;117;113;94m"
	monokaiRed        = "\x1b[38;2;249;38;114m"
	monokaiOrange     = "\x1b[38;2;253;151;31m"
	monokaiYellow     = "\x1b[38;2;230;219;116m"
	monokaiGreen      = "\x1b[38;2;166;226;46m"
	monokaiBlue       = "\x1b[38;2;102;217;239m"
	monokaiPurple     = "\x1b[38;2;174;129;255m"
)

// Highlighter adapts language highlighting to terminal.CodeHighlighter.
//
// The module keeps the core Markdown module free of optional language
// dependencies while still providing a stronger fallback than plain text.
type Highlighter struct {
	goHighlighter *terminal.DefaultHighlighter
	lang          string
}

// New creates the adapter highlighter.
func New() *Highlighter {
	return &Highlighter{goHighlighter: terminal.NewDefaultHighlighter()}
}

// Start selects the active language for a fenced code block.
func (h *Highlighter) Start(lang string, info string) {
	h.lang = strings.ToLower(strings.TrimSpace(lang))
	if h.lang == "" {
		h.lang = strings.ToLower(firstWord(info))
	}
	if h.lang == "go" || h.lang == "golang" {
		h.goHighlighter.Start(lang, info)
		return
	}
	h.goHighlighter.End()
}

// HighlightLine highlights one code line.
func (h *Highlighter) HighlightLine(line string) string {
	switch h.lang {
	case "go", "golang":
		return h.goHighlighter.HighlightLine(line)
	case "rust", "rs", "javascript", "js", "typescript", "ts", "python", "py", "bash", "sh", "zsh", "shell", "sql":
		return highlightGenericLine(line, h.lang)
	default:
		return monokaiForeground + line + reset
	}
}

// End finishes a fenced-code block.
func (h *Highlighter) End() {
	h.lang = ""
	h.goHighlighter.End()
}

var _ terminal.CodeHighlighter = (*Highlighter)(nil)

// HybridHighlighter keeps Go on the core fast path and uses the fallback
// highlighter for other languages.
type HybridHighlighter struct {
	goHighlighter *terminal.DefaultHighlighter
	fallback      *Highlighter
	useGo         bool
}

// NewHybrid creates a Go-fast-path, non-Go fallback highlighter.
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

func highlightGenericLine(line string, lang string) string {
	commentPrefix := genericCommentPrefix(lang)
	keywords := genericKeywords(lang)
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

func genericKeywords(lang string) map[string]struct{} {
	words := []string{
		"fn", "let", "mut", "pub", "impl", "struct", "enum", "trait", "match", "use",
		"mod", "crate", "self", "super", "const", "static", "ref", "return", "if",
		"else", "while", "for", "loop", "break", "continue", "async", "await",
		"function", "const", "class", "import", "export", "default", "extends",
		"new", "this", "switch", "case", "try", "catch", "finally", "throw",
		"def", "lambda", "yield", "with", "pass", "raise", "from", "as",
		"true", "false", "null", "nil", "None", "undefined", "let", "var", "do",
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

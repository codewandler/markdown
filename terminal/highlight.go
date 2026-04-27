package terminal

import (
	"go/scanner"
	"go/token"
	"strings"
	"unicode/utf8"
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

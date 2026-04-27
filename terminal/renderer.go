package terminal

import (
	"fmt"
	"io"
	"strings"

	"github.com/codewandler/markdown/stream"
)

const (
	reset     = "\x1b[0m"
	bold      = "\x1b[1m"
	italic    = "\x1b[3m"
	underline = "\x1b[4m"

	monokaiForeground = "\x1b[38;2;248;248;242m"
	monokaiComment    = "\x1b[38;2;117;113;94m"
	monokaiRed        = "\x1b[38;2;249;38;114m"
	monokaiOrange     = "\x1b[38;2;253;151;31m"
	monokaiYellow     = "\x1b[38;2;230;219;116m"
	monokaiGreen      = "\x1b[38;2;166;226;46m"
	monokaiBlue       = "\x1b[38;2;102;217;239m"
	monokaiPurple     = "\x1b[38;2;174;129;255m"
)

// Renderer writes terminal output for stream parser events.
type Renderer struct {
	w           io.Writer
	highlighter CodeHighlighter
	codeBlock   CodeBlockStyle
	inCode      bool
	codeLang    string
	lineStart   bool
	quoteDepth  int
	listDepth   int
	listItem    string
	spaced      bool
	pending     bool
}

// RendererOption configures a terminal renderer.
type RendererOption func(*Renderer)

// CodeBlockStyle controls terminal rendering for fenced-code blocks.
type CodeBlockStyle struct {
	Indent      int
	Border      bool
	BorderText  string
	BorderColor string
	Padding     int
}

// WithCodeBlockStyle configures fenced-code block layout.
func WithCodeBlockStyle(style CodeBlockStyle) RendererOption {
	return func(r *Renderer) {
		r.SetCodeBlockStyle(style)
	}
}

// DefaultCodeBlockStyle returns the default fenced-code block layout.
func DefaultCodeBlockStyle() CodeBlockStyle {
	return defaultCodeBlockStyle()
}

// NewRenderer creates a terminal renderer that writes to w.
func NewRenderer(w io.Writer, opts ...RendererOption) *Renderer {
	r := &Renderer{
		w:           w,
		highlighter: NewDefaultHighlighter(),
		codeBlock:   defaultCodeBlockStyle(),
		lineStart:   true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// SetCodeHighlighter sets the renderer's fenced-code highlighter.
//
// Passing nil restores the dependency-free default highlighter.
func (r *Renderer) SetCodeHighlighter(highlighter CodeHighlighter) {
	if highlighter == nil {
		highlighter = NewDefaultHighlighter()
	}
	r.highlighter = highlighter
}

// SetCodeBlockStyle changes fenced-code block layout.
func (r *Renderer) SetCodeBlockStyle(style CodeBlockStyle) {
	r.codeBlock = normalizeCodeBlockStyle(style)
}

// Render writes terminal output for events.
func (r *Renderer) Render(events []stream.Event) error {
	for _, event := range events {
		if err := r.render(event); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) render(event stream.Event) error {
	switch event.Kind {
	case stream.EventEnterBlock:
		return r.enterBlock(event)
	case stream.EventExitBlock:
		return r.exitBlock(event)
	case stream.EventText:
		return r.text(event)
	case stream.EventSoftBreak:
		_, err := fmt.Fprint(r.w, "\n")
		r.lineStart = true
		return err
	case stream.EventLineBreak:
		_, err := fmt.Fprint(r.w, "\n")
		r.lineStart = true
		return err
	default:
		return nil
	}
}

func (r *Renderer) enterBlock(event stream.Event) error {
	switch event.Block {
	case stream.BlockDocument:
		return nil
	case stream.BlockHeading, stream.BlockParagraph, stream.BlockFencedCode, stream.BlockThematicBreak, stream.BlockBlockquote, stream.BlockList:
		if r.pending && !r.spaced {
			if _, err := fmt.Fprint(r.w, "\n"); err != nil {
				return err
			}
		}
		r.spaced = false
	}
	if event.Block == stream.BlockBlockquote {
		r.quoteDepth++
		return nil
	}
	if event.Block == stream.BlockList {
		r.listDepth++
		return nil
	}
	if event.Block == stream.BlockListItem {
		r.listItem = listMarker(event)
		r.lineStart = true
		return nil
	}
	if event.Block == stream.BlockFencedCode {
		r.inCode = true
		r.codeLang = language(event.Info)
		r.highlighter.Start(r.codeLang, event.Info)
		return nil
	}
	if event.Block == stream.BlockHeading {
		_, err := fmt.Fprint(r.w, bold, monokaiGreen)
		return err
	}
	if event.Block == stream.BlockParagraph {
		_, err := fmt.Fprint(r.w, monokaiForeground)
		return err
	}
	if event.Block == stream.BlockThematicBreak {
		_, err := fmt.Fprint(r.w, monokaiComment, strings.Repeat("─", 24), reset, "\n")
		r.pending = true
		return err
	}
	return nil
}

func (r *Renderer) exitBlock(event stream.Event) error {
	switch event.Block {
	case stream.BlockHeading:
		_, err := fmt.Fprint(r.w, reset, "\n")
		r.pending = true
		return err
	case stream.BlockParagraph:
		_, err := fmt.Fprint(r.w, reset, "\n")
		r.lineStart = true
		r.pending = true
		return err
	case stream.BlockFencedCode:
		r.inCode = false
		r.codeLang = ""
		r.highlighter.End()
		r.pending = true
		return nil
	case stream.BlockBlockquote:
		if r.quoteDepth > 0 {
			r.quoteDepth--
		}
		r.pending = true
		return nil
	case stream.BlockList:
		if r.listDepth > 0 {
			r.listDepth--
		}
		r.pending = true
		return nil
	case stream.BlockListItem:
		r.listItem = ""
		r.lineStart = true
		return nil
	default:
		return nil
	}
}

func (r *Renderer) text(event stream.Event) error {
	if r.inCode {
		_, err := fmt.Fprint(r.w, r.codePrefix(), r.highlighter.HighlightLine(event.Text))
		r.lineStart = false
		return err
	}
	if err := r.writeLinePrefix(); err != nil {
		return err
	}
	_, err := fmt.Fprint(r.w, r.styleText(event))
	r.lineStart = false
	return err
}

func (r *Renderer) writeLinePrefix() error {
	if !r.lineStart {
		return nil
	}
	if r.quoteDepth > 0 {
		for range r.quoteDepth {
			if _, err := fmt.Fprint(r.w, monokaiComment, "│ ", reset); err != nil {
				return err
			}
		}
	}
	if r.listItem != "" {
		if _, err := fmt.Fprint(r.w, strings.Repeat("  ", max(0, r.listDepth-1)), monokaiComment, r.listItem, reset); err != nil {
			return err
		}
		r.listItem = strings.Repeat(" ", visibleListMarkerWidth(r.listItem))
	}
	r.lineStart = false
	return nil
}

func (r *Renderer) styleText(event stream.Event) string {
	text := event.Text
	if event.Style.Code {
		text = monokaiYellow + text + reset + monokaiForeground
	}
	if event.Style.Emphasis {
		text = italic + text + reset + monokaiForeground
	}
	if event.Style.Strong {
		text = bold + text + reset + monokaiForeground
	}
	if event.Style.Link != "" {
		text = underline + monokaiBlue + text + reset + monokaiComment + " (" + event.Style.Link + ")" + reset + monokaiForeground
	}
	return text
}

func (r *Renderer) codePrefix() string {
	style := normalizeCodeBlockStyle(r.codeBlock)
	var out strings.Builder
	if style.Indent > 0 {
		out.WriteString(strings.Repeat(" ", style.Indent))
	}
	if style.Border {
		if style.BorderColor != "" {
			out.WriteString(style.BorderColor)
		}
		out.WriteString(style.BorderText)
		if style.BorderColor != "" {
			out.WriteString(reset)
		}
	}
	if style.Padding > 0 {
		out.WriteString(strings.Repeat(" ", style.Padding))
	}
	return out.String()
}

func defaultCodeBlockStyle() CodeBlockStyle {
	return CodeBlockStyle{
		Indent:      4,
		Border:      true,
		BorderText:  "│",
		BorderColor: monokaiComment,
		Padding:     1,
	}
}

func normalizeCodeBlockStyle(style CodeBlockStyle) CodeBlockStyle {
	if style.Indent < 0 {
		style.Indent = 0
	}
	if style.Padding < 0 {
		style.Padding = 0
	}
	if style.Border && style.BorderText == "" {
		style.BorderText = "│"
	}
	return style
}

func language(info string) string {
	lang, _, _ := strings.Cut(strings.TrimSpace(info), " ")
	return strings.ToLower(lang)
}

func listMarker(event stream.Event) string {
	if event.List != nil && event.List.Ordered {
		start := event.List.Start
		if start <= 0 {
			start = 1
		}
		marker := event.List.Marker
		if marker == "" {
			marker = "."
		}
		return fmt.Sprintf("%d%s ", start, marker)
	}
	return "- "
}

func visibleListMarkerWidth(marker string) int {
	return len(marker)
}

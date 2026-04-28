package terminal

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/codewandler/markdown/stream"
)

const (
	reset     = "\x1b[0m"
	bold      = "\x1b[1m"
	italic    = "\x1b[3m"
	strike    = "\x1b[9m"
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
	inTable     bool
	table       tableBuffer
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

type tableBuffer struct {
	align       []stream.TableAlign
	rows        []tableRow
	currentRow  *tableRow
	currentCell *tableCell
}

type tableRow struct {
	cells []tableCell
}

type tableCell struct {
	events []stream.Event
}

// WithCodeBlockStyle configures fenced-code block layout.
func WithCodeBlockStyle(style CodeBlockStyle) RendererOption {
	return func(r *Renderer) {
		r.SetCodeBlockStyle(style)
	}
}

// WithCodeHighlighter configures fenced-code highlighting.
//
// Passing nil restores the dependency-free default highlighter.
func WithCodeHighlighter(highlighter CodeHighlighter) RendererOption {
	return func(r *Renderer) {
		if highlighter == nil {
			highlighter = NewDefaultHighlighter()
		}
		r.highlighter = highlighter
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
	if r.inTable {
		switch event.Kind {
		case stream.EventEnterBlock:
			switch event.Block {
			case stream.BlockTableRow:
				row := tableRow{}
				r.table.currentRow = &row
				return nil
			case stream.BlockTableCell:
				cell := tableCell{}
				r.table.currentCell = &cell
				return nil
			default:
				return nil
			}
		case stream.EventExitBlock:
			switch event.Block {
			case stream.BlockTableCell:
				if r.table.currentRow != nil && r.table.currentCell != nil {
					r.table.currentRow.cells = append(r.table.currentRow.cells, *r.table.currentCell)
				}
				r.table.currentCell = nil
				return nil
			case stream.BlockTableRow:
				if r.table.currentRow != nil {
					r.table.rows = append(r.table.rows, *r.table.currentRow)
				}
				r.table.currentRow = nil
				return nil
			case stream.BlockTable:
				return r.exitBlock(event)
			default:
				return nil
			}
		case stream.EventText, stream.EventSoftBreak, stream.EventLineBreak:
			if r.table.currentCell != nil {
				r.table.currentCell.events = append(r.table.currentCell.events, event)
			}
			return nil
		}
	}

	switch event.Kind {
	case stream.EventEnterBlock:
		return r.enterBlock(event)
	case stream.EventExitBlock:
		return r.exitBlock(event)
	case stream.EventText:
		return r.text(event)
	case stream.EventSoftBreak:
		_, err := fmt.Fprint(r.w, " ")
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
	case stream.BlockHeading, stream.BlockParagraph, stream.BlockFencedCode, stream.BlockThematicBreak, stream.BlockBlockquote, stream.BlockList, stream.BlockTable:
		if r.pending && !r.spaced {
			if _, err := fmt.Fprint(r.w, "\n"); err != nil {
				return err
			}
		}
		r.spaced = false
	}
	if event.Block == stream.BlockTable {
		r.inTable = true
		align := []stream.TableAlign(nil)
		if event.Table != nil {
			align = append([]stream.TableAlign(nil), event.Table.Align...)
		}
		r.table = tableBuffer{
			align: align,
		}
		return nil
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
	case stream.BlockTable:
		if err := r.flushTable(); err != nil {
			return err
		}
		r.inTable = false
		r.pending = true
		r.lineStart = true
		return nil
	case stream.BlockTableRow, stream.BlockTableCell:
		return nil
	default:
		return nil
	}
}

func (r *Renderer) text(event stream.Event) error {
	if r.inTable {
		if r.table.currentCell != nil {
			r.table.currentCell.events = append(r.table.currentCell.events, event)
		}
		return nil
	}
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

func (r *Renderer) flushTable() error {
	if len(r.table.rows) == 0 {
		r.table = tableBuffer{}
		return nil
	}
	cols := len(r.table.align)
	for _, row := range r.table.rows {
		if len(row.cells) > cols {
			cols = len(row.cells)
		}
	}
	widths := make([]int, cols)
	rendered := make([][]string, len(r.table.rows))
	for i, row := range r.table.rows {
		rendered[i] = make([]string, cols)
		for c := 0; c < cols; c++ {
			var cell tableCell
			if c < len(row.cells) {
				cell = row.cells[c]
			}
			text := r.renderTableCell(cell.events)
			rendered[i][c] = text
			if w := utf8.RuneCountInString(stripANSI(text)); w > widths[c] {
				widths[c] = w
			}
		}
	}
	for _, row := range rendered {
		if err := r.writeTableRow(row, widths); err != nil {
			return err
		}
	}
	r.table = tableBuffer{}
	return nil
}

func (r *Renderer) renderTableCell(events []stream.Event) string {
	var out strings.Builder
	for _, ev := range events {
		switch ev.Kind {
		case stream.EventText:
			out.WriteString(r.styleText(ev))
		case stream.EventSoftBreak, stream.EventLineBreak:
			out.WriteByte(' ')
		}
	}
	return out.String()
}

func (r *Renderer) writeTableRow(cells []string, widths []int) error {
	if err := r.writeLinePrefix(); err != nil {
		return err
	}
	var out strings.Builder
	out.WriteString(monokaiComment)
	out.WriteString("│")
	out.WriteString(reset)
	for i := 0; i < len(widths); i++ {
		out.WriteByte(' ')
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		width := utf8.RuneCountInString(stripANSI(cell))
		align := stream.TableAlignNone
		if i < len(r.table.align) {
			align = r.table.align[i]
		}
		out.WriteString(padTableCell(cell, widths[i], width, align))
		out.WriteByte(' ')
		out.WriteString(monokaiComment)
		out.WriteString("│")
		out.WriteString(reset)
	}
	out.WriteByte('\n')
	if _, err := io.WriteString(r.w, out.String()); err != nil {
		return err
	}
	r.lineStart = true
	return nil
}

func padTableCell(cell string, targetWidth, visibleWidth int, align stream.TableAlign) string {
	if targetWidth <= visibleWidth {
		return cell
	}
	pad := targetWidth - visibleWidth
	switch align {
	case stream.TableAlignRight:
		return strings.Repeat(" ", pad) + cell
	case stream.TableAlignCenter:
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + cell + strings.Repeat(" ", right)
	default:
		return cell + strings.Repeat(" ", pad)
	}
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
	if event.Style.Strike {
		text = strike + text + reset + monokaiForeground
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
		prefix := fmt.Sprintf("%d%s ", start, marker)
		if event.List.Task {
			prefix += taskCheckbox(event.List.Checked)
		}
		return prefix
	}
	prefix := "- "
	if event.List != nil && event.List.Task {
		prefix += taskCheckbox(event.List.Checked)
	}
	return prefix
}

func visibleListMarkerWidth(marker string) int {
	return len(marker)
}

func taskCheckbox(checked bool) string {
	if checked {
		return "[x] "
	}
	return "[ ] "
}

func stripANSI(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

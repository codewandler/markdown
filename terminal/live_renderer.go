package terminal

import (
	"fmt"
	"io"
	"strings"

	"github.com/codewandler/markdown/stream"
)

// LiveRenderer wires a parser to a terminal renderer and redraws tables as rows
// arrive. It is intended for interactive TTY output; unlike Renderer, it emits
// cursor-control escape sequences and is not suitable for logs or pipes.
type LiveRenderer struct {
	parser    stream.Parser
	render    *Renderer
	flushed   bool
	liveLines int
}

// NewLiveRenderer creates a streaming renderer that redraws table output when
// table widths grow. Non-table content is rendered append-only like
// NewStreamRenderer.
func NewLiveRenderer(w io.Writer, opts ...RendererOption) *LiveRenderer {
	r := NewRenderer(w, opts...)
	// Live tables compute widths from rows seen so far. Keep the underlying table
	// renderer in buffered mode even if fixed-width options were supplied.
	r.tableLayout = TableLayout{Mode: TableModeBuffered}
	return &LiveRenderer{
		parser: stream.NewParser(r.parserOptions...),
		render: r,
	}
}

// Write parses Markdown bytes and renders newly emitted events, redrawing the
// current table region after each completed table row.
func (r *LiveRenderer) Write(p []byte) (int, error) {
	if r.flushed {
		return 0, io.ErrClosedPipe
	}
	events, err := r.parser.Write(p)
	if err != nil {
		return len(p), err
	}
	if err := r.Render(events); err != nil {
		return len(p), err
	}
	return len(p), nil
}

// Flush finalizes parsing and renders any remaining events.
func (r *LiveRenderer) Flush() error {
	if r.flushed {
		return nil
	}
	events, err := r.parser.Flush()
	if err != nil {
		return err
	}
	if err := r.Render(events); err != nil {
		return err
	}
	r.flushed = true
	return nil
}

// Render writes terminal output for events.
func (r *LiveRenderer) Render(events []stream.Event) error {
	for _, event := range events {
		if err := r.renderEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (r *LiveRenderer) renderEvent(event stream.Event) error {
	if r.render.inTable && event.Kind == stream.EventExitBlock && event.Block == stream.BlockTable {
		// The table is already visible from the last row redraw. Commit the live
		// region without calling Renderer.exitBlock, which would flush and print a
		// duplicate buffered table.
		r.render.inTable = false
		r.render.table = tableBuffer{}
		r.render.pending = true
		r.render.lineStart = true
		r.liveLines = 0
		return nil
	}

	if err := r.render.render(event); err != nil {
		return err
	}

	if event.Kind == stream.EventEnterBlock && event.Block == stream.BlockTable {
		r.liveLines = 0
		return nil
	}
	if r.render.inTable && event.Kind == stream.EventExitBlock && event.Block == stream.BlockTableRow {
		return r.redrawTable()
	}
	return nil
}

func (r *LiveRenderer) redrawTable() error {
	if len(r.render.table.rows) == 0 {
		return nil
	}
	var out strings.Builder
	if r.liveLines > 0 {
		fmt.Fprintf(&out, "\x1b[%dA", r.liveLines)
		out.WriteString("\x1b[J")
	}
	var tableOut strings.Builder
	tmp := *r.render
	tmp.w = &tableOut
	tmp.table = tableBuffer{
		align: append([]stream.TableAlign(nil), r.render.table.align...),
		rows:  append([]tableRow(nil), r.render.table.rows...),
	}
	tmp.lineStart = true
	if err := tmp.flushTable(); err != nil {
		return err
	}
	rendered := tableOut.String()
	out.WriteString(rendered)
	if _, err := io.WriteString(r.render.w, out.String()); err != nil {
		return err
	}
	r.liveLines = countRenderedLines(rendered)
	r.render.lineStart = true
	r.render.lineWidth = 0
	return nil
}

func countRenderedLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		lines++
	}
	return lines
}

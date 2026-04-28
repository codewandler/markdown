// Package markdown provides markdown parsing and rendering primitives for streaming AI-agent output.
//
// This package exports the stream-based parser and terminal renderer APIs.
// For block-level buffering with goldmark, see the Buffer type in buffer.go.
//
// The stream API is designed around an event model:
//   - Parser consumes markdown chunks and emits events
//   - Renderer consumes events and writes terminal output
//   - Convenience functions handle simple cases
//
// Simple usage:
//
//	rendered, err := markdown.RenderString("# Hello\n\nWorld")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(rendered)
//
// Streaming usage:
//
//	parser := markdown.NewParser()
//	renderer := markdown.NewRenderer(os.Stdout)
//
//	for _, chunk := range chunks {
//		events, err := parser.Write([]byte(chunk))
//		if err != nil {
//			log.Fatal(err)
//		}
//		if err := renderer.Render(events); err != nil {
//			log.Fatal(err)
//		}
//	}
//
//	events, err := parser.Flush()
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := renderer.Render(events); err != nil {
//		log.Fatal(err)
//	}
package markdown

import (
	"bytes"
	"io"

	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

// Re-export types from stream and terminal subpackages for convenience.
// Note: BlockKind is defined in buffer.go for the goldmark-based Buffer API.
// The stream.BlockKind constants below are for the stream-based Parser API.

type (
	// Parser consumes markdown chunks and emits events.
	// Designed for streaming/incremental parsing.
	Parser = stream.Parser

	// TerminalRenderer consumes events and writes terminal output.
	// Designed for streaming/incremental rendering.
	TerminalRenderer = terminal.Renderer

	// Event represents a markdown syntax event.
	Event = stream.Event

	// EventKind describes the type of event.
	EventKind = stream.EventKind

	// StreamBlockKind describes a markdown block type in the stream API.
	// (Note: buffer.go defines a separate BlockKind for the Buffer API)
	StreamBlockKind = stream.BlockKind

	// TableAlign describes column alignment in the stream API.
	StreamTableAlign = stream.TableAlign

	// InlineStyle describes inline formatting.
	InlineStyle = stream.InlineStyle

	// ListData describes list properties.
	ListData = stream.ListData

	// TableData describes table properties.
	TableData = stream.TableData

	// TableAlign describes column alignment.
	TableAlign = stream.TableAlign

	// CodeBlockStyle controls terminal rendering for fenced-code blocks.
	CodeBlockStyle = terminal.CodeBlockStyle

	// CodeHighlighter highlights code blocks.
	CodeHighlighter = terminal.CodeHighlighter

	// ParserOption configures parser behavior.
	ParserOption = stream.ParserOption

	// RendererOption configures renderer behavior.
	RendererOption = terminal.RendererOption
)

// Re-export constants from stream and terminal subpackages.
// Note: buffer.go defines BlockKind constants for the Buffer API.
// These are StreamBlockKind constants for the stream-based Parser API.

const (
	// Event kinds
	EventEnterBlock = stream.EventEnterBlock
	EventExitBlock  = stream.EventExitBlock
	EventText       = stream.EventText

	// Stream block kinds (for stream.Parser API)
	StreamBlockDocument      = stream.BlockDocument
	StreamBlockParagraph     = stream.BlockParagraph
	StreamBlockHeading       = stream.BlockHeading
	StreamBlockList          = stream.BlockList
	StreamBlockListItem      = stream.BlockListItem
	StreamBlockTable         = stream.BlockTable
	StreamBlockTableRow      = stream.BlockTableRow
	StreamBlockTableCell     = stream.BlockTableCell
	StreamBlockBlockquote    = stream.BlockBlockquote
	StreamBlockFencedCode    = stream.BlockFencedCode
	StreamBlockIndentedCode  = stream.BlockIndentedCode
	StreamBlockThematicBreak = stream.BlockThematicBreak
	StreamBlockHTML          = stream.BlockHTML

	// Table alignment
	TableAlignNone   = stream.TableAlignNone
	TableAlignLeft   = stream.TableAlignLeft
	TableAlignCenter = stream.TableAlignCenter
	TableAlignRight  = stream.TableAlignRight
)

// RenderString renders markdown to terminal output string.
// Handles parsing and rendering internally.
// Returns rendered string or error.
func RenderString(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := RenderToWriter(&buf, markdown); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderToWriter renders markdown to the given writer.
// Handles parsing and rendering internally.
func RenderToWriter(w io.Writer, markdown string) error {
	parser := stream.NewParser()
	renderer := terminal.NewRenderer(w)

	events, err := parser.Write([]byte(markdown))
	if err != nil {
		return err
	}
	if err := renderer.Render(events); err != nil {
		return err
	}

	events, err = parser.Flush()
	if err != nil {
		return err
	}
	return renderer.Render(events)
}

// NewParser creates a streaming parser.
// The parser consumes markdown chunks via Write() and emits events.
// Call Flush() after the last Write() to finalize buffered content.
func NewParser(opts ...ParserOption) Parser {
	return stream.NewParser(opts...)
}

// NewRenderer creates a terminal renderer.
// The renderer consumes events via Render() and writes ANSI-styled output.
// Can be called multiple times as events arrive.
func NewRenderer(w io.Writer, opts ...RendererOption) *TerminalRenderer {
	return terminal.NewRenderer(w, opts...)
}

// DefaultCodeBlockStyle returns the default fenced-code block layout.
func DefaultCodeBlockStyle() CodeBlockStyle {
	return terminal.DefaultCodeBlockStyle()
}

// NewDefaultHighlighter creates the default syntax highlighter.
func NewDefaultHighlighter() CodeHighlighter {
	return terminal.NewDefaultHighlighter()
}

// WithCodeBlockStyle configures fenced-code block layout.
func WithCodeBlockStyle(style CodeBlockStyle) RendererOption {
	return terminal.WithCodeBlockStyle(style)
}

// SetCodeHighlighter sets custom syntax highlighter on an existing renderer.
// This is a method on *TerminalRenderer, not an option function.
// Use renderer.SetCodeHighlighter(h) after creating the renderer.

package stream

// EventKind identifies the kind of parser event.
type EventKind string

const (
	EventEnterBlock EventKind = "enter_block"
	EventExitBlock  EventKind = "exit_block"
	EventText       EventKind = "text"
	EventSoftBreak  EventKind = "soft_break"
	EventLineBreak  EventKind = "line_break"
)

// Position identifies a byte position in the streamed Markdown source.
//
// Offset is zero-based. Line and Column are one-based byte coordinates.
type Position struct {
	Offset int64
	Line   int
	Column int
}

// Span identifies the source range that produced an event.
type Span struct {
	Start Position
	End   Position
}

// BlockKind describes a Markdown block represented in the event stream.
type BlockKind string

const (
	BlockDocument      BlockKind = "document"
	BlockParagraph     BlockKind = "paragraph"
	BlockHeading       BlockKind = "heading"
	BlockList          BlockKind = "list"
	BlockListItem      BlockKind = "list_item"
	BlockBlockquote    BlockKind = "blockquote"
	BlockFencedCode    BlockKind = "fenced_code"
	BlockIndentedCode  BlockKind = "indented_code"
	BlockThematicBreak BlockKind = "thematic_break"
	BlockHTML          BlockKind = "html"
)

// InlineStyle describes inline presentation discovered by the parser.
//
// Renderers may ignore fields they do not support.
type InlineStyle struct {
	Emphasis bool
	Strong   bool
	Code     bool
	Link     string
}

// ListData describes a Markdown list represented by a list block event.
type ListData struct {
	Ordered bool
	Start   int
	Marker  string
	Tight   bool
}

// Event is one append-only parser output item.
//
// Block is set for block boundary events. Text and Style are set for text
// events. Level is used by hierarchical blocks such as headings. Span identifies
// the source range that produced the event when a meaningful range exists.
type Event struct {
	Kind  EventKind
	Block BlockKind
	Text  string
	Style InlineStyle
	Level int
	Info  string
	Span  Span
	List  *ListData
}

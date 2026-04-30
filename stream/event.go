package stream

// EventKind identifies the kind of parser event.
type EventKind uint8

const (
	EventEnterBlock EventKind = iota + 1
	EventExitBlock
	EventText
	EventSoftBreak
	EventLineBreak
)

func (k EventKind) String() string {
	switch k {
	case EventEnterBlock:
		return "enter_block"
	case EventExitBlock:
		return "exit_block"
	case EventText:
		return "text"
	case EventSoftBreak:
		return "soft_break"
	case EventLineBreak:
		return "line_break"
	default:
		return "unknown"
	}
}

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
type BlockKind uint8

const (
	BlockDocument      BlockKind = iota + 1
	BlockParagraph
	BlockHeading
	BlockList
	BlockListItem
	BlockTable
	BlockTableRow
	BlockTableCell
	BlockBlockquote
	BlockFencedCode
	BlockIndentedCode
	BlockThematicBreak
	BlockHTML
)

func (k BlockKind) String() string {
	switch k {
	case BlockDocument:
		return "document"
	case BlockParagraph:
		return "paragraph"
	case BlockHeading:
		return "heading"
	case BlockList:
		return "list"
	case BlockListItem:
		return "list_item"
	case BlockTable:
		return "table"
	case BlockTableRow:
		return "table_row"
	case BlockTableCell:
		return "table_cell"
	case BlockBlockquote:
		return "blockquote"
	case BlockFencedCode:
		return "fenced_code"
	case BlockIndentedCode:
		return "indented_code"
	case BlockThematicBreak:
		return "thematic_break"
	case BlockHTML:
		return "html"
	default:
		return "unknown"
	}
}

// InlineStyle describes inline presentation discovered by the parser.
//
// Renderers may ignore fields they do not support.
type InlineStyle struct {
	Emphasis      bool
	Strong        bool
	Strike        bool
	Code          bool
	RawHTML       bool   // true for inline raw HTML tags
	Image         bool   // true for ![alt](url) and ![alt][ref]
	EmphasisDepth int16  // nesting depth for emphasis (0 = not emphasized)
	StrongDepth   int16  // nesting depth for strong (0 = not strong)
	LinkData      *LinkData // non-nil for link/image events
}

// LinkData holds link-related strings, allocated only for link/image events.
type LinkData struct {
	Link           string
	LinkTitle      string
	HasLink        bool
	ImageLink      string // wrapping link href when image is inside a link
	ImageLinkTitle string // wrapping link title
}

// GetLink returns the link URL, or "" if no link data.
func (s InlineStyle) GetLink() string {
	if s.LinkData == nil { return "" }
	return s.LinkData.Link
}

// GetLinkTitle returns the link title, or "" if no link data.
func (s InlineStyle) GetLinkTitle() string {
	if s.LinkData == nil { return "" }
	return s.LinkData.LinkTitle
}

// GetHasLink reports whether this style carries link information.
func (s InlineStyle) GetHasLink() bool {
	return s.LinkData != nil && s.LinkData.HasLink
}

// GetImageLink returns the wrapping link href, or "" if none.
func (s InlineStyle) GetImageLink() string {
	if s.LinkData == nil { return "" }
	return s.LinkData.ImageLink
}

// GetImageLinkTitle returns the wrapping link title, or "" if none.
func (s InlineStyle) GetImageLinkTitle() string {
	if s.LinkData == nil { return "" }
	return s.LinkData.ImageLinkTitle
}

// WithLink returns a new InlineStyle with link data set.
func WithLink(link, title string) InlineStyle {
	return InlineStyle{LinkData: &LinkData{Link: link, LinkTitle: title, HasLink: true}}
}

// ListData describes a Markdown list represented by a list block event.
type ListData struct {
	Ordered bool
	Start   int
	Marker  string
	Tight   bool
	Task    bool
	Checked bool
}

// TableData describes a Markdown table represented by table block events.
type TableData struct {
	Align []TableAlign
}

// TableRowData describes a table row in the event stream.
type TableRowData struct {
	Header bool // true for the header row (first row before delimiter)
}

// TableAlign describes the alignment of a table column.
type TableAlign int

const (
	TableAlignNone TableAlign = iota
	TableAlignLeft
	TableAlignCenter
	TableAlignRight
)

// Event is one append-only parser output item.
//
// Block is set for block boundary events. Text and Style are set for text
// events. Level is used by hierarchical blocks such as headings. Span identifies
// the source range that produced the event when a meaningful range exists.
type Event struct {
	Kind     EventKind
	Block    BlockKind
	Text     string
	Style    InlineStyle
	Level    int
	Info     string
	Span     Span
	List     *ListData
	Table    *TableData
	TableRow *TableRowData
}

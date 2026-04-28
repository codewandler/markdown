package stream

import (
	"bytes"
	"html"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const releaseBufferCap = 64 * 1024

// NewParser creates an incremental Markdown parser.
func NewParser(opts ...ParserOption) Parser {
	cfg := defaultParserConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &parser{config: cfg, line: 1, column: 1}
}

type parser struct {
	config ParserConfig

	started bool
	flushed bool

	offset int64
	line   int
	column int

	partial []byte

	paragraph paragraphState
	fence     fenceState
	refs      map[string]linkReference
	refDef    linkReferenceDefinitionState
	table     tableState

	// pendingBlocks buffers closed paragraph/heading events so that
	// link reference definitions appearing after the paragraph can be
	// collected before inline parsing runs. The pending blocks are
	// drained (inline-parsed and emitted) when a non-definition block
	// starts or at Flush.
	pendingBlocks []pendingBlock

	inBlockquote       bool
	bqInsideListItem   bool // true if blockquote was opened inside a list item
	inList             bool
	listData           ListData
	listLoose          bool
	inListItem         bool
	listItemIndent     int  // content column: marker indent + marker width + padding
	listItemBlankLine  bool // saw a blank line inside the current list item
	listStack          []savedList // stack for nested lists
	inIndented         bool
	indentedBlankLines int
	inHTMLBlock        bool
	htmlBlockType      int    // 1-7 per CommonMark spec
	htmlBlockEnd       string // end condition string for types 1-5
}

// savedList stores the state of an outer list when entering a sublist.
type savedList struct {
	inList            bool
	listData          ListData
	listLoose         bool
	inListItem        bool
	listItemIndent    int
	listItemBlankLine bool
}

// pendingBlock stores a closed paragraph whose inline content has not yet
// been parsed. This allows forward link reference definitions to be
// collected before inline parsing resolves references.
type pendingBlock struct {
	text string
	span Span
}

type linkReference struct {
	dest  string
	title string
}

type linkReferenceDefinitionState struct {
	active  bool
	label   string
	dest    string
	hasDest bool
	title   string
	lines   []lineInfo
}

type lineInfo struct {
	text       string
	start      Position
	end        Position
	nextOffset int64
}

type paragraphState struct {
	lines []paragraphLine
}

type paragraphLine struct {
	text string
	span Span
}

type fenceState struct {
	open   bool
	marker byte
	length int
	indent int
	info   string
}

type tableState struct {
	active bool
	align  []TableAlign
	span   Span
}

func (p *parser) Write(chunk []byte) ([]Event, error) {
	if len(chunk) == 0 || p.flushed {
		return nil, nil
	}
	p.partial = append(p.partial, chunk...)

	var events []Event
	for {
		i := bytes.IndexByte(p.partial, '\n')
		if i < 0 {
			break
		}
		raw := p.partial[:i]
		if len(raw) > 0 && raw[len(raw)-1] == '\r' {
			raw = raw[:len(raw)-1]
		}
		info := p.nextLineInfo(string(raw), true)
		p.processLine(info, &events)
		p.partial = p.partial[i+1:]
		if cap(p.partial) > releaseBufferCap && len(p.partial) == 0 {
			p.partial = nil
		}
	}
	return events, nil
}

func (p *parser) Flush() ([]Event, error) {
	if p.flushed {
		return nil, nil
	}
	var events []Event
	if len(p.partial) > 0 {
		raw := p.partial
		if len(raw) > 0 && raw[len(raw)-1] == '\r' {
			raw = raw[:len(raw)-1]
		}
		info := p.nextLineInfo(string(raw), false)
		p.processLine(info, &events)
		p.partial = nil
	}
	p.closePendingLinkReferenceDefinition(&events)
	p.closeTable(&events)
	p.closeParagraph(&events)
	p.closeIndentedCode(&events)
	if p.fence.open {
		p.fence.open = false
		events = append(events, Event{Kind: EventExitBlock, Block: BlockFencedCode})
	}
	p.closeHTMLBlock(&events)
	p.drainPendingBlocks(&events)
	p.closeContainers(&events)
	if p.started {
		events = append(events, Event{Kind: EventExitBlock, Block: BlockDocument})
	}
	p.flushed = true
	return events, nil
}

func (p *parser) Reset() {
	*p = parser{config: p.config, line: 1, column: 1}
}

func (p *parser) nextLineInfo(text string, hadNewline bool) lineInfo {
	start := Position{Offset: p.offset, Line: p.line, Column: p.column}
	end := Position{Offset: p.offset + int64(len(text)), Line: p.line, Column: p.column + len(text)}
	nextOffset := end.Offset
	if hadNewline {
		nextOffset++
	}
	p.offset = nextOffset
	p.line++
	p.column = 1
	return lineInfo{text: text, start: start, end: end, nextOffset: nextOffset}
}

func (p *parser) processLine(line lineInfo, events *[]Event) {
	if p.table.active {
		if strings.TrimSpace(line.text) == "" {
			p.closeTable(events)
		} else if p.processActiveTableLine(line, events) {
			return
		} else {
			p.closeTable(events)
			p.processLine(line, events)
			return
		}
	}

	// HTML block continuation.
	if p.inHTMLBlock {
		p.processHTMLBlockLine(line, events)
		return
	}

	// When inside a list item with an open fenced/indented code block,
	// route through list item continuation so that indentation is
	// interpreted relative to the item's content column.
	if p.inListItem && (p.fence.open || p.inIndented) {
		indent, _ := leadingIndent(line.text)
		if indent >= p.listItemIndent || strings.TrimSpace(line.text) == "" {
			inner := line
			if strings.TrimSpace(line.text) != "" {
				inner.text = stripIndent(line.text, p.listItemIndent)
			}
			p.processListItemContent(inner, events)
			return
		}
		// Not enough indent — close the code block and fall through.
		p.closeIndentedCode(events)
		p.closeFencedCode(events)
	}

	if p.fence.open {
		// Inside a blockquote, non-> lines close the blockquote
		// (and the fence inside it).
		if p.inBlockquote {
			if _, ok := blockquoteContent(line.text); !ok {
				p.closeFencedCode(events)
				p.closeBlockquote(line, events)
				// Fall through to process the line normally.
			} else {
				p.processFenceLine(line, events)
				return
			}
		} else {
			p.processFenceLine(line, events)
			return
		}
	}

	if p.inIndented {
		// Inside a blockquote, non-> lines close the blockquote.
		if p.inBlockquote {
			if _, ok := blockquoteContent(line.text); !ok {
				p.closeIndentedCode(events)
				p.closeBlockquote(line, events)
				// Fall through to process normally.
			} else {
				if indentedCode(line.text) {
					p.emitIndentedCodeLine(line, events)
					return
				}
				p.closeIndentedCode(events)
			}
		} else {
			if indentedCode(line.text) {
				p.emitIndentedCodeLine(line, events)
				return
			}
			if strings.TrimSpace(line.text) == "" {
				p.indentedBlankLines++
				return
			}
			p.closeIndentedCode(events)
		}
	}

	if p.refDef.active {
		if p.continueLinkReferenceDefinition(line, events) {
			return
		}
	}

	if strings.TrimSpace(line.text) == "" {
		p.closeParagraph(events)
		p.closeIndentedCode(events)
		if p.inListItem {
			// Don't close the list item yet — the next non-blank line
			// may be a continuation if it's indented enough.
			p.listItemBlankLine = true
		} else {
			p.closeListItem(events)
			p.closeContainers(events)
		}
		// Drain pending blocks that don't contain bracket syntax,
		// since they can't benefit from deferred reference resolution.
		// Blocks with brackets are kept pending so that link reference
		// definitions following the blank line can be collected first.
		p.drainPendingBlocksEager(events)
		return
	}

	// Non-blank line: check for list item continuation after a blank line.
	if p.inListItem && p.listItemBlankLine {
		p.listItemBlankLine = false
		indent, _ := leadingIndent(line.text)
		if indent >= p.listItemIndent {
			// Continuation after blank line — the list becomes loose.
			p.listLoose = true
			inner := line
			inner.text = stripIndent(line.text, p.listItemIndent)
			p.processListItemContent(inner, events)
			return
		}
		// Check if this is a new list item — blank line between items
		// makes the list loose.
		if _, ok := listItem(line.text); ok {
			p.listLoose = true
		}
		// Not enough indent — close the list item.
		p.closeListItem(events)
		if _, ok := listItem(line.text); !ok {
			p.closeList(events)
		}
	}

	if len(p.paragraph.lines) == 0 && !p.inList && !p.inListItem {
		if p.startLinkReferenceDefinition(line) {
			return
		}
	}

	// If we reach here, the line is non-blank and not a ref def continuation.
	// Drain any pending blocks now — all definitions before this point have
	// been collected.
	p.drainPendingBlocks(events)

	// When inside a blockquote and the line doesn't continue it,
	// close the blockquote before checking list item continuation.
	// Lists opened inside the blockquote are closed with it, so
	// they must not be treated as continuation targets.
	if p.inBlockquote && !p.fence.open && !p.inIndented {
		if _, ok := blockquoteContent(line.text); !ok {
			// Lazy continuation: a paragraph open inside the blockquote
			// can continue without the > prefix.
			if len(p.paragraph.lines) > 0 && !thematicBreak(line.text) {
				if _, _, _, _, isFence := openingFence(line.text); !isFence {
					if _, ok2 := listItem(line.text); !ok2 {
						p.addParagraphLine(line)
						return
					}
				}
			}
			p.closeBlockquote(line, events)
		}
	}

	// When inside a list item, check if the line is indented enough
	// to be continuation content. This must come before blockquote
	// and other container checks so that indented `>` markers inside
	// list items are treated as list item content, not top-level
	// blockquotes.
	if p.inListItem && !p.fence.open && !p.inIndented {
		indent, _ := leadingIndent(line.text)
		// Unwind sublists that don't match.
		for p.inListItem && indent < p.listItemIndent && len(p.listStack) > 0 {
			p.closeParagraph(events)
			p.drainPendingBlocks(events)
			p.closeIndentedCode(events)
			p.closeFencedCode(events)
			p.inListItem = false
			*events = append(*events, Event{Kind: EventExitBlock, Block: BlockListItem})
			// Check for sibling in the sublist.
			if item, ok := listItem(line.text); ok {
				stripped := stripIndent(line.text, p.listStack[len(p.listStack)-1].listItemIndent)
				if sItem, ok2 := listItem(stripped); ok2 && p.listData.Ordered == sItem.data.Ordered && p.listData.Marker == sItem.data.Marker {
					_ = item
					p.inListItem = true
					p.listItemIndent = sItem.contentIndent
					p.listItemBlankLine = false
					data := sItem.data
					*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockListItem, List: &data, Span: Span{Start: line.start, End: line.end}})
					if strings.TrimSpace(sItem.content) != "" {
						inner := line
						inner.text = sItem.content
						p.processListItemFirstLine(inner, events)
					}
					return
				}
			}
			// Not a sibling — close the sublist.
			p.inList = false
			data := p.listData
			data.Tight = !p.listLoose
			*events = append(*events, Event{Kind: EventExitBlock, Block: BlockList, List: &data})
			p.listData = ListData{}
			p.listLoose = false
			p.popList()
		}
		if p.inListItem && indent >= p.listItemIndent {
			inner := line
			inner.text = stripIndent(line.text, p.listItemIndent)
			p.processListItemContent(inner, events)
			return
		}
		// Lazy continuation: if a paragraph is open inside the list
		// item, a non-blank line continues it even without enough indent.
		if p.inListItem && len(p.paragraph.lines) > 0 && !thematicBreak(line.text) {
			if _, ok := listItem(line.text); !ok {
				p.addParagraphLine(line)
				return
			}
		}
	}

	if content, ok := blockquoteContent(line.text); ok {
		p.ensureDocument(events)
		if !p.inBlockquote {
			p.closeParagraph(events)
			p.closeList(events)
			p.inBlockquote = true; p.bqInsideListItem = p.inListItem
			p.emitBlockStart(events, Event{Kind: EventEnterBlock, Block: BlockBlockquote, Span: Span{Start: line.start, End: line.end}})
		}
		if strings.TrimSpace(content) == "" {
			p.closeParagraph(events)
			p.closeIndentedCode(events)
			return
		}
		inner := line
		inner.text = content
		// Route blockquote content through full block detection
		// so that list items, fenced code, headings, etc. work
		// inside blockquotes.
		if p.fence.open {
			if p.isClosingFence(inner.text) {
				p.fence.open = false
				*events = append(*events, Event{Kind: EventExitBlock, Block: BlockFencedCode, Span: Span{Start: inner.start, End: inner.end}})
				return
			}
			*events = append(*events,
				Event{Kind: EventText, Text: stripFenceIndent(inner.text, p.fence.indent), Span: Span{Start: inner.start, End: inner.end}},
				Event{Kind: EventLineBreak, Span: Span{Start: inner.end, End: inner.end}},
			)
			return
		}
		if p.inIndented {
			if indentedCode(inner.text) {
				p.emitIndentedCodeLine(inner, events)
				return
			}
			p.closeIndentedCode(events)
		}
		if item, ok := listItem(inner.text); ok {
			p.closeParagraph(events)
			if !p.inList || p.listData.Ordered != item.data.Ordered || p.listData.Marker != item.data.Marker {
				p.closeList(events)
				p.inList = true
				p.listData = listBlockData(item.data)
				p.listLoose = false
				data := p.listData
				*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockList, List: &data, Span: Span{Start: inner.start, End: inner.end}})
			}
			p.closeListItem(events)
			p.inListItem = true
			p.listItemIndent = item.contentIndent
			p.listItemBlankLine = false
			data := item.data
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockListItem, List: &data, Span: Span{Start: inner.start, End: inner.end}})
			if strings.TrimSpace(item.content) != "" {
				ci := inner
				ci.text = item.content
				p.processListItemFirstLine(ci, events)
			}
			return
		}
		p.processNonContainerLine(inner, events)
		return
	}

	if p.inBlockquote {
		if len(p.paragraph.lines) > 0 && !thematicBreak(line.text) {
			if _, _, _, _, isFence := openingFence(line.text); !isFence {
				if _, ok := listItem(line.text); !ok {
					p.addParagraphLine(line)
					return
				}
			}
		}
		p.closeBlockquote(line, events)
	}

	if level, ok := setextHeading(line.text); ok && len(p.paragraph.lines) > 0 && !p.inList && !p.inListItem {
		p.closeSetextHeading(level, Span{Start: line.start, End: line.end}, events)
		return
	}

	if thematicBreak(line.text) {
		p.ensureDocument(events)
		p.closeParagraph(events)
		p.closeListItem(events)
		p.closeList(events)
		p.emitThematicBreak(line, events)
		return
	}

	if item, ok := listItem(line.text); ok {
		p.ensureDocument(events)
		if !p.inList && len(p.paragraph.lines) > 0 && item.data.Ordered && item.data.Start != 1 {
			p.addParagraphLine(line)
			return
		}
		p.closeParagraph(events)
		if !p.inList || p.listData.Ordered != item.data.Ordered || p.listData.Marker != item.data.Marker {
			p.closeList(events)
			p.inList = true
			p.listData = listBlockData(item.data)
			p.listLoose = false
			data := p.listData
			p.emitBlockStart(events, Event{Kind: EventEnterBlock, Block: BlockList, List: &data, Span: Span{Start: line.start, End: line.end}})
		}
		p.closeListItem(events)
		p.inListItem = true
		p.listItemIndent = item.contentIndent
		p.listItemBlankLine = false
		data := item.data
		p.emitBlockStart(events, Event{Kind: EventEnterBlock, Block: BlockListItem, List: &data, Span: Span{Start: line.start, End: line.end}})
		if strings.TrimSpace(item.content) != "" {
			inner := line
			inner.text = item.content
			p.processListItemFirstLine(inner, events)
		}
		return
	}

	if p.inListItem {
		// Not enough indent and not a list item — close list.
		p.closeParagraph(events)
		p.closeListItem(events)
		p.closeList(events)
	} else if p.inList {
		if p.tryStartTable(line, events) {
			return
		}
		p.closeParagraph(events)
		p.closeListItem(events)
		p.closeList(events)
	}

	p.processNonContainerLine(line, events)
}

func (p *parser) processNonContainerLine(line lineInfo, events *[]Event) {
	if p.tryStartTable(line, events) {
		return
	}
	if marker, n, indent, info, ok := openingFence(line.text); ok {
		p.ensureDocument(events)
		p.closeParagraph(events)
		p.closeIndentedCode(events)
		p.fence = fenceState{open: true, marker: marker, length: n, indent: indent, info: info}
		p.emitBlockStart(events, Event{Kind: EventEnterBlock, Block: BlockFencedCode, Info: info, Span: Span{Start: line.start, End: line.end}})
		return
	}

	// HTML block detection (types 1-6 can interrupt paragraphs,
	// type 7 cannot).
	if htmlType, htmlEnd := detectHTMLBlockStart(line.text); htmlType > 0 {
		if htmlType <= 6 || len(p.paragraph.lines) == 0 {
			p.ensureDocument(events)
			p.closeParagraph(events)
			p.closeIndentedCode(events)
			p.emitBlockStart(events, Event{Kind: EventEnterBlock, Block: BlockHTML, Span: Span{Start: line.start, End: line.end}})
			p.inHTMLBlock = true
			p.htmlBlockType = htmlType
			p.htmlBlockEnd = htmlEnd
			p.processHTMLBlockLine(line, events)
			return
		}
	}

	if level, text, ok := heading(line.text); ok {
		p.ensureDocument(events)
		p.closeParagraph(events)
		p.drainPendingBlocks(events)
		span := Span{Start: line.start, End: line.end}
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockHeading, Level: level, Span: span})
		*events = append(*events, p.parseInline(text, span)...)
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockHeading, Level: level, Span: span})
		return
	}

	if thematicBreak(line.text) {
		p.ensureDocument(events)
		p.closeParagraph(events)
		p.emitThematicBreak(line, events)
		return
	}

	if indentedCode(line.text) {
		if len(p.paragraph.lines) > 0 {
			inner := line
			inner.text = strings.TrimLeft(line.text, " \t")
			p.addParagraphLine(inner)
			return
		}
		p.ensureDocument(events)
		p.closeParagraph(events)
		if !p.inIndented {
			p.inIndented = true
			p.emitBlockStart(events, Event{Kind: EventEnterBlock, Block: BlockIndentedCode, Span: Span{Start: line.start, End: line.end}})
		}
		p.emitIndentedCodeLine(line, events)
		return
	}

	p.addParagraphLine(line)
}

func (p *parser) ensureDocument(events *[]Event) {
	if p.started {
		return
	}
	p.started = true
	*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockDocument})
}

func (p *parser) addParagraphLine(line lineInfo) {
	text := line.text
	if len(p.paragraph.lines) == 0 {
		if indent, bytes := leadingIndent(text); indent <= 3 {
			text = text[bytes:]
		}
	} else {
		text = strings.TrimLeft(text, " \t")
	}
	p.paragraph.lines = append(p.paragraph.lines, paragraphLine{
		text: text,
		span: Span{Start: line.start, End: line.end},
	})
}

func (p *parser) closeParagraph(events *[]Event) {
	if len(p.paragraph.lines) == 0 {
		return
	}
	p.ensureDocument(events)
	start := p.paragraph.lines[0].span.Start
	end := p.paragraph.lines[len(p.paragraph.lines)-1].span.End
	var text strings.Builder
	for i, line := range p.paragraph.lines {
		if i > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(line.text)
	}
	p.pendingBlocks = append(p.pendingBlocks, pendingBlock{
		text: text.String(),
		span: Span{Start: start, End: end},
	})
	clear(p.paragraph.lines)
	if cap(p.paragraph.lines) > 1024 {
		p.paragraph.lines = nil
	} else {
		p.paragraph.lines = p.paragraph.lines[:0]
	}
}

// drainPendingBlocks inline-parses and emits all buffered paragraph/heading
// blocks. Called before emitting any non-definition block and at Flush, so
// that forward link reference definitions are available for resolution.
func (p *parser) drainPendingBlocks(events *[]Event) {
	for _, pb := range p.pendingBlocks {
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockParagraph, Span: pb.span})
		*events = append(*events, p.parseInline(pb.text, pb.span)...)
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockParagraph, Span: pb.span})
	}
	p.pendingBlocks = p.pendingBlocks[:0]
}

// drainPendingBlocksEager emits pending blocks that don't contain bracket
// syntax (and thus can't benefit from deferred reference resolution).
// Blocks with brackets are kept pending for forward reference resolution.
func (p *parser) drainPendingBlocksEager(events *[]Event) {
	// In-place filter: kept shares the backing array with p.pendingBlocks.
	// This is safe because the kept write index never advances past the
	// read index — every iteration either keeps or emits the element.
	kept := p.pendingBlocks[:0]
	for _, pb := range p.pendingBlocks {
		if strings.ContainsAny(pb.text, "[]") {
			kept = append(kept, pb)
			continue
		}
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockParagraph, Span: pb.span})
		*events = append(*events, p.parseInline(pb.text, pb.span)...)
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockParagraph, Span: pb.span})
	}
	p.pendingBlocks = kept
}

// emitBlockStart drains any pending paragraph/heading blocks before emitting
// a new structural block. This ensures forward link reference definitions
// collected after a pending paragraph are available for inline resolution.
func (p *parser) emitBlockStart(events *[]Event, ev Event) {
	p.drainPendingBlocks(events)
	*events = append(*events, ev)
}

func (p *parser) closeSetextHeading(level int, underline Span, events *[]Event) {
	if len(p.paragraph.lines) == 0 {
		return
	}
	p.ensureDocument(events)
	p.drainPendingBlocks(events)
	start := p.paragraph.lines[0].span.Start
	end := underline.End
	*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockHeading, Level: level, Span: Span{Start: start, End: end}})
	var text strings.Builder
	for i, line := range p.paragraph.lines {
		if i > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(line.text)
	}
	*events = append(*events, p.parseInline(text.String(), Span{Start: start, End: end})...)
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockHeading, Level: level, Span: Span{Start: start, End: end}})
	clear(p.paragraph.lines)
	if cap(p.paragraph.lines) > 1024 {
		p.paragraph.lines = nil
	} else {
		p.paragraph.lines = p.paragraph.lines[:0]
	}
}

func (p *parser) tryStartTable(line lineInfo, events *[]Event) bool {
	if p.table.active || len(p.paragraph.lines) != 1 {
		return false
	}
	align, ok := parseTableSeparator(line.text)
	if !ok {
		return false
	}
	header := p.paragraph.lines[0]
	_, headerHasPipe := splitTableRow(header.text)
	if !headerHasPipe {
		return false
	}

	p.ensureDocument(events)
	p.closeIndentedCode(events)
	clear(p.paragraph.lines)
	if cap(p.paragraph.lines) > 1024 {
		p.paragraph.lines = nil
	} else {
		p.paragraph.lines = p.paragraph.lines[:0]
	}

	tableAlign := append([]TableAlign(nil), align...)
	p.table = tableState{
		active: true,
		align:  tableAlign,
		span:   Span{Start: header.span.Start, End: line.end},
	}
	p.emitBlockStart(events, Event{
		Kind:  EventEnterBlock,
		Block: BlockTable,
		Table: &TableData{Align: append([]TableAlign(nil), tableAlign...)},
		Span:  p.table.span,
	})
	headerLine := lineInfo{text: header.text, start: header.span.Start, end: header.span.End}
	p.emitTableRow(headerLine.text, headerLine, events)
	return true
}

func (p *parser) processActiveTableLine(line lineInfo, events *[]Event) bool {
	if !p.table.active {
		return false
	}
	cells, hasPipe := splitTableRow(line.text)
	if !hasPipe && len(p.table.align) != 1 {
		return false
	}
	if len(cells) == 0 {
		return false
	}
	p.table.span.End = line.end
	p.emitTableRow(line.text, line, events)
	return true
}

func (p *parser) emitTableRow(text string, line lineInfo, events *[]Event) {
	rowSpan := Span{Start: line.start, End: line.end}
	*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockTableRow, Span: rowSpan})
	cells, _ := splitTableRow(text)
	for _, cell := range cells {
		cellSpan := Span{Start: line.start, End: line.end}
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockTableCell, Span: cellSpan})
		*events = append(*events, p.parseInline(cell, cellSpan)...)
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockTableCell, Span: cellSpan})
	}
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockTableRow, Span: rowSpan})
}

func (p *parser) closeTable(events *[]Event) {
	if !p.table.active {
		return
	}
	span := p.table.span
	align := append([]TableAlign(nil), p.table.align...)
	p.table = tableState{}
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockTable, Table: &TableData{Align: align}, Span: span})
}

func parseTableSeparator(line string) ([]TableAlign, bool) {
	cells, hasPipe := splitTableRow(line)
	if !hasPipe || len(cells) == 0 {
		return nil, false
	}
	align := make([]TableAlign, 0, len(cells))
	for _, cell := range cells {
		a, ok := parseTableAlign(cell)
		if !ok {
			return nil, false
		}
		align = append(align, a)
	}
	return align, true
}

func parseTableAlign(cell string) (TableAlign, bool) {
	cell = strings.TrimSpace(cell)
	if len(cell) == 0 {
		return TableAlignNone, false
	}
	left := strings.HasPrefix(cell, ":")
	right := strings.HasSuffix(cell, ":")
	if left {
		cell = cell[1:]
	}
	if right && len(cell) > 0 {
		cell = cell[:len(cell)-1]
	}
	if len(cell) < 3 {
		return TableAlignNone, false
	}
	for i := 0; i < len(cell); i++ {
		if cell[i] != '-' {
			return TableAlignNone, false
		}
	}
	switch {
	case left && right:
		return TableAlignCenter, true
	case left:
		return TableAlignLeft, true
	case right:
		return TableAlignRight, true
	default:
		return TableAlignNone, true
	}
}

func splitTableRow(line string) ([]string, bool) {
	s := strings.TrimSpace(line)
	if s == "" {
		return nil, false
	}
	hasPipe := false
	if s[0] == '|' {
		hasPipe = true
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '|' {
		hasPipe = true
		s = s[:len(s)-1]
	}
	var cells []string
	start := 0
	escaped := false
	codeRun := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '`' {
			run := countRun(s[i:], '`')
			if codeRun == 0 {
				codeRun = run
			} else if run == codeRun {
				codeRun = 0
			}
			i += run - 1
			continue
		}
		if c == '|' && codeRun == 0 {
			cells = append(cells, strings.TrimSpace(s[start:i]))
			start = i + 1
			hasPipe = true
		}
	}
	cells = append(cells, strings.TrimSpace(s[start:]))
	return cells, hasPipe
}

func (p *parser) startLinkReferenceDefinition(line lineInfo) bool {
	label, text, rest, ok := parseLinkReferenceDefinitionStart(line.text)
	if !ok {
		return false
	}
	state := linkReferenceDefinitionState{
		active: true,
		label:  label,
		lines:  []lineInfo{line},
	}
	dest, title, hasDest, hasTitle, pending, ok := parseLinkReferenceDefinitionTail(text, rest)
	if !ok {
		return false
	}
	state.dest = dest
	state.hasDest = hasDest
	state.title = title
	p.refDef = state
	if hasTitle {
		p.finishLinkReferenceDefinition()
		return true
	}
	if pending {
		return true
	}
	p.finishLinkReferenceDefinition()
	return true
}

func (p *parser) continueLinkReferenceDefinition(line lineInfo, events *[]Event) bool {
	if strings.TrimSpace(line.text) == "" {
		if !p.refDef.hasDest {
			p.failLinkReferenceDefinition()
		} else {
			p.finishLinkReferenceDefinition()
		}
		return false
	}

	if !p.refDef.hasDest {
		dest, title, hasDest, hasTitle, pending, ok := parseLinkReferenceDefinitionTail(line.text, 0)
		if !ok || !hasDest {
			p.failLinkReferenceDefinition()
			return false
		}
		p.refDef.lines = append(p.refDef.lines, line)
		p.refDef.dest = dest
		p.refDef.hasDest = hasDest
		p.refDef.title = title
		if hasTitle {
			p.finishLinkReferenceDefinition()
			return true
		}
		if pending {
			return true
		}
		p.finishLinkReferenceDefinition()
		return true
	}

	title, next, ok := parseInlineLinkTitle(line.text, skipMarkdownSpace(line.text, 0))
	if !ok {
		p.finishLinkReferenceDefinition()
		return false
	}
	if skipMarkdownSpace(line.text, next) != len(line.text) {
		p.finishLinkReferenceDefinition()
		return false
	}
	p.refDef.lines = append(p.refDef.lines, line)
	p.refDef.title = title
	p.finishLinkReferenceDefinition()
	return true
}

func (p *parser) closePendingLinkReferenceDefinition(events *[]Event) {
	if !p.refDef.active {
		return
	}
	if !p.refDef.hasDest {
		p.failLinkReferenceDefinition()
		p.closeParagraph(events)
		return
	}
	p.finishLinkReferenceDefinition()
}

func (p *parser) finishLinkReferenceDefinition() {
	if !p.refDef.active {
		return
	}
	if p.refDef.hasDest {
		if p.refs == nil {
			p.refs = make(map[string]linkReference)
		}
		if _, exists := p.refs[p.refDef.label]; !exists {
			p.refs[p.refDef.label] = linkReference{dest: p.refDef.dest, title: p.refDef.title}
		}
	}
	p.refDef = linkReferenceDefinitionState{}
}

func (p *parser) failLinkReferenceDefinition() {
	if !p.refDef.active {
		return
	}
	for _, line := range p.refDef.lines {
		p.addParagraphLine(line)
	}
	p.refDef = linkReferenceDefinitionState{}
}

func (p *parser) emitThematicBreak(line lineInfo, events *[]Event) {
	p.drainPendingBlocks(events)
	*events = append(*events,
		Event{Kind: EventEnterBlock, Block: BlockThematicBreak, Span: Span{Start: line.start, End: line.end}},
		Event{Kind: EventExitBlock, Block: BlockThematicBreak, Span: Span{Start: line.start, End: line.end}},
	)
}

func (p *parser) emitIndentedCodeLine(line lineInfo, events *[]Event) {
	for p.indentedBlankLines > 0 {
		*events = append(*events,
			Event{Kind: EventText, Text: ""},
			Event{Kind: EventLineBreak},
		)
		p.indentedBlankLines--
	}
	text := stripIndentColumns(line.text, 4)
	*events = append(*events,
		Event{Kind: EventText, Text: text, Span: Span{Start: line.start, End: line.end}},
		Event{Kind: EventLineBreak, Span: Span{Start: line.end, End: line.end}},
	)
}

func (p *parser) closeIndentedCode(events *[]Event) {
	if !p.inIndented {
		return
	}
	p.inIndented = false
	p.indentedBlankLines = 0
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockIndentedCode})
}

func (p *parser) closeContainers(events *[]Event) {
	p.closeTable(events)
	p.closeIndentedCode(events)
	// Close blockquote before list — the list may be inside the blockquote.
	if p.inBlockquote {
		p.closeBlockquote(lineInfo{}, events)
	}
	if p.inList {
		p.closeListItem(events)
		p.closeList(events)
	}
}

// processHTMLBlockLine handles a line inside an HTML block.
func (p *parser) processHTMLBlockLine(line lineInfo, events *[]Event) {
	// Types 6-7: blank line closes the block.
	if (p.htmlBlockType == 6 || p.htmlBlockType == 7) && strings.TrimSpace(line.text) == "" {
		p.closeHTMLBlock(events)
		return
	}

	// Emit the line as raw text.
	*events = append(*events,
		Event{Kind: EventText, Text: line.text, Span: Span{Start: line.start, End: line.end}},
		Event{Kind: EventLineBreak, Span: Span{Start: line.end, End: line.end}},
	)

	// Check end condition for types 1-5.
	if p.htmlBlockType >= 1 && p.htmlBlockType <= 5 {
		if strings.Contains(strings.ToLower(line.text), strings.ToLower(p.htmlBlockEnd)) {
			p.closeHTMLBlock(events)
		}
	}
}

func (p *parser) closeHTMLBlock(events *[]Event) {
	if !p.inHTMLBlock {
		return
	}
	p.inHTMLBlock = false
	p.htmlBlockType = 0
	p.htmlBlockEnd = ""
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockHTML})
}

func (p *parser) closeBlockquote(line lineInfo, events *[]Event) {
	if !p.inBlockquote {
		return
	}
	p.closeParagraph(events)
	p.drainPendingBlocks(events)
	p.closeIndentedCode(events)
	p.closeFencedCode(events)
	// Set inBlockquote = false BEFORE closing lists to prevent
	// mutual recursion: closeListItem -> closeBlockquote -> closeListItem.
	bqOpenedInsideList := p.bqInsideListItem
	p.inBlockquote = false
	// Close all lists that were opened inside this blockquote.
	// Don't close any lists if the blockquote was opened inside a list item
	// — those lists belong to the outer context.
	if !bqOpenedInsideList {
		for p.inList {
			if p.inListItem {
				p.closeParagraph(events)
				p.drainPendingBlocks(events)
				p.inListItem = false
				*events = append(*events, Event{Kind: EventExitBlock, Block: BlockListItem})
			}
			data := p.listData
			data.Tight = !p.listLoose
			p.inList = false
			*events = append(*events, Event{Kind: EventExitBlock, Block: BlockList, List: &data})
			p.listData = ListData{}
			p.listLoose = false
			p.popList()
		}
	}
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockBlockquote, Span: Span{Start: line.start, End: line.start}})
}

func (p *parser) closeListItem(events *[]Event) {
	if !p.inListItem {
		return
	}
	p.closeParagraph(events)
	p.drainPendingBlocks(events)
	p.closeIndentedCode(events)
	p.closeFencedCode(events)
	// Only close the blockquote if it was opened inside this list item.
	// When the list item is inside the blockquote (bqInsideListItem is
	// false), the blockquote is the outer container and must not be
	// closed here.
	if p.inBlockquote && p.bqInsideListItem {
		p.closeBlockquote(lineInfo{}, events)
	}
	// closeBlockquote may have already closed this list item
	// (when the list was inside the blockquote). Bail out to
	// avoid emitting a duplicate exit event.
	if !p.inListItem {
		return
	}
	// Close any open sublists inside this list item.
	// closeList closes the current list (and its open item), then
	// popList restores the outer list context. We must not close
	// the restored outer list item — only the original one.
	if len(p.listStack) > 0 {
		// The current list item will be closed by closeList as part
		// of closing the innermost list. After the loop, popList may
		// restore an outer list item — leave that alone.
		for len(p.listStack) > 0 {
			p.closeList(events)
		}
		// closeList already emitted exit list_item for the original
		// item. Don't emit another one.
		return
	}
	p.inListItem = false
	p.listItemBlankLine = false
	p.listItemIndent = 0
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockListItem})
}


func (p *parser) processFenceLine(line lineInfo, events *[]Event) {
	if p.isClosingFence(line.text) {
		p.fence.open = false
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockFencedCode, Span: Span{Start: line.start, End: line.end}})
		return
	}
	p.ensureDocument(events)
	*events = append(*events,
		Event{Kind: EventText, Text: stripFenceIndent(line.text, p.fence.indent), Span: Span{Start: line.start, End: line.end}},
		Event{Kind: EventLineBreak, Span: Span{Start: line.end, End: line.end}},
	)
}

func (p *parser) closeFencedCode(events *[]Event) {
	if !p.fence.open {
		return
	}
	p.fence.open = false
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockFencedCode})
}

func (p *parser) closeList(events *[]Event) {
	if !p.inList {
		return
	}
	// Close any open item in this list first.
	if p.inListItem {
		p.closeParagraph(events)
		p.drainPendingBlocks(events)
		p.closeIndentedCode(events)
		p.closeFencedCode(events)
		// Only close the blockquote if it was opened inside this
		// list item. When the list is inside the blockquote, the
		// blockquote is the outer container.
		if p.inBlockquote && p.bqInsideListItem {
			p.closeBlockquote(lineInfo{}, events)
		}
		p.inListItem = false
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockListItem})
	}
	p.inList = false
	data := p.listData
	data.Tight = !p.listLoose
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockList, List: &data})
	p.listData = ListData{}
	p.listLoose = false
	// Restore outer list state if we were in a sublist.
	p.popList()
}

func (p *parser) pushList() {
	p.listStack = append(p.listStack, savedList{
		inList:            p.inList,
		listData:          p.listData,
		listLoose:         p.listLoose,
		inListItem:        p.inListItem,
		listItemIndent:    p.listItemIndent,
		listItemBlankLine: p.listItemBlankLine,
	})
}

func (p *parser) popList() {
	if len(p.listStack) == 0 {
		return
	}
	saved := p.listStack[len(p.listStack)-1]
	p.listStack = p.listStack[:len(p.listStack)-1]
	p.inList = saved.inList
	p.listData = saved.listData
	p.listLoose = saved.listLoose
	p.inListItem = saved.inListItem
	p.listItemIndent = saved.listItemIndent
	p.listItemBlankLine = saved.listItemBlankLine
}

// processListItemFirstLine handles the first content line of a list item.
// Unlike continuation lines, this is the content after the marker on the
// same line. It checks for block-level constructs before falling back to
// paragraph text.
func (p *parser) processListItemFirstLine(line lineInfo, events *[]Event) {
	// HTML block inside list item.
	if htmlType, htmlEnd := detectHTMLBlockStart(line.text); htmlType > 0 {
		p.inHTMLBlock = true
		p.htmlBlockType = htmlType
		p.htmlBlockEnd = htmlEnd
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockHTML, Span: Span{Start: line.start, End: line.end}})
		p.processHTMLBlockLine(line, events)
		return
	}
	// Indented code block (4+ spaces of content indent).
	if indentedCode(line.text) {
		p.inIndented = true
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockIndentedCode, Span: Span{Start: line.start, End: line.end}})
		p.emitIndentedCodeLine(line, events)
		return
	}
	// Fenced code block.
	if marker, n, indent, info, ok := openingFence(line.text); ok {
		p.fence = fenceState{open: true, marker: marker, length: n, indent: indent, info: info}
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockFencedCode, Info: info, Span: Span{Start: line.start, End: line.end}})
		return
	}
	// ATX heading.
	if level, text, ok := heading(line.text); ok {
		span := Span{Start: line.start, End: line.end}
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockHeading, Level: level, Span: span})
		*events = append(*events, p.parseInline(text, span)...)
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockHeading, Span: span})
		return
	}
	// Thematic break inside list item.
	if thematicBreak(line.text) {
		span := Span{Start: line.start, End: line.end}
		*events = append(*events,
			Event{Kind: EventEnterBlock, Block: BlockThematicBreak, Span: span},
			Event{Kind: EventExitBlock, Block: BlockThematicBreak, Span: span},
		)
		return
	}
	// Sublist on same line as outer marker (e.g., "- - foo").
	if item, ok := listItem(line.text); ok {
		p.pushList()
		p.inList = true
		p.listData = listBlockData(item.data)
		p.listLoose = false
		data := p.listData
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockList, List: &data, Span: Span{Start: line.start, End: line.end}})
		p.inListItem = true
		p.listItemIndent = item.contentIndent
		p.listItemBlankLine = false
		idata := item.data
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockListItem, List: &idata, Span: Span{Start: line.start, End: line.end}})
		if strings.TrimSpace(item.content) != "" {
			inner := line
			inner.text = item.content
			p.processListItemFirstLine(inner, events)
		}
		return
	}
	// Default: paragraph text.
	p.addParagraphLine(line)
}

// processListItemContent processes a line of content inside a list item.
// The line has already been stripped of the list item's content indent.
// It handles sublists, blockquotes, fenced code, headings, and other
// block-level constructs that can appear inside list items.
func (p *parser) processListItemContent(line lineInfo, events *[]Event) {
	// Handle fenced code continuation inside list items.
	if p.fence.open {
		if p.isClosingFence(line.text) {
			p.fence.open = false
			*events = append(*events, Event{Kind: EventExitBlock, Block: BlockFencedCode, Span: Span{Start: line.start, End: line.end}})
			return
		}
		*events = append(*events,
			Event{Kind: EventText, Text: stripFenceIndent(line.text, p.fence.indent), Span: Span{Start: line.start, End: line.end}},
			Event{Kind: EventLineBreak, Span: Span{Start: line.end, End: line.end}},
		)
		return
	}

	// Handle indented code continuation inside list items.
	if p.inIndented {
		if indentedCode(line.text) {
			p.emitIndentedCodeLine(line, events)
			return
		}
		if strings.TrimSpace(line.text) == "" {
			p.indentedBlankLines++
			return
		}
		p.closeIndentedCode(events)
	}

	// Blank line inside continuation.
	if strings.TrimSpace(line.text) == "" {
		p.closeParagraph(events)
		p.closeIndentedCode(events)
		return
	}

	// Setext heading inside list item.
	if level, ok := setextHeading(line.text); ok && len(p.paragraph.lines) > 0 {
		p.closeSetextHeading(level, Span{Start: line.start, End: line.end}, events)
		return
	}

	// Thematic break inside list item continuation.
	if thematicBreak(line.text) {
		p.closeParagraph(events)
		p.drainPendingBlocks(events)
		span := Span{Start: line.start, End: line.end}
		*events = append(*events,
			Event{Kind: EventEnterBlock, Block: BlockThematicBreak, Span: span},
			Event{Kind: EventExitBlock, Block: BlockThematicBreak, Span: span},
		)
		return
	}

	// Blockquote inside list item.
	if content, ok := blockquoteContent(line.text); ok {
		p.closeParagraph(events)
		p.drainPendingBlocks(events)
		if !p.inBlockquote {
			p.inBlockquote = true; p.bqInsideListItem = p.inListItem
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockBlockquote, Span: Span{Start: line.start, End: line.end}})
		}
		if strings.TrimSpace(content) == "" {
			p.closeParagraph(events)
			return
		}
		inner := line
		inner.text = content
		p.processNonContainerLine(inner, events)
		return
	}
	if p.inBlockquote {
		if len(p.paragraph.lines) > 0 && !thematicBreak(line.text) {
			if _, _, _, _, isFence := openingFence(line.text); !isFence {
				if _, ok := listItem(line.text); !ok {
					p.addParagraphLine(line)
					return
				}
			}
		}
		p.closeBlockquote(line, events)
	}

	// HTML block inside list item continuation.
	if htmlType, htmlEnd := detectHTMLBlockStart(line.text); htmlType > 0 {
		if htmlType <= 6 || len(p.paragraph.lines) == 0 {
			p.closeParagraph(events)
			p.drainPendingBlocks(events)
			p.inHTMLBlock = true
			p.htmlBlockType = htmlType
			p.htmlBlockEnd = htmlEnd
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockHTML, Span: Span{Start: line.start, End: line.end}})
			p.processHTMLBlockLine(line, events)
			return
		}
	}

	// Sublist inside list item.
	if item, ok := listItem(line.text); ok {
		p.closeParagraph(events)
		p.drainPendingBlocks(events)
		// Check if this is a sibling in an existing sublist (not the outer list).
		// A sibling must be at indent 0 (same level as existing sublist items).
		// An indented list item creates a deeper sublist.
		itemIndent, _ := leadingIndent(line.text)
		inSublist := len(p.listStack) > 0 && p.inList
		if inSublist && itemIndent == 0 && p.listData.Ordered == item.data.Ordered && p.listData.Marker == item.data.Marker {
			// Sibling item in the existing sublist.
			// Close just the current item, not the sublist.
			if p.inListItem {
				p.closeParagraph(events)
				p.drainPendingBlocks(events)
				p.closeIndentedCode(events)
				p.closeFencedCode(events)
				p.inListItem = false
				*events = append(*events, Event{Kind: EventExitBlock, Block: BlockListItem})
			}
		} else {
			// New sublist — save outer state.
			p.pushList()
			p.inList = true
			p.listData = listBlockData(item.data)
			p.listLoose = false
			data := p.listData
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockList, List: &data, Span: Span{Start: line.start, End: line.end}})
		}
		p.inListItem = true
		p.listItemIndent = item.contentIndent
		p.listItemBlankLine = false
		idata := item.data
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockListItem, List: &idata, Span: Span{Start: line.start, End: line.end}})
		if strings.TrimSpace(item.content) != "" {
			inner := line
			inner.text = item.content
			p.processListItemFirstLine(inner, events)
		}
		return
	}

	// Link reference definition inside list item.
	if len(p.paragraph.lines) == 0 {
		if p.startLinkReferenceDefinition(line) {
			return
		}
	}

	p.processNonContainerLine(line, events)
}

// stripIndent removes up to n columns of leading indentation from a line.
func stripIndent(line string, n int) string {
	col := 0
	for i := 0; i < len(line); i++ {
		if col >= n {
			return line[i:]
		}
		switch line[i] {
		case ' ':
			col++
		case '\t':
			col += 4 - col%4
			if col > n {
				// Tab extends past the indent boundary; replace with spaces.
				return strings.Repeat(" ", col-n) + line[i+1:]
			}
		default:
			return line[i:]
		}
	}
	return ""
}

func (p *parser) isClosingFence(line string) bool {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return false
	}
	trimmed := line[indentBytes:]
	if len(trimmed) < p.fence.length {
		return false
	}
	for i := 0; i < p.fence.length; i++ {
		if trimmed[i] != p.fence.marker {
			return false
		}
	}
	return strings.TrimSpace(trimmed[p.fence.length:]) == ""
}

func openingFence(line string) (byte, int, int, string, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return 0, 0, 0, "", false
	}
	trimmed := line[indentBytes:]
	if len(trimmed) < 3 {
		return 0, 0, 0, "", false
	}
	marker := trimmed[0]
	if marker != '`' && marker != '~' {
		return 0, 0, 0, "", false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == marker {
		n++
	}
	if n < 3 {
		return 0, 0, 0, "", false
	}
	info := decodeCharacterReferences(unescapeBackslashPunctuation(strings.TrimSpace(trimmed[n:])))
	if marker == '`' && strings.Contains(info, "`") {
		return 0, 0, 0, "", false
	}
	return marker, n, indent, info, true
}

func stripFenceIndent(line string, indent int) string {
	n := 0
	for n < len(line) && n < indent && line[n] == ' ' {
		n++
	}
	return line[n:]
}

func heading(line string) (int, string, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return 0, "", false
	}
	trimmed := line[indentBytes:]
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if level < len(trimmed) && trimmed[level] != ' ' && trimmed[level] != '\t' {
		return 0, "", false
	}
	text := strings.TrimSpace(trimmed[level:])
	if i := closingATXSequence(text); i >= 0 {
		text = strings.TrimSpace(text[:i])
	}
	return level, text, true
}

func closingATXSequence(text string) int {
	i := len(text) - 1
	for i >= 0 && text[i] == '#' {
		i--
	}
	if i == len(text)-1 {
		return -1
	}
	if i < 0 || text[i] == ' ' || text[i] == '\t' {
		return i + 1
	}
	return -1
}

func thematicBreak(line string) bool {
	if indent, _ := leadingIndent(line); indent > 3 {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	var marker byte
	count := 0
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		if c == ' ' || c == '\t' {
			continue
		}
		if marker == 0 {
			if c != '-' && c != '*' && c != '_' {
				return false
			}
			marker = c
		}
		if c != marker {
			return false
		}
		count++
	}
	return count >= 3
}

func setextHeading(line string) (int, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return 0, false
	}
	trimmed := strings.TrimSpace(line[indentBytes:])
	if trimmed == "" {
		return 0, false
	}
	marker := trimmed[0]
	if marker != '=' && marker != '-' {
		return 0, false
	}
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] != marker {
			return 0, false
		}
	}
	if marker == '=' {
		return 1, true
	}
	return 2, true
}

func indentedCode(line string) bool {
	indent, _ := leadingIndent(line)
	return indent >= 4 && strings.TrimSpace(line) != ""
}

func blockquoteContent(line string) (string, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 || indentBytes >= len(line) || line[indentBytes] != '>' {
		return "", false
	}
	content := line[indentBytes+1:]
	if strings.HasPrefix(content, " ") || strings.HasPrefix(content, "\t") {
		content = content[1:]
	}
	return content, true
}

type listItemData struct {
	data         ListData
	content      string
	contentIndent int // column where content starts
}

func listItem(line string) (listItemData, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return listItemData{}, false
	}
	trimmed := line[indentBytes:]

	// Bullet list marker: -, +, or *
	if len(trimmed) >= 1 && strings.ContainsRune("-+*", rune(trimmed[0])) {
		markerWidth := 1 // the bullet character
		if len(trimmed) == 1 {
			// Marker at end of line — content starts at marker + 1 space.
			return listItemData{
				data:          ListData{Ordered: false, Marker: string(trimmed[0]), Tight: true},
				contentIndent: indent + markerWidth + 1,
			}, true
		}
		if trimmed[1] != ' ' && trimmed[1] != '\t' {
			if len(trimmed) < 2 {
				return listItemData{}, false
			}
			// Not a list item — no space after marker.
			goto tryOrdered
		}
		// Count spaces after marker (1-4, per spec).
		padding := countListPadding(trimmed[markerWidth:])
		data := ListData{Ordered: false, Marker: string(trimmed[0]), Tight: true}
		content := trimmed[markerWidth+padding:]
		data.Task, data.Checked, content = parseTaskListItem(content)
		return listItemData{
			data:          data,
			content:       content,
			contentIndent: indent + markerWidth + padding,
		}, true
	}

tryOrdered:
	// Ordered list marker: 1-9 digits followed by . or )
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	if i == 0 || i > 9 || i >= len(trimmed) {
		return listItemData{}, false
	}
	marker := trimmed[i]
	if marker != '.' && marker != ')' {
		return listItemData{}, false
	}
	markerWidth := i + 1 // digits + delimiter
	start, err := strconv.Atoi(trimmed[:i])
	if err != nil {
		return listItemData{}, false
	}
	if markerWidth >= len(trimmed) {
		// Marker at end of line.
		return listItemData{
			data:          ListData{Ordered: true, Start: start, Marker: string(marker), Tight: true},
			contentIndent: indent + markerWidth + 1,
		}, true
	}
	if trimmed[markerWidth] != ' ' && trimmed[markerWidth] != '\t' {
		return listItemData{}, false
	}
	padding := countListPadding(trimmed[markerWidth:])
	data := ListData{Ordered: true, Start: start, Marker: string(marker), Tight: true}
	content := trimmed[markerWidth+padding:]
	data.Task, data.Checked, content = parseTaskListItem(content)
	return listItemData{
		data:          data,
		content:       content,
		contentIndent: indent + markerWidth + padding,
	}, true
}

// countListPadding counts the spaces/tabs after a list marker.
// Per CommonMark, 1-4 spaces are consumed as padding. If the content
// is blank (only spaces), exactly 1 space of padding is used.
func countListPadding(afterMarker string) int {
	spaces := 0
	bytes := 0
	for bytes < len(afterMarker) {
		switch afterMarker[bytes] {
		case ' ':
			spaces++
			bytes++
		case '\t':
			spaces += 4 - spaces%4
			bytes++
		default:
			if spaces > 4 {
				return 1 // content starts with indented code
			}
			return bytes
		}
	}
	// All spaces — blank content, use 1 space padding.
	return 1
}

func listBlockData(item ListData) ListData {
	item.Task = false
	item.Checked = false
	return item
}

func parseTaskListItem(content string) (task bool, checked bool, rest string) {
	if len(content) < 3 || content[0] != '[' || content[2] != ']' {
		return false, false, content
	}
	switch content[1] {
	case ' ':
		checked = false
	case 'x', 'X':
		checked = true
	default:
		return false, false, content
	}
	if len(content) == 3 {
		return true, checked, ""
	}
	if content[3] != ' ' && content[3] != '\t' {
		return false, false, content
	}
	return true, checked, strings.TrimLeft(content[4:], " \t")
}

func leadingIndent(line string) (int, int) {
	columns := 0
	bytes := 0
	for bytes < len(line) {
		switch line[bytes] {
		case ' ':
			columns++
			bytes++
		case '\t':
			columns += 4 - columns%4
			bytes++
		default:
			return columns, bytes
		}
	}
	return columns, bytes
}

func stripIndentColumns(line string, columns int) string {
	current := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			current++
		case '\t':
			current += 4 - current%4
		default:
			return line[i:]
		}
		if current >= columns {
			return line[i+1:]
		}
	}
	return ""
}

func (p *parser) parseInline(text string, span Span) []Event {
	return parseInline(text, span, p.refs)
}

func parseInline(text string, span Span, refs map[string]linkReference) []Event {
	if text == "" {
		return []Event{{Kind: EventText, Text: "", Span: span}}
	}
	tokens := tokenizeInline(text, span, refs)
	tokens = resolveEmphasis(tokens)
	return coalesceInlineTokens(tokens, span)
}

type inlineTokenKind int

const (
	inlineTokenText inlineTokenKind = iota
	inlineTokenSoftBreak
	inlineTokenLineBreak
	inlineTokenDelimiter
)

type inlineToken struct {
	kind    inlineTokenKind
	text    string
	style   InlineStyle
	delim   byte
	run     int
	origRun int // original run length, for rule-of-three checks
	open    bool
	close   bool
}

func tokenizeInline(text string, span Span, refs map[string]linkReference) []inlineToken {
	var tokens []inlineToken
	var prevSource string
	imagePossible := strings.Contains(text, "](")
	linkPossible := strings.Contains(text, "](")
	autolinkPossible := strings.Contains(text, ">")
	for len(text) > 0 {
		if text[0] == '\n' {
			if trimHardBreakMarkerTokens(&tokens) {
				tokens = append(tokens, inlineToken{kind: inlineTokenLineBreak})
			} else {
				tokens = append(tokens, inlineToken{kind: inlineTokenSoftBreak})
			}
			prevSource = "\n"
			text = text[1:]
			continue
		}
		if text[0] == '\\' && len(text) > 1 && isEscapablePunctuation(text[1]) {
			tokens = append(tokens, inlineToken{kind: inlineTokenText, text: text[1:2]})
			prevSource = text[:2]
			text = text[2:]
			continue
		}
		if text[0] == '\\' {
			tokens = append(tokens, inlineToken{kind: inlineTokenText, text: text[:1]})
			prevSource = text[:1]
			text = text[1:]
			continue
		}
		if text[0] == '`' {
			if ev, rest, ok := parseCodeSpan(text, span); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
			n := countRun(text, '`')
			tokens = append(tokens, inlineToken{kind: inlineTokenText, text: text[:n]})
			prevSource = text[:n]
			text = text[n:]
			continue
		}
		if text[0] == '!' {
			if imagePossible {
				if ev, rest, ok := parseInlineImage(text, span); ok {
					tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
					prevSource = text[:len(text)-len(rest)]
					text = rest
					continue
				}
				imagePossible = strings.Contains(text[1:], "](")
			}
			if len(refs) > 0 {
				if ev, rest, ok := parseReferenceImage(text, span, refs); ok {
					tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
					prevSource = text[:len(text)-len(rest)]
					text = rest
					continue
				}
			}
		}
		if text[0] == '[' && linkPossible {
			if ev, rest, labelRaw, ok := parseInlineLink(text, span); ok {
				linkStyle := InlineStyle{Link: ev.Style.Link, LinkTitle: ev.Style.LinkTitle}
				tokens = append(tokens, tokenizeLinkContent(labelRaw, linkStyle, span, refs)...)
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
			linkPossible = strings.Contains(text[1:], "](")
		}
		if text[0] == '[' && len(refs) > 0 {
			if ev, rest, labelRaw, ok := parseReferenceLink(text, span, refs); ok {
				_ = ev
				linkStyle := InlineStyle{Link: ev.Style.Link, LinkTitle: ev.Style.LinkTitle}
				tokens = append(tokens, tokenizeLinkContent(labelRaw, linkStyle, span, refs)...)
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
		}
		// If [ failed as a link opener, emit just the [ as text
		// so the next character gets a chance to be a link opener.
		if text[0] == '[' {
			tokens = append(tokens, inlineToken{kind: inlineTokenText, text: "["})
			prevSource = text[:1]
			text = text[1:]
			continue
		}
		if text[0] == '<' && autolinkPossible {
			if ev, rest, ok := parseAutolink(text, span); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
			// Raw HTML tag (same precedence as autolinks and code spans).
			if tag, ok := parseRawHTMLTag(text); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: tag})
				prevSource = tag
				text = text[len(tag):]
				continue
			}
			autolinkPossible = strings.Contains(text[1:], ">")
		}
		if ev, rest, ok := parseAutolinkLiteral(text, span, prevSource); ok {
			tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
			prevSource = text[:len(text)-len(rest)]
			text = rest
			continue
		}
		if text[0] == '&' {
			if decoded, rest, ok := parseCharacterReference(text); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: decoded})
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
		}
		if isInlineDelimiterByte(text[0]) {
			n := countRun(text, text[0])
			if text[0] == '~' && n < 2 {
				next := nextInlineDelimiter(text)
				if next <= 0 {
					next = 1
				}
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: text[:next]})
				prevSource = text[:next]
				text = text[next:]
				continue
			}
			open, close := emphasisDelimRun(prevSource, text[n:], text[0], n)
			tokens = append(tokens, inlineToken{kind: inlineTokenDelimiter, text: text[:n], delim: text[0], run: n, origRun: n, open: open, close: close})
			prevSource = text[:n]
			text = text[n:]
			continue
		}
		next := nextInlineDelimiter(text)
		if start := nextAutolinkLiteralStart(text, prevSource); start >= 0 && start < next {
			next = start
		}
		if next <= 0 {
			next = 1
		}
		tokens = append(tokens, inlineToken{kind: inlineTokenText, text: text[:next]})
		prevSource = text[:next]
		text = text[next:]
	}
	return tokens
}

// resolveEmphasis implements the CommonMark "process emphasis" algorithm.
//
// Phase 1: match openers to closers, recording (opener, closer, use) triples.
// Phase 2: walk tokens and use the recorded matches to build nested styled output.
func resolveEmphasis(tokens []inlineToken) []inlineToken {
	if len(tokens) == 0 {
		return tokens
	}

	// Collect delimiter indices.
	var dstack []int // indices into tokens, acts as the delimiter stack
	for i, tok := range tokens {
		if tok.kind == inlineTokenDelimiter {
			dstack = append(dstack, i)
		}
	}
	if len(dstack) == 0 {
		return tokens
	}

	// emphPair records a matched opener-closer pair.
	type emphPair struct {
		openerIdx int
		closerIdx int
		use       int
		delim     byte
	}
	var pairs []emphPair

	// removed[di] = true means dstack[di] has been removed from the stack.
	removed := make([]bool, len(dstack))

	// openersBottom keyed by (delim, closerCanOpen, closerOrigRun%3).
	openersBottom := map[int]int{} // value = dstack index (exclusive lower bound)
	obKey := func(delim byte, canOpen bool, origRun int) int {
		k := int(delim) * 6
		if canOpen {
			k += 3
		}
		k += origRun % 3
		return k
	}

	for ci := 0; ci < len(dstack); ci++ {
		if removed[ci] {
			continue
		}
		closerTokIdx := dstack[ci]
		closer := &tokens[closerTokIdx]
		if !closer.close || (closer.delim != '*' && closer.delim != '_' && closer.delim != '~') {
			continue
		}

		bk := obKey(closer.delim, closer.open, closer.origRun)
		bottom := openersBottom[bk] // dstack index; search only above this

		found := false
		for oi := ci - 1; oi >= bottom; oi-- {
			if removed[oi] {
				continue
			}
			openerTokIdx := dstack[oi]
			opener := &tokens[openerTokIdx]
			if opener.delim != closer.delim || !opener.open {
				continue
			}

			// Rule of three (spec rules 9-10).
			if closer.delim != '~' {
				if (opener.open && opener.close) || (closer.open && closer.close) {
					if (opener.origRun+closer.origRun)%3 == 0 &&
						opener.origRun%3 != 0 && closer.origRun%3 != 0 {
						continue
					}
				}
			}

			var use int
			if closer.delim == '~' {
				if opener.run < 2 || closer.run < 2 {
					continue
				}
				use = 2
			} else if opener.run >= 2 && closer.run >= 2 {
				use = 2
			} else {
				use = 1
			}

			pairs = append(pairs, emphPair{
				openerIdx: openerTokIdx,
				closerIdx: closerTokIdx,
				use:       use,
				delim:     closer.delim,
			})

			// Consume delimiters.
			opener.run -= use
			opener.text = opener.text[:opener.run]
			closer.run -= use
			closer.text = closer.text[use:]

			// Remove delimiters between opener and closer from the stack.
			for ri := oi + 1; ri < ci; ri++ {
				removed[ri] = true
			}

			if opener.run == 0 {
				removed[oi] = true
			}
			if closer.run == 0 {
				removed[ci] = true
			} else {
				ci-- // re-process closer
			}

			found = true
			break
		}

		if !found {
			openersBottom[bk] = ci
			if !closer.open {
				removed[ci] = true
			}
		}
	}

	if len(pairs) == 0 {
		// No matches — emit all delimiters as literal text.
		var out []inlineToken
		for _, tok := range tokens {
			switch tok.kind {
			case inlineTokenDelimiter:
				if tok.run > 0 {
					out = append(out, inlineToken{kind: inlineTokenText, text: tok.text})
				}
			default:
				out = append(out, tok)
			}
		}
		return out
	}

	// Phase 2: emit output.
	//
	// Pairs are recorded in match order (outermost first for nested cases
	// like ***foo***). We need to sort them by opener position and handle
	// nesting. A pair (A, B) is nested inside (C, D) if C < A and B < D.
	//
	// We build a stack of active styles. For each token position, we check
	// if any pair opens or closes here.

	// Build open/close events keyed by token index.
	type styleEvent struct {
		use   int
		delim byte
		open  bool // true = open, false = close
		seq   int  // ordering for multiple events at same position
	}
	events := map[int][]styleEvent{}
	for i, p := range pairs {
		// Open event goes right after the opener token.
		// Close event goes right before the closer token.
		// We key by token index and use a convention:
		// opener: event at openerIdx, phase=open
		// closer: event at closerIdx, phase=close
		events[p.openerIdx] = append(events[p.openerIdx], styleEvent{use: p.use, delim: p.delim, open: true, seq: i})
		events[p.closerIdx] = append(events[p.closerIdx], styleEvent{use: p.use, delim: p.delim, open: false, seq: i})
	}

	// Sort events at each position so opens come before closes.
	// When multiple opens share a token (e.g. ***foo***), the first
	// matched pair is the outermost, so opens sort by seq ascending.
	// Closes are the reverse: inner-first (seq descending).
	for idx := range events {
		ev := events[idx]
		sort.SliceStable(ev, func(i, j int) bool {
			if ev[i].open != ev[j].open {
				return ev[i].open // opens before closes
			}
			if ev[i].open {
				return ev[i].seq < ev[j].seq // opens: outer first
			}
			return ev[i].seq > ev[j].seq // closes: inner first
		})
		events[idx] = ev
	}

	var out []inlineToken
	var styleStack []InlineStyle
	currentStyle := InlineStyle{}

	for i, tok := range tokens {
		if evs, ok := events[i]; ok {
			// Emit remaining delimiter text before processing events.
			if tok.kind == inlineTokenDelimiter && tok.run > 0 {
				out = append(out, inlineToken{kind: inlineTokenText, text: tok.text, style: currentStyle})
			}
			for _, ev := range evs {
				if ev.open {
					styleStack = append(styleStack, currentStyle)
					currentStyle = applyDelimiterStyle(currentStyle, ev.use, ev.delim)
				} else {
					if len(styleStack) > 0 {
						currentStyle = styleStack[len(styleStack)-1]
						styleStack = styleStack[:len(styleStack)-1]
					}
				}
			}
			continue
		}

		switch tok.kind {
		case inlineTokenDelimiter:
			if tok.run > 0 {
				out = append(out, inlineToken{kind: inlineTokenText, text: tok.text, style: currentStyle})
			}
		case inlineTokenText:
			out = append(out, inlineToken{kind: inlineTokenText, text: tok.text, style: mergeInlineStyles(currentStyle, tok.style)})
		case inlineTokenSoftBreak:
			out = append(out, inlineToken{kind: inlineTokenSoftBreak})
		case inlineTokenLineBreak:
			out = append(out, inlineToken{kind: inlineTokenLineBreak})
		}
	}
	return out
}

func applyDelimiterStyle(style InlineStyle, count int, marker byte) InlineStyle {
	out := style
	if marker == '~' {
		out.Strike = true
		return out
	}
	for count >= 2 {
		out.Strong = true
		count -= 2
	}
	if count == 1 {
		out.Emphasis = true
	}
	return out
}

// tokenizeLinkContent tokenizes and resolves emphasis in link label
// text, then applies the link style to every resulting token.
func tokenizeLinkContent(labelRaw string, linkStyle InlineStyle, span Span, refs map[string]linkReference) []inlineToken {
	label := decodeCharacterReferences(unescapeBackslashPunctuation(labelRaw))
	if label == "" {
		return []inlineToken{{kind: inlineTokenText, text: "", style: linkStyle}}
	}
	inner := tokenizeInline(label, span, refs)
	inner = resolveEmphasis(inner)
	for i := range inner {
		inner[i].style = mergeInlineStyles(inner[i].style, linkStyle)
	}
	return inner
}

func mergeInlineStyles(base, add InlineStyle) InlineStyle {
	out := base
	out.Emphasis = out.Emphasis || add.Emphasis
	out.Strong = out.Strong || add.Strong
	out.Strike = out.Strike || add.Strike
	out.Code = out.Code || add.Code
	if add.Link != "" {
		out.Link = add.Link
		out.LinkTitle = add.LinkTitle
	}
	return out
}

func coalesceInlineTokens(tokens []inlineToken, span Span) []Event {
	if len(tokens) == 0 {
		return nil
	}
	var events []Event
	for _, tok := range tokens {
		switch tok.kind {
		case inlineTokenText:
			events = append(events, Event{Kind: EventText, Text: tok.text, Style: tok.style, Span: span})
		case inlineTokenSoftBreak:
			events = append(events, Event{Kind: EventSoftBreak, Span: span})
		case inlineTokenLineBreak:
			events = append(events, Event{Kind: EventLineBreak, Span: span})
		}
	}
	return coalesceText(events)
}

func parseInlineImage(text string, span Span) (Event, string, bool) {
	parsed, rest, ok := parseInlineImageAsLink(text, span)
	if !ok {
		return Event{}, text, false
	}
	return parsed, rest, true
}

func parseReferenceImage(text string, span Span, refs map[string]linkReference) (Event, string, bool) {
	if len(refs) == 0 || !strings.HasPrefix(text, "![") {
		return Event{}, text, false
	}
	ev, rest, _, ok := parseReferenceLink(text[1:], span, refs)
	if !ok {
		return Event{}, text, false
	}
	return ev, text[len(text)-len(rest):], true
}

func parseInlineImageAsLink(text string, span Span) (Event, string, bool) {
	if !strings.HasPrefix(text, "![") {
		return Event{}, text, false
	}
	// Images may contain links in their alt text, so skip the
	// nested-link rejection.
	ev, rest, _, ok := parseInlineLinkInner(text[1:], span, false)
	if !ok {
		return Event{}, text, false
	}
	rest = text[len(text)-len(rest):]
	return ev, rest, true
}

func parseInlineLink(text string, span Span) (Event, string, string, bool) {
	return parseInlineLinkInner(text, span, true)
}

func parseInlineLinkInner(text string, span Span, rejectNestedLinks bool) (Event, string, string, bool) {
	closeText := matchingLinkLabelEnd(text)
	if closeText < 0 || closeText+1 >= len(text) || text[closeText+1] != '(' {
		return Event{}, text, "", false
	}
	labelRaw := text[1:closeText]
	// Links cannot nest (CommonMark spec §6.5). If the label text
	// itself contains an inline link, this outer link is invalid.
	// Images are allowed to contain links, so this check is skipped
	// when called from the image path.
	if rejectNestedLinks && containsInlineLink(labelRaw) {
		return Event{}, text, "", false
	}
	label := decodeCharacterReferences(unescapeBackslashPunctuation(labelRaw))
	dest, title, end, ok := parseInlineLinkTail(text[closeText+2:])
	if !ok {
		return Event{}, text, "", false
	}
	return Event{Kind: EventText, Text: label, Style: InlineStyle{Link: dest, LinkTitle: title}, Span: span}, text[closeText+2+end:], labelRaw, true
}

// containsInlineLink reports whether text contains a valid inline link
// [...](...). Used to enforce the no-nested-links rule.
func containsInlineLink(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) {
			i++ // skip escaped char
			continue
		}
		if text[i] == '`' {
			n := countRun(text[i:], '`')
			close := findClosingBackticks(text[i+n:], n)
			if close >= 0 {
				i += n + close + n - 1
				continue
			}
			i += n - 1
			continue
		}
		if text[i] != '[' {
			continue
		}
		// Try to parse an inline link starting at text[i:].
		sub := text[i:]
		close := matchingLinkLabelEnd(sub)
		if close < 0 || close+1 >= len(sub) || sub[close+1] != '(' {
			continue
		}
		if _, _, _, ok := parseInlineLinkTail(sub[close+2:]); ok {
			return true
		}
	}
	return false
}

func parseReferenceLink(text string, span Span, refs map[string]linkReference) (Event, string, string, bool) {
	closeLabel := matchingLinkLabelEnd(text)
	if closeLabel <= 0 {
		return Event{}, text, "", false
	}
	labelRaw := text[1:closeLabel]
	// Links cannot nest (CommonMark spec §6.5).
	if containsInlineLink(labelRaw) {
		return Event{}, text, "", false
	}
	labelText := decodeCharacterReferences(unescapeBackslashPunctuation(labelRaw))
	end := closeLabel + 1

	// Try full reference [text][label] or collapsed [text][].
	// The spec forbids spaces between the two bracket pairs.
	// The second label must not contain unescaped brackets.
	if end < len(text) && text[end] == '[' {
		closeRef := matchingStrictLinkLabelEnd(text[end:])
		if closeRef >= 0 {
			if closeRef > 1 {
				// Full reference: [text][label]
				// Normalize the raw label — no unescaping (spec §6.3).
				refLabelRaw := text[end+1 : end+closeRef]
				ref, ok := refs[normalizeReferenceLabel(refLabelRaw)]
				if ok {
					return Event{Kind: EventText, Text: labelText, Style: InlineStyle{Link: ref.dest, LinkTitle: ref.title}, Span: span}, text[end+closeRef+1:], labelRaw, true
				}
				// Full reference label not found — don't fall through to shortcut.
				return Event{}, text, "", false
			}
			// Collapsed reference: [text][]
			// Use raw first label for lookup.
			ref, ok := refs[normalizeReferenceLabel(labelRaw)]
			if ok {
				return Event{Kind: EventText, Text: labelText, Style: InlineStyle{Link: ref.dest, LinkTitle: ref.title}, Span: span}, text[end+closeRef+1:], labelRaw, true
			}
			return Event{}, text, "", false
		}
		// No matching ] for the second [ — fall through to shortcut.
	}

	// Shortcut reference: [text]
	// Use raw first label for lookup.
	ref, ok := refs[normalizeReferenceLabel(labelRaw)]
	if !ok {
		return Event{}, text, "", false
	}
	return Event{Kind: EventText, Text: labelText, Style: InlineStyle{Link: ref.dest, LinkTitle: ref.title}, Span: span}, text[end:], labelRaw, true
}

func parseLinkReferenceDefinition(line string) (string, linkReference, bool) {
	label, text, rest, ok := parseLinkReferenceDefinitionStart(line)
	if !ok {
		return "", linkReference{}, false
	}
	dest, title, hasDest, hasTitle, pending, ok := parseLinkReferenceDefinitionTail(text, rest)
	if !ok || pending || !hasDest {
		return "", linkReference{}, false
	}
	if !hasTitle {
		title = ""
	}
	return label, linkReference{dest: dest, title: title}, true
}

func parseLinkReferenceDefinitionStart(line string) (string, string, int, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return "", "", 0, false
	}
	text := line[indentBytes:]
	// Link labels in definitions must not contain unescaped brackets.
	closeLabel := matchingStrictLinkLabelEnd(text)
	if closeLabel <= 0 || closeLabel+1 >= len(text) || text[closeLabel+1] != ':' {
		return "", "", 0, false
	}
	// Normalize the raw label text — the spec does not unescape
	// or decode character references during normalization.
	label := normalizeReferenceLabel(text[1:closeLabel])
	if label == "" {
		return "", "", 0, false
	}
	return label, text, closeLabel + 2, true
}

func parseLinkReferenceDefinitionTail(text string, start int) (dest string, title string, hasDest bool, hasTitle bool, pending bool, ok bool) {
	i := skipMarkdownSpace(text, start)
	if i == len(text) {
		return "", "", false, false, true, true
	}
	dest, next, ok := parseInlineLinkDestination(text, i)
	if !ok || next == i {
		return "", "", false, false, false, false
	}
	spaceAfterDest := skipMarkdownSpace(text, next)
	i = spaceAfterDest
	if i < len(text) {
		// Title must be separated from destination by whitespace.
		if spaceAfterDest == next {
			return "", "", false, false, false, false
		}
		parsedTitle, next, ok := parseInlineLinkTitle(text, i)
		if !ok {
			return "", "", false, false, false, false
		}
		title = parsedTitle
		i = skipMarkdownSpace(text, next)
	}
	if i != len(text) {
		return "", "", false, false, false, false
	}
	return dest, title, true, title != "", title == "", true
}

func normalizeReferenceLabel(label string) string {
	fields := strings.Fields(label)
	if len(fields) == 0 {
		return ""
	}
	return unicodeCaseFold(strings.Join(fields, " "))
}

// unicodeCaseFold performs a simple Unicode case fold. It handles
// the standard ToLower mapping plus the special case of
// U+1E9E LATIN CAPITAL LETTER SHARP S (ẞ) which folds to "ss".
func unicodeCaseFold(s string) string {
	if !strings.ContainsRune(s, '\u1e9e') {
		return strings.ToLower(s)
	}
	var b strings.Builder
	for _, r := range s {
		if r == '\u1e9e' {
			b.WriteString("ss")
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// parseRawHTMLTag tries to parse a raw HTML tag at the start of text.
// Returns the full tag string and true if successful.
// Handles open tags, closing tags, comments, processing instructions,
// declarations, and CDATA sections per CommonMark spec §6.6.
func parseRawHTMLTag(text string) (string, bool) {
	if len(text) < 2 || text[0] != '<' {
		return "", false
	}
	// Closing tag: </tagname>
	if len(text) >= 3 && text[1] == '/' {
		if !isASCIILetter(text[2]) {
			return "", false
		}
		i := 3
		for i < len(text) && (isASCIILetter(text[i]) || isASCIIDigit(text[i]) || text[i] == '-') {
			i++
		}
		for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n') {
			i++
		}
		if i < len(text) && text[i] == '>' {
			return text[:i+1], true
		}
		return "", false
	}
	// Comment: <!-- ... -->
	if strings.HasPrefix(text, "<!--") {
		end := strings.Index(text[4:], "-->")
		if end >= 0 {
			return text[:4+end+3], true
		}
		return "", false
	}
	// Processing instruction: <? ... ?>
	if strings.HasPrefix(text, "<?") {
		end := strings.Index(text[2:], "?>")
		if end >= 0 {
			return text[:2+end+2], true
		}
		return "", false
	}
	// Declaration: <! LETTER ... >
	if len(text) >= 3 && text[1] == '!' && isASCIILetter(text[2]) {
		for i := 3; i < len(text); i++ {
			if text[i] == '>' {
				return text[:i+1], true
			}
		}
		return "", false
	}
	// CDATA: <![CDATA[ ... ]]>
	if strings.HasPrefix(text, "<![CDATA[") {
		end := strings.Index(text[9:], "]]>")
		if end >= 0 {
			return text[:9+end+3], true
		}
		return "", false
	}
	// Open tag: <tagname attributes? /? >
	if !isASCIILetter(text[1]) {
		return "", false
	}
	i := 2
	for i < len(text) && (isASCIILetter(text[i]) || isASCIIDigit(text[i]) || text[i] == '-') {
		i++
	}
	// Skip attributes.
	for i < len(text) {
		// Skip whitespace.
		j := i
		for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n') {
			i++
		}
		if i >= len(text) {
			return "", false
		}
		if text[i] == '>' {
			return text[:i+1], true
		}
		if text[i] == '/' {
			if i+1 < len(text) && text[i+1] == '>' {
				return text[:i+2], true
			}
			return "", false
		}
		// Must have whitespace before attribute.
		if i == j {
			return "", false
		}
		// Attribute name: [a-zA-Z_:][a-zA-Z0-9_.:-]*
		if !isASCIILetter(text[i]) && text[i] != '_' && text[i] != ':' {
			return "", false
		}
		i++
		for i < len(text) && (isASCIILetter(text[i]) || isASCIIDigit(text[i]) || text[i] == '_' || text[i] == '.' || text[i] == ':' || text[i] == '-') {
			i++
		}
		// Optional value specification.
		j = i
		for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n') {
			i++
		}
		if i < len(text) && text[i] == '=' {
			i++
			for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\n') {
				i++
			}
			if i >= len(text) {
				return "", false
			}
			if text[i] == '\'' || text[i] == '"' {
				quote := text[i]
				i++
				for i < len(text) && text[i] != quote {
					i++
				}
				if i >= len(text) {
					return "", false
				}
				i++ // skip closing quote
			} else {
				// Unquoted value.
				if text[i] == ' ' || text[i] == '\t' || text[i] == '\n' || text[i] == '"' || text[i] == '\'' || text[i] == '=' || text[i] == '<' || text[i] == '>' || text[i] == '`' {
					return "", false
				}
				for i < len(text) && text[i] != ' ' && text[i] != '\t' && text[i] != '\n' && text[i] != '"' && text[i] != '\'' && text[i] != '=' && text[i] != '<' && text[i] != '>' && text[i] != '`' {
					i++
				}
			}
		} else {
			i = j // no value, restore position
		}
	}
	return "", false
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// htmlBlockType6Tags are the block-level HTML element names for type 6.
var htmlBlockType6Tags = map[string]bool{
	"address": true, "article": true, "aside": true, "base": true,
	"basefont": true, "blockquote": true, "body": true, "caption": true,
	"center": true, "col": true, "colgroup": true, "dd": true,
	"details": true, "dialog": true, "dir": true, "div": true,
	"dl": true, "dt": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "frame": true,
	"frameset": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "head": true,
	"header": true, "hr": true, "html": true, "iframe": true,
	"legend": true, "li": true, "link": true, "main": true,
	"menu": true, "menuitem": true, "nav": true, "noframes": true,
	"ol": true, "optgroup": true, "option": true, "p": true,
	"param": true, "search": true, "section": true, "summary": true,
	"table": true, "tbody": true, "td": true, "tfoot": true,
	"th": true, "thead": true, "title": true, "tr": true,
	"track": true, "ul": true,
}

// detectHTMLBlockStart checks if a line starts an HTML block.
// Returns the block type (1-7) or 0 if not an HTML block.
func detectHTMLBlockStart(line string) (int, string) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return 0, ""
	}
	s := line[indentBytes:]
	if len(s) < 2 || s[0] != '<' {
		return 0, ""
	}

	// Type 1: <pre, <script, <style, <textarea (case-insensitive)
	for _, tag := range []string{"pre", "script", "style", "textarea"} {
		if matchHTMLBlockTag(s[1:], tag) {
			return 1, "</" + tag + ">"
		}
	}

	// Type 2: <!--
	if strings.HasPrefix(s, "<!--") {
		return 2, "-->"
	}

	// Type 3: <?
	if strings.HasPrefix(s, "<?") {
		return 3, "?>"
	}

	// Type 4: <! + ASCII letter
	if len(s) >= 3 && s[1] == '!' && isASCIILetter(s[2]) {
		return 4, ">"
	}

	// Type 5: <![CDATA[
	if strings.HasPrefix(s, "<![CDATA[") {
		return 5, "]]>"
	}

	// Type 6: block-level tag
	closing := s[1] == '/'
	tagStart := 1
	if closing {
		tagStart = 2
	}
	if tagStart < len(s) && isASCIILetter(s[tagStart]) {
		i := tagStart + 1
		for i < len(s) && (isASCIILetter(s[i]) || isASCIIDigit(s[i])) {
			i++
		}
		tagName := strings.ToLower(s[tagStart:i])
		if htmlBlockType6Tags[tagName] {
			if i >= len(s) || s[i] == ' ' || s[i] == '\t' || s[i] == '>' || (s[i] == '/' && i+1 < len(s) && s[i+1] == '>') || s[i] == '\n' {
				return 6, ""
			}
		}
	}

	// Type 7: complete open or closing tag on its own line
	if tag, ok := parseRawHTMLTag(s); ok {
		rest := strings.TrimSpace(s[len(tag):])
		if rest == "" {
			// Must not be a type-1 tag (pre, script, style, textarea)
			tagName := extractTagName(s)
			switch strings.ToLower(tagName) {
			case "pre", "script", "style", "textarea":
				// Already handled as type 1
			default:
				return 7, ""
			}
		}
	}

	return 0, ""
}

// matchHTMLBlockTag checks if text starts with a tag name (case-insensitive)
// followed by space, tab, >, newline, or end of string.
func matchHTMLBlockTag(text, tag string) bool {
	if len(text) < len(tag) {
		return false
	}
	if !strings.EqualFold(text[:len(tag)], tag) {
		return false
	}
	if len(text) == len(tag) {
		return true
	}
	c := text[len(tag)]
	return c == ' ' || c == '\t' || c == '>' || c == '\n'
}

// extractTagName extracts the tag name from an HTML tag string like "<div ...>".
func extractTagName(s string) string {
	if len(s) < 2 || s[0] != '<' {
		return ""
	}
	i := 1
	if i < len(s) && s[i] == '/' {
		i++
	}
	start := i
	for i < len(s) && (isASCIILetter(s[i]) || isASCIIDigit(s[i]) || s[i] == '-') {
		i++
	}
	return s[start:i]
}

func matchingLinkLabelEnd(text string) int {
	return matchingBracketEnd(text, false)
}

// matchingStrictLinkLabelEnd finds the closing ] for a link label,
// rejecting any unescaped [ inside the label (CommonMark spec §6.3).
func matchingStrictLinkLabelEnd(text string) int {
	return matchingBracketEnd(text, true)
}

func matchingBracketEnd(text string, strict bool) int {
	if text == "" || text[0] != '[' {
		return -1
	}
	depth := 0
	escaped := false
	for i := 1; i < len(text); i++ {
		c := text[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		// Code spans take precedence over link structure (CommonMark
		// spec §6.3). Skip over any code span so that brackets inside
		// it are not counted.
		if c == '`' {
			n := countRun(text[i:], '`')
			close := findClosingBackticks(text[i+n:], n)
			if close < 0 {
				// No matching close — the backticks are literal.
				i += n - 1
				continue
			}
			i += n + close + n - 1
			continue
		}
		// Raw HTML tags take precedence over link structure.
		// Skip over any HTML tag so that brackets inside
		// attributes are not counted.
		if c == '<' {
			if tag, ok := parseRawHTMLTag(text[i:]); ok {
				i += len(tag) - 1
				continue
			}
		}
		switch c {
		case '[':
			if strict {
				return -1
			}
			depth++
		case ']':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// findClosingBackticks finds the position of a closing backtick sequence
// of exactly n backticks in text. Returns the byte offset of the start of
// the closing sequence, or -1 if not found.
func findClosingBackticks(text string, n int) int {
	for i := 0; i < len(text); {
		if text[i] != '`' {
			i++
			continue
		}
		run := countRun(text[i:], '`')
		if run == n {
			return i
		}
		i += run
	}
	return -1
}

func parseInlineLinkTail(text string) (string, string, int, bool) {
	i := skipMarkdownSpace(text, 0)
	dest, next, ok := parseInlineLinkDestination(text, i)
	if !ok {
		return "", "", 0, false
	}
	i = skipMarkdownSpace(text, next)
	if i < len(text) && text[i] == ')' {
		return dest, "", i + 1, true
	}
	title, next, ok := parseInlineLinkTitle(text, i)
	if !ok {
		return "", "", 0, false
	}
	i = skipMarkdownSpace(text, next)
	if i >= len(text) || text[i] != ')' {
		return "", "", 0, false
	}
	return dest, title, i + 1, true
}

func parseInlineLinkDestination(text string, start int) (string, int, bool) {
	if start >= len(text) {
		return "", start, false
	}
	if text[start] == '<' {
		escaped := false
		for i := start + 1; i < len(text); i++ {
			c := text[i]
			if c == '\n' {
				return "", start, false
			}
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '>' {
				dest := decodeCharacterReferences(unescapeBackslashPunctuation(text[start+1 : i]))
				return dest, i + 1, true
			}
		}
		return "", start, false
	}
	if text[start] == ')' || isMarkdownSpace(text[start]) {
		return "", start, true
	}
	escaped := false
	depth := 0
	for i := start; i < len(text); i++ {
		c := text[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if isMarkdownSpace(c) {
			if depth == 0 {
				return decodeCharacterReferences(unescapeBackslashPunctuation(text[start:i])), i, true
			}
			return "", start, false
		}
		switch c {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return decodeCharacterReferences(unescapeBackslashPunctuation(text[start:i])), i, true
			}
			depth--
		case '<':
			return "", start, false
		}
	}
	if depth == 0 {
		return decodeCharacterReferences(unescapeBackslashPunctuation(text[start:])), len(text), true
	}
	return "", start, false
}

func parseInlineLinkTitle(text string, start int) (string, int, bool) {
	if start >= len(text) {
		return "", start, false
	}
	open := text[start]
	close := open
	if open == '(' {
		close = ')'
	} else if open != '"' && open != '\'' {
		return "", start, false
	}
	escaped := false
	for i := start + 1; i < len(text); i++ {
		c := text[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == close {
			title := decodeCharacterReferences(unescapeBackslashPunctuation(text[start+1 : i]))
			return title, i + 1, true
		}
		if open == '(' && c == '(' {
			return "", start, false
		}
	}
	return "", start, false
}

func skipMarkdownSpace(text string, start int) int {
	for start < len(text) && isMarkdownSpace(text[start]) {
		start++
	}
	return start
}

func isMarkdownSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func parseCharacterReference(text string) (string, string, bool) {
	end := strings.IndexByte(text, ';')
	if end < 0 || end > 32 {
		return "", text, false
	}
	candidate := text[:end+1]
	decoded, ok := decodeCharacterReference(candidate)
	if !ok {
		return "", text, false
	}
	return decoded, text[end+1:], true
}

func decodeCharacterReferences(text string) string {
	if !strings.Contains(text, "&") {
		return text
	}
	var out strings.Builder
	out.Grow(len(text))
	for len(text) > 0 {
		i := strings.IndexByte(text, '&')
		if i < 0 {
			out.WriteString(text)
			break
		}
		out.WriteString(text[:i])
		decoded, rest, ok := parseCharacterReference(text[i:])
		if !ok {
			out.WriteByte('&')
			text = text[i+1:]
			continue
		}
		out.WriteString(decoded)
		text = rest
	}
	return out.String()
}

func decodeCharacterReference(candidate string) (string, bool) {
	if strings.HasPrefix(candidate, "&#x") || strings.HasPrefix(candidate, "&#X") {
		return decodeNumericCharacterReference(candidate[3:len(candidate)-1], 16)
	}
	if strings.HasPrefix(candidate, "&#") {
		return decodeNumericCharacterReference(candidate[2:len(candidate)-1], 10)
	}
	if !validNamedCharacterReference(candidate) {
		return "", false
	}
	decoded := html.UnescapeString(candidate)
	if decoded == candidate {
		return "", false
	}
	return decoded, true
}

func validNamedCharacterReference(candidate string) bool {
	if len(candidate) < 3 || candidate[0] != '&' || candidate[len(candidate)-1] != ';' {
		return false
	}
	if !isASCIIAlpha(candidate[1]) {
		return false
	}
	for i := 2; i < len(candidate)-1; i++ {
		if !isASCIIAlphaNumeric(candidate[i]) {
			return false
		}
	}
	return true
}

func decodeNumericCharacterReference(digits string, base int) (string, bool) {
	if digits == "" {
		return "", false
	}
	value, err := strconv.ParseInt(digits, base, 32)
	if err != nil || value > utf8.MaxRune {
		return "", false
	}
	r := rune(value)
	if r == 0 || !utf8.ValidRune(r) {
		return "\uFFFD", true
	}
	return string(r), true
}

func findInlineCloser(text string, marker string) int {
	escaped := false
	for i := 0; i <= len(text)-len(marker); i++ {
		if text[i] == '\n' {
			return -1
		}
		if escaped {
			escaped = false
			continue
		}
		if text[i] == '\\' {
			escaped = true
			continue
		}
		if strings.HasPrefix(text[i:], marker) {
			return i
		}
	}
	return -1
}

func parseCodeSpan(text string, span Span) (Event, string, bool) {
	n := countRun(text, '`')
	for i := n; i < len(text); {
		if text[i] != '`' {
			i++
			continue
		}
		closing := countRun(text[i:], '`')
		if closing == n {
			content := normalizeCodeSpan(text[n:i])
			return Event{Kind: EventText, Text: content, Style: InlineStyle{Code: true}, Span: span}, text[i+closing:], true
		}
		i += closing
	}
	return Event{}, text, false
}

func countRun(text string, marker byte) int {
	n := 0
	for n < len(text) && text[n] == marker {
		n++
	}
	return n
}

func normalizeCodeSpan(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	if len(text) >= 2 && text[0] == ' ' && text[len(text)-1] == ' ' && strings.Trim(text, " ") != "" {
		text = text[1 : len(text)-1]
	}
	return text
}

func trimHardBreakMarkerTokens(tokens *[]inlineToken) bool {
	if len(*tokens) == 0 {
		return false
	}
	last := &(*tokens)[len(*tokens)-1]
	if last.kind != inlineTokenText || last.style != (InlineStyle{}) {
		return false
	}
	if strings.HasSuffix(last.text, "\\") {
		last.text = strings.TrimSuffix(last.text, "\\")
		return true
	}
	trimmed := strings.TrimRight(last.text, " ")
	if len(last.text)-len(trimmed) >= 2 {
		last.text = trimmed
		return true
	}
	return false
}

func emphasisDelimRun(prevSource, nextSource string, marker byte, runLen int) (bool, bool) {
	prevRune, _ := lastRune(prevSource)
	nextRune, _ := firstRune(nextSource)
	left := isLeftFlanking(prevRune, nextRune)
	right := isRightFlanking(prevRune, nextRune)
	if marker == '*' || marker == '~' {
		return left, right
	}
	if marker == '_' {
		open := left && (!right || isPunctuationRune(prevRune))
		close := right && (!left || isPunctuationRune(nextRune))
		return open, close
	}
	return false, false
}

func firstRune(text string) (rune, bool) {
	if text == "" {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(text)
	return r, true
}

func lastRune(text string) (rune, bool) {
	if text == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(text)
	return r, true
}

func isLeftFlanking(prev, next rune) bool {
	if next == 0 || isSpaceOrControlRune(next) {
		return false
	}
	if isPunctuationRune(next) && !isSpaceOrControlRune(prev) && !isPunctuationRune(prev) {
		return false
	}
	return true
}

func isRightFlanking(prev, next rune) bool {
	if prev == 0 || isSpaceOrControlRune(prev) {
		return false
	}
	if isPunctuationRune(prev) && !isSpaceOrControlRune(next) && !isPunctuationRune(next) {
		return false
	}
	return true
}

func isSpaceOrControlRune(r rune) bool {
	return r == 0 || unicode.IsSpace(r) || unicode.IsControl(r)
}

func isPunctuationRune(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func trimHardBreakMarker(events []Event) bool {
	if len(events) == 0 {
		return false
	}
	last := &events[len(events)-1]
	if last.Kind != EventText || last.Style != (InlineStyle{}) {
		return false
	}
	if strings.HasSuffix(last.Text, "\\") {
		last.Text = strings.TrimSuffix(last.Text, "\\")
		return true
	}
	trimmed := strings.TrimRight(last.Text, " ")
	if len(last.Text)-len(trimmed) >= 2 {
		last.Text = trimmed
		return true
	}
	return false
}

func parseAutolink(text string, span Span) (Event, string, bool) {
	end := strings.IndexByte(text, '>')
	if end <= 1 {
		return Event{}, text, false
	}
	target := text[1:end]
	if isURIAutolink(target) {
		return Event{Kind: EventText, Text: target, Style: InlineStyle{Link: target}, Span: span}, text[end+1:], true
	}
	if isEmailAutolink(target) {
		return Event{Kind: EventText, Text: target, Style: InlineStyle{Link: "mailto:" + target}, Span: span}, text[end+1:], true
	}
	return Event{}, text, false
}

func parseAutolinkLiteral(text string, span Span, prevSource string) (Event, string, bool) {
	if !autolinkLiteralBoundary(prevSource) {
		return Event{}, text, false
	}
	candidate, _ := scanAutolinkLiteralCandidate(text)
	if candidate == "" {
		return Event{}, text, false
	}
	candidate = trimAutolinkLiteralSuffix(candidate)
	if candidate == "" {
		return Event{}, text, false
	}
	lower := strings.ToLower(candidate)
	switch {
	case strings.HasPrefix(lower, "http://"):
		if len(candidate) > len("http://") && isURIAutolink(candidate) {
			return Event{Kind: EventText, Text: candidate, Style: InlineStyle{Link: candidate}, Span: span}, text[len(candidate):], true
		}
	case strings.HasPrefix(lower, "https://"):
		if len(candidate) > len("https://") && isURIAutolink(candidate) {
			return Event{Kind: EventText, Text: candidate, Style: InlineStyle{Link: candidate}, Span: span}, text[len(candidate):], true
		}
	case strings.HasPrefix(lower, "www."):
		if isWWWAutolink(candidate) {
			return Event{Kind: EventText, Text: candidate, Style: InlineStyle{Link: "http://" + candidate}, Span: span}, text[len(candidate):], true
		}
	}
	if isEmailAutolink(candidate) {
		return Event{Kind: EventText, Text: candidate, Style: InlineStyle{Link: "mailto:" + candidate}, Span: span}, text[len(candidate):], true
	}
	return Event{}, text, false
}

func isURIAutolink(target string) bool {
	colon := strings.IndexByte(target, ':')
	if colon < 2 || colon > 32 || !isASCIIAlpha(target[0]) {
		return false
	}
	for i := 1; i < colon; i++ {
		c := target[i]
		if !isASCIIAlphaNumeric(c) && c != '.' && c != '+' && c != '-' {
			return false
		}
	}
	for i := colon + 1; i < len(target); i++ {
		c := target[i]
		if c <= ' ' || c == '<' || c == '>' {
			return false
		}
	}
	return colon+1 < len(target)
}

func autolinkLiteralBoundary(prevSource string) bool {
	if prevSource == "" {
		return true
	}
	prev, ok := lastRune(prevSource)
	if !ok {
		return true
	}
	return !isAlphaNumeric(prev)
}

func scanAutolinkLiteralCandidate(text string) (string, string) {
	end := 0
	for end < len(text) {
		c := text[end]
		if c <= ' ' || c == '<' || c == '>' {
			break
		}
		end++
	}
	if end == 0 {
		return "", text
	}
	return text[:end], text[end:]
}

func trimAutolinkLiteralSuffix(candidate string) string {
	for len(candidate) > 0 {
		switch candidate[len(candidate)-1] {
		case '.', ',', ':', ';', '!', '?':
			candidate = candidate[:len(candidate)-1]
			continue
		case ')':
			if strings.Count(candidate, ")") > strings.Count(candidate, "(") {
				candidate = candidate[:len(candidate)-1]
				continue
			}
		case ']':
			if strings.Count(candidate, "]") > strings.Count(candidate, "[") {
				candidate = candidate[:len(candidate)-1]
				continue
			}
		}
		return candidate
	}
	return candidate
}

func nextAutolinkLiteralStart(text string, prevSource string) int {
	best := -1
	for _, prefix := range []string{"http://", "https://", "www."} {
		lower := strings.ToLower(text)
		search := 0
		for {
			i := strings.Index(lower[search:], prefix)
			if i < 0 {
				break
			}
			i += search
			if autolinkLiteralBoundaryAt(text, i, prevSource) && (best < 0 || i < best) {
				best = i
				break
			}
			search = i + 1
		}
	}
	for i := 0; i < len(text); i++ {
		if text[i] != '@' {
			continue
		}
		start := i
		for start > 0 && isEmailLocalAutolinkByte(text[start-1]) {
			start--
		}
		if start == i || !autolinkLiteralBoundaryAt(text, start, prevSource) {
			continue
		}
		candidate, _ := scanAutolinkLiteralCandidate(text[start:])
		candidate = trimAutolinkLiteralSuffix(candidate)
		if isEmailAutolink(candidate) && (best < 0 || start < best) {
			best = start
		}
	}
	return best
}

func autolinkLiteralBoundaryAt(text string, start int, prevSource string) bool {
	if start < 0 || start > len(text) {
		return false
	}
	if start == 0 {
		return autolinkLiteralBoundary(prevSource)
	}
	prev, _ := lastRune(text[:start])
	return !isAlphaNumeric(prev)
}

func isEmailLocalAutolinkByte(c byte) bool {
	return isASCIIAlphaNumeric(c) || strings.ContainsRune(".!#$%&'*+/=?^_`{|}~-", rune(c))
}

func isWWWAutolink(target string) bool {
	if len(target) < 5 || !strings.EqualFold(target[:4], "www.") {
		return false
	}
	hostPort := target[4:]
	if hostPort == "" {
		return false
	}
	end := len(hostPort)
	for i := 0; i < len(hostPort); i++ {
		switch hostPort[i] {
		case '/', '?', '#':
			end = i
			i = len(hostPort)
		}
	}
	hostPort = hostPort[:end]
	if hostPort == "" {
		return false
	}
	host := hostPort
	if colon := strings.LastIndexByte(hostPort, ':'); colon >= 0 {
		port := hostPort[colon+1:]
		if port == "" {
			return false
		}
		for i := 0; i < len(port); i++ {
			if port[i] < '0' || port[i] > '9' {
				return false
			}
		}
		host = hostPort[:colon]
	}
	return isDomainAutolink(host)
}

func isEmailAutolink(target string) bool {
	at := strings.IndexByte(target, '@')
	if at <= 0 || at != strings.LastIndexByte(target, '@') || at == len(target)-1 {
		return false
	}
	local, domain := target[:at], target[at+1:]
	if strings.Contains(domain, ".") == false {
		return false
	}
	for i := 0; i < len(local); i++ {
		c := local[i]
		if !isASCIIAlphaNumeric(c) && !strings.ContainsRune(".!#$%&'*+/=?^_`{|}~-", rune(c)) {
			return false
		}
	}
	return isDomainAutolink(domain)
}

func isDomainAutolink(domain string) bool {
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if !isASCIIAlphaNumeric(c) && c != '-' {
				return false
			}
		}
	}
	return true
}

func nextInlineDelimiter(text string) int {
	if i := strings.IndexAny(text, "\n\\*~_`[<&"); i >= 0 {
		return i
	}
	return len(text)
}

func isInlineDelimiterByte(c byte) bool {
	return c == '\n' || c == '\\' || c == '*' || c == '~' || c == '_' || c == '`' || c == '[' || c == '<' || c == '&'
}

func coalesceText(events []Event) []Event {
	if len(events) < 2 {
		return events
	}
	out := events[:0]
	current := events[0]
	var builder strings.Builder
	merging := false
	flush := func() {
		if merging {
			current.Text = builder.String()
			builder.Reset()
			merging = false
		}
		out = append(out, current)
	}
	for _, ev := range events[1:] {
		if current.Kind == EventText && ev.Kind == EventText && sameStyle(current.Style, ev.Style) {
			if !merging {
				builder.WriteString(current.Text)
				merging = true
			}
			builder.WriteString(ev.Text)
			current.Span.End = ev.Span.End
			continue
		}
		flush()
		current = ev
	}
	flush()
	return out
}

func sameStyle(a, b InlineStyle) bool {
	return a.Emphasis == b.Emphasis && a.Strong == b.Strong && a.Strike == b.Strike && a.Code == b.Code && a.Link == b.Link && a.LinkTitle == b.LinkTitle
}

func isEscapablePunctuation(c byte) bool {
	return strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", rune(c))
}

func unescapeBackslashPunctuation(text string) string {
	if !strings.Contains(text, "\\") {
		return text
	}
	var out strings.Builder
	out.Grow(len(text))
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) && isEscapablePunctuation(text[i+1]) {
			out.WriteByte(text[i+1])
			i++
			continue
		}
		out.WriteByte(text[i])
	}
	return out.String()
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIAlphaNumeric(c byte) bool {
	return isASCIIAlpha(c) || (c >= '0' && c <= '9')
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

var _ = isAlphaNumeric

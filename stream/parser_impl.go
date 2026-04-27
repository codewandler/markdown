package stream

import (
	"bytes"
	"html"
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

	inBlockquote       bool
	inList             bool
	listData           ListData
	inListItem         bool
	inIndented         bool
	indentedBlankLines int
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
	p.closeParagraph(&events)
	p.closeIndentedCode(&events)
	if p.fence.open {
		p.fence.open = false
		events = append(events, Event{Kind: EventExitBlock, Block: BlockFencedCode})
	}
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
	if p.fence.open {
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
		return
	}

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

	if p.refDef.active {
		if p.continueLinkReferenceDefinition(line, events) {
			return
		}
	}

	if strings.TrimSpace(line.text) == "" {
		p.closeParagraph(events)
		p.closeIndentedCode(events)
		p.closeListItem(events)
		p.closeContainers(events)
		return
	}

	if len(p.paragraph.lines) == 0 && !p.inList && !p.inListItem {
		if p.startLinkReferenceDefinition(line) {
			return
		}
	}

	if content, ok := blockquoteContent(line.text); ok {
		p.ensureDocument(events)
		if !p.inBlockquote {
			p.closeParagraph(events)
			p.closeList(events)
			p.inBlockquote = true
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
			p.addParagraphLine(line)
			return
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
			p.listData = item.data
			data := p.listData
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockList, List: &data, Span: Span{Start: line.start, End: line.end}})
		}
		p.closeListItem(events)
		p.inListItem = true
		data := item.data
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockListItem, List: &data, Span: Span{Start: line.start, End: line.end}})
		if strings.TrimSpace(item.content) != "" {
			inner := line
			inner.text = item.content
			p.addParagraphLine(inner)
		}
		return
	}

	if p.inList {
		if strings.HasPrefix(line.text, "  ") || strings.HasPrefix(line.text, "\t") {
			inner := line
			inner.text = strings.TrimLeft(line.text, " \t")
			p.addParagraphLine(inner)
			return
		}
		p.closeParagraph(events)
		p.closeListItem(events)
		p.closeList(events)
	}

	p.processNonContainerLine(line, events)
}

func (p *parser) processNonContainerLine(line lineInfo, events *[]Event) {
	if marker, n, indent, info, ok := openingFence(line.text); ok {
		p.ensureDocument(events)
		p.closeParagraph(events)
		p.closeIndentedCode(events)
		p.fence = fenceState{open: true, marker: marker, length: n, indent: indent, info: info}
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockFencedCode, Info: info, Span: Span{Start: line.start, End: line.end}})
		return
	}

	if level, text, ok := heading(line.text); ok {
		p.ensureDocument(events)
		p.closeParagraph(events)
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
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockIndentedCode, Span: Span{Start: line.start, End: line.end}})
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
	*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockParagraph, Span: Span{Start: start, End: end}})
	var text strings.Builder
	for i, line := range p.paragraph.lines {
		if i > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(line.text)
	}
	*events = append(*events, p.parseInline(text.String(), Span{Start: start, End: end})...)
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockParagraph, Span: Span{Start: start, End: end}})
	clear(p.paragraph.lines)
	if cap(p.paragraph.lines) > 1024 {
		p.paragraph.lines = nil
	} else {
		p.paragraph.lines = p.paragraph.lines[:0]
	}
}

func (p *parser) closeSetextHeading(level int, underline Span, events *[]Event) {
	if len(p.paragraph.lines) == 0 {
		return
	}
	p.ensureDocument(events)
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
	p.closeIndentedCode(events)
	if p.inList {
		p.closeListItem(events)
		p.closeList(events)
	}
	if p.inBlockquote {
		p.closeBlockquote(lineInfo{}, events)
	}
}

func (p *parser) closeBlockquote(line lineInfo, events *[]Event) {
	if !p.inBlockquote {
		return
	}
	p.closeParagraph(events)
	p.inBlockquote = false
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockBlockquote, Span: Span{Start: line.start, End: line.start}})
}

func (p *parser) closeListItem(events *[]Event) {
	if !p.inListItem {
		return
	}
	p.closeParagraph(events)
	p.inListItem = false
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockListItem})
}

func (p *parser) closeList(events *[]Event) {
	if !p.inList {
		return
	}
	p.closeListItem(events)
	p.inList = false
	data := p.listData
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockList, List: &data})
	p.listData = ListData{}
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
	data    ListData
	content string
}

func listItem(line string) (listItemData, bool) {
	indent, indentBytes := leadingIndent(line)
	if indent > 3 {
		return listItemData{}, false
	}
	trimmed := line[indentBytes:]
	if len(trimmed) == 1 && strings.ContainsRune("-+*", rune(trimmed[0])) {
		return listItemData{
			data: ListData{Ordered: false, Marker: string(trimmed[0]), Tight: true},
		}, true
	}
	if len(trimmed) < 2 {
		return listItemData{}, false
	}
	if strings.ContainsRune("-+*", rune(trimmed[0])) && (trimmed[1] == ' ' || trimmed[1] == '\t') {
		return listItemData{
			data:    ListData{Ordered: false, Marker: string(trimmed[0]), Tight: true},
			content: strings.TrimLeft(trimmed[2:], " \t"),
		}, true
	}
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
	if i+1 == len(trimmed) {
		start, err := strconv.Atoi(trimmed[:i])
		if err != nil {
			return listItemData{}, false
		}
		return listItemData{
			data: ListData{Ordered: true, Start: start, Marker: string(marker), Tight: true},
		}, true
	}
	if trimmed[i+1] != ' ' && trimmed[i+1] != '\t' {
		return listItemData{}, false
	}
	start, err := strconv.Atoi(trimmed[:i])
	if err != nil {
		return listItemData{}, false
	}
	return listItemData{
		data:    ListData{Ordered: true, Start: start, Marker: string(marker), Tight: true},
		content: strings.TrimLeft(trimmed[i+2:], " \t"),
	}, true
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
	kind  inlineTokenKind
	text  string
	style InlineStyle
	delim byte
	run   int
	open  bool
	close bool
}

func tokenizeInline(text string, span Span, refs map[string]linkReference) []inlineToken {
	var tokens []inlineToken
	var prevSource string
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
		if text[0] == '[' && linkPossible {
			if ev, rest, ok := parseInlineLink(text, span); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
			linkPossible = strings.Contains(text[1:], "](")
		}
		if text[0] == '[' && len(refs) > 0 {
			if ev, rest, ok := parseReferenceLink(text, span, refs); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
		}
		if text[0] == '<' && autolinkPossible {
			if ev, rest, ok := parseAutolink(text, span); ok {
				tokens = append(tokens, inlineToken{kind: inlineTokenText, text: ev.Text, style: ev.Style})
				prevSource = text[:len(text)-len(rest)]
				text = rest
				continue
			}
			autolinkPossible = strings.Contains(text[1:], ">")
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
			open, close := emphasisDelimRun(prevSource, text[n:], text[0], n)
			tokens = append(tokens, inlineToken{kind: inlineTokenDelimiter, text: text[:n], delim: text[0], run: n, open: open, close: close})
			prevSource = text[:n]
			text = text[n:]
			continue
		}
		next := nextInlineDelimiter(text)
		if next <= 0 {
			next = 1
		}
		tokens = append(tokens, inlineToken{kind: inlineTokenText, text: text[:next]})
		prevSource = text[:next]
		text = text[next:]
	}
	return tokens
}

func resolveEmphasis(tokens []inlineToken) []inlineToken {
	if len(tokens) == 0 {
		return tokens
	}
	pairs := make(map[int]int)
	var stack []int
	for i := range tokens {
		tok := tokens[i]
		if tok.kind != inlineTokenDelimiter {
			continue
		}
		if tok.close {
			for j := len(stack) - 1; j >= 0; j-- {
				openIdx := stack[j]
				openTok := tokens[openIdx]
				if openTok.delim != tok.delim || !openTok.open {
					continue
				}
				pairs[openIdx] = i
				pairs[i] = openIdx
				stack = append(stack[:j], stack[j+1:]...)
				break
			}
		}
		if tok.open {
			stack = append(stack, i)
		}
	}

	var out []inlineToken
	var emitRange func(start, end int, style InlineStyle)
	emitText := func(text string, style InlineStyle) {
		if text == "" {
			return
		}
		out = append(out, inlineToken{kind: inlineTokenText, text: text, style: style})
	}
	emitRange = func(start, end int, style InlineStyle) {
		for i := start; i < end; {
			tok := tokens[i]
			if tok.kind == inlineTokenDelimiter {
				if closeIdx, ok := pairs[i]; ok && closeIdx > i && closeIdx < end {
					openCount := tok.run
					closeCount := tokens[closeIdx].run
					use := openCount
					if closeCount < use {
						use = closeCount
					}
					emitText(tok.text[:openCount-use], style)
					emitRange(i+1, closeIdx, applyDelimiterStyle(style, use))
					emitText(tokens[closeIdx].text[use:], style)
					i = closeIdx + 1
					continue
				}
				emitText(tok.text, style)
				i++
				continue
			}
			switch tok.kind {
			case inlineTokenText:
				out = append(out, inlineToken{kind: inlineTokenText, text: tok.text, style: mergeInlineStyles(style, tok.style)})
			case inlineTokenSoftBreak:
				out = append(out, inlineToken{kind: inlineTokenSoftBreak})
			case inlineTokenLineBreak:
				out = append(out, inlineToken{kind: inlineTokenLineBreak})
			}
			i++
		}
	}
	emitRange(0, len(tokens), InlineStyle{})
	return out
}

func applyDelimiterStyle(style InlineStyle, count int) InlineStyle {
	out := style
	for count >= 2 {
		out.Strong = true
		count -= 2
	}
	if count == 1 {
		out.Emphasis = true
	}
	return out
}

func mergeInlineStyles(base, add InlineStyle) InlineStyle {
	out := base
	out.Emphasis = out.Emphasis || add.Emphasis
	out.Strong = out.Strong || add.Strong
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

func parseInlineLink(text string, span Span) (Event, string, bool) {
	closeText := matchingLinkLabelEnd(text)
	if closeText < 0 || closeText+1 >= len(text) || text[closeText+1] != '(' {
		return Event{}, text, false
	}
	label := decodeCharacterReferences(unescapeBackslashPunctuation(text[1:closeText]))
	dest, title, end, ok := parseInlineLinkTail(text[closeText+2:])
	if !ok {
		return Event{}, text, false
	}
	return Event{Kind: EventText, Text: label, Style: InlineStyle{Link: dest, LinkTitle: title}, Span: span}, text[closeText+2+end:], true
}

func parseReferenceLink(text string, span Span, refs map[string]linkReference) (Event, string, bool) {
	closeLabel := matchingLinkLabelEnd(text)
	if closeLabel <= 0 {
		return Event{}, text, false
	}
	labelText := decodeCharacterReferences(unescapeBackslashPunctuation(text[1:closeLabel]))
	refLabel := labelText
	end := closeLabel + 1
	if end < len(text) && text[end] == '[' {
		closeRef := matchingLinkLabelEnd(text[end:])
		if closeRef < 0 {
			return Event{}, text, false
		}
		if closeRef > 1 {
			refLabel = decodeCharacterReferences(unescapeBackslashPunctuation(text[end+1 : end+closeRef]))
		}
		end += closeRef + 1
	}
	ref, ok := refs[normalizeReferenceLabel(refLabel)]
	if !ok {
		return Event{}, text, false
	}
	return Event{Kind: EventText, Text: labelText, Style: InlineStyle{Link: ref.dest, LinkTitle: ref.title}, Span: span}, text[end:], true
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
	closeLabel := matchingLinkLabelEnd(text)
	if closeLabel <= 0 || closeLabel+1 >= len(text) || text[closeLabel+1] != ':' {
		return "", "", 0, false
	}
	label := normalizeReferenceLabel(decodeCharacterReferences(unescapeBackslashPunctuation(text[1:closeLabel])))
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
	i = skipMarkdownSpace(text, next)
	if i < len(text) {
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
	return strings.ToLower(strings.Join(fields, " "))
}

func matchingLinkLabelEnd(text string) int {
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
		switch c {
		case '[':
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
	if marker == '*' {
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
	labels := strings.Split(domain, ".")
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
	if i := strings.IndexAny(text, "\n\\*_`[<&"); i >= 0 {
		return i
	}
	return len(text)
}

func isInlineDelimiterByte(c byte) bool {
	return c == '\n' || c == '\\' || c == '*' || c == '_' || c == '`' || c == '[' || c == '<' || c == '&'
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
	return a.Emphasis == b.Emphasis && a.Strong == b.Strong && a.Code == b.Code && a.Link == b.Link && a.LinkTitle == b.LinkTitle
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

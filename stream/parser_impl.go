package stream

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"
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

	inBlockquote bool
	inList       bool
	listData     ListData
	inListItem   bool
	inIndented   bool
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
		p.closeIndentedCode(events)
	}

	if strings.TrimSpace(line.text) == "" {
		p.closeParagraph(events)
		p.closeIndentedCode(events)
		p.closeListItem(events)
		p.closeContainers(events)
		return
	}

	if content, ok := blockquoteContent(line.text); ok {
		p.ensureDocument(events)
		if !p.inBlockquote {
			p.closeParagraph(events)
			p.closeList(events)
			p.inBlockquote = true
			*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockBlockquote, Span: Span{Start: line.start, End: line.end}})
		}
		inner := line
		inner.text = content
		p.processNonContainerLine(inner, events)
		return
	}

	if p.inBlockquote {
		p.closeParagraph(events)
		p.inBlockquote = false
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockBlockquote, Span: Span{Start: line.start, End: line.start}})
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
		p.closeParagraph(events)
		if !p.inList || p.listData.Ordered != item.data.Ordered {
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
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockHeading, Level: level, Span: Span{Start: line.start, End: line.end}})
		*events = append(*events, parseInline(text, Span{Start: line.start, End: line.end})...)
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockHeading, Level: level, Span: Span{Start: line.start, End: line.end}})
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
		if indent := leadingSpaces(text); indent <= 3 {
			text = text[indent:]
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
	*events = append(*events, parseInline(text.String(), Span{Start: start, End: end})...)
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockParagraph, Span: Span{Start: start, End: end}})
	clear(p.paragraph.lines)
	if cap(p.paragraph.lines) > 1024 {
		p.paragraph.lines = nil
	} else {
		p.paragraph.lines = p.paragraph.lines[:0]
	}
}

func (p *parser) emitThematicBreak(line lineInfo, events *[]Event) {
	*events = append(*events,
		Event{Kind: EventEnterBlock, Block: BlockThematicBreak, Span: Span{Start: line.start, End: line.end}},
		Event{Kind: EventExitBlock, Block: BlockThematicBreak, Span: Span{Start: line.start, End: line.end}},
	)
}

func (p *parser) emitIndentedCodeLine(line lineInfo, events *[]Event) {
	text := strings.TrimPrefix(line.text, "    ")
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
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockIndentedCode})
}

func (p *parser) closeContainers(events *[]Event) {
	p.closeIndentedCode(events)
	if p.inList {
		p.closeListItem(events)
		p.closeList(events)
	}
	if p.inBlockquote {
		p.inBlockquote = false
		*events = append(*events, Event{Kind: EventExitBlock, Block: BlockBlockquote})
	}
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
	indent := leadingSpaces(line)
	if indent > 3 {
		return false
	}
	trimmed := line[indent:]
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
	indent := leadingSpaces(line)
	if indent > 3 {
		return 0, 0, 0, "", false
	}
	trimmed := line[indent:]
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
	info := strings.TrimSpace(trimmed[n:])
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
	indent := leadingSpaces(line)
	if indent > 3 {
		return 0, "", false
	}
	trimmed := line[indent:]
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
	if leadingSpaces(line) > 3 {
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

func indentedCode(line string) bool {
	return strings.HasPrefix(line, "    ") && strings.TrimSpace(line) != ""
}

func blockquoteContent(line string) (string, bool) {
	indent := leadingSpaces(line)
	if indent > 3 || indent >= len(line) || line[indent] != '>' {
		return "", false
	}
	content := line[indent+1:]
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
	indent := leadingSpaces(line)
	if indent > 3 {
		return listItemData{}, false
	}
	trimmed := line[indent:]
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
	if i == 0 || i > 9 || i+1 >= len(trimmed) {
		return listItemData{}, false
	}
	marker := trimmed[i]
	if marker != '.' && marker != ')' {
		return listItemData{}, false
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

func leadingSpaces(line string) int {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}
	return n
}

func parseInline(text string, span Span) []Event {
	if text == "" {
		return []Event{{Kind: EventText, Text: "", Span: span}}
	}
	var events []Event
	linkPossible := strings.Contains(text, "](")
	autolinkPossible := strings.Contains(text, ">")
	for len(text) > 0 {
		if text[0] == '\n' {
			if trimHardBreakMarker(events) {
				events = append(events, Event{Kind: EventLineBreak, Span: span})
			} else {
				events = append(events, Event{Kind: EventSoftBreak, Span: span})
			}
			text = text[1:]
			continue
		}
		if text[0] == '\\' && len(text) > 1 && isEscapablePunctuation(text[1]) {
			events = append(events, Event{Kind: EventText, Text: text[1:2], Span: span})
			text = text[2:]
			continue
		}
		if strings.HasPrefix(text, "**") {
			if end := strings.Index(text[2:], "**"); end >= 0 {
				content := text[2 : 2+end]
				events = append(events, Event{Kind: EventText, Text: content, Style: InlineStyle{Strong: true}, Span: span})
				text = text[2+end+2:]
				continue
			}
		}
		if strings.HasPrefix(text, "__") {
			if end := strings.Index(text[2:], "__"); end >= 0 {
				content := text[2 : 2+end]
				events = append(events, Event{Kind: EventText, Text: content, Style: InlineStyle{Strong: true}, Span: span})
				text = text[2+end+2:]
				continue
			}
		}
		if strings.HasPrefix(text, "*") {
			if end := strings.Index(text[1:], "*"); end >= 0 {
				content := text[1 : 1+end]
				if content != "" {
					events = append(events, Event{Kind: EventText, Text: content, Style: InlineStyle{Emphasis: true}, Span: span})
					text = text[1+end+1:]
					continue
				}
			}
		}
		if strings.HasPrefix(text, "_") {
			if end := strings.Index(text[1:], "_"); end >= 0 {
				content := text[1 : 1+end]
				if content != "" {
					events = append(events, Event{Kind: EventText, Text: content, Style: InlineStyle{Emphasis: true}, Span: span})
					text = text[1+end+1:]
					continue
				}
			}
		}
		if text[0] == '`' {
			if ev, rest, ok := parseCodeSpan(text, span); ok {
				events = append(events, ev)
				text = rest
				continue
			}
		}
		if text[0] == '[' && linkPossible {
			if ev, rest, ok := parseInlineLink(text, span); ok {
				events = append(events, ev)
				text = rest
				continue
			}
			linkPossible = strings.Contains(text[1:], "](")
		}
		if text[0] == '<' && autolinkPossible {
			if ev, rest, ok := parseAutolink(text, span); ok {
				events = append(events, ev)
				text = rest
				continue
			}
			autolinkPossible = strings.Contains(text[1:], ">")
		}
		next := nextInlineDelimiter(text)
		if next <= 0 {
			next = 1
		}
		events = append(events, Event{Kind: EventText, Text: text[:next], Span: span})
		text = text[next:]
	}
	return coalesceText(events)
}

func parseInlineLink(text string, span Span) (Event, string, bool) {
	closeText := strings.Index(text, "](")
	if closeText <= 0 {
		return Event{}, text, false
	}
	closeURL := strings.IndexByte(text[closeText+2:], ')')
	if closeURL < 0 {
		return Event{}, text, false
	}
	label := text[1:closeText]
	url := text[closeText+2 : closeText+2+closeURL]
	if label == "" || strings.TrimSpace(url) == "" {
		return Event{}, text, false
	}
	return Event{Kind: EventText, Text: label, Style: InlineStyle{Link: url}, Span: span}, text[closeText+2+closeURL+1:], true
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
	next := len(text)
	for _, d := range []string{"\n", "\\", "**", "__", "*", "_", "`", "[", "<"} {
		if i := strings.Index(text, d); i >= 0 && i < next {
			next = i
		}
	}
	return next
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
	return a.Emphasis == b.Emphasis && a.Strong == b.Strong && a.Code == b.Code && a.Link == b.Link
}

func isEscapablePunctuation(c byte) bool {
	return strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", rune(c))
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

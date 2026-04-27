package stream

import (
	"strings"
)

// NewParser returns the current incremental parser prototype.
//
// The implementation is intentionally small while the event contract is being
// exercised. It handles headings, paragraphs, and fenced code blocks without
// reparsing the accumulated document.
func NewParser() Parser {
	return &lineParser{}
}

type lineParser struct {
	started bool
	tail    string

	paragraph []string

	inFence  bool
	fence    byte
	fenceLen int
}

func (p *lineParser) Write(chunk []byte) ([]Event, error) {
	if len(chunk) == 0 {
		return nil, nil
	}

	var events []Event
	p.startDocument(&events)

	p.tail += string(chunk)
	for {
		i := strings.IndexByte(p.tail, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSuffix(p.tail[:i], "\r")
		p.tail = p.tail[i+1:]
		p.processLine(line, &events)
	}

	return events, nil
}

func (p *lineParser) Flush() ([]Event, error) {
	var events []Event
	p.startDocument(&events)

	if p.tail != "" {
		p.processLine(strings.TrimSuffix(p.tail, "\r"), &events)
		p.tail = ""
	}
	p.closeParagraph(&events)
	if p.inFence {
		p.inFence = false
		events = append(events, Event{Kind: EventExitBlock, Block: BlockFencedCode})
	}
	if p.started {
		events = append(events, Event{Kind: EventExitBlock, Block: BlockDocument})
		p.started = false
	}
	return events, nil
}

func (p *lineParser) Reset() {
	*p = lineParser{}
}

func (p *lineParser) startDocument(events *[]Event) {
	if p.started {
		return
	}
	p.started = true
	*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockDocument})
}

func (p *lineParser) processLine(line string, events *[]Event) {
	if p.inFence {
		if p.isClosingFence(line) {
			p.inFence = false
			*events = append(*events, Event{Kind: EventExitBlock, Block: BlockFencedCode})
			return
		}
		*events = append(*events,
			Event{Kind: EventText, Text: line},
			Event{Kind: EventLineBreak},
		)
		return
	}

	if strings.TrimSpace(line) == "" {
		p.closeParagraph(events)
		return
	}

	if marker, n, info, ok := openingFence(line); ok {
		p.closeParagraph(events)
		p.inFence = true
		p.fence = marker
		p.fenceLen = n
		*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockFencedCode, Info: info})
		return
	}

	if level, text, ok := heading(line); ok {
		p.closeParagraph(events)
		*events = append(*events,
			Event{Kind: EventEnterBlock, Block: BlockHeading, Level: level},
			Event{Kind: EventText, Text: text},
			Event{Kind: EventExitBlock, Block: BlockHeading, Level: level},
		)
		return
	}

	p.paragraph = append(p.paragraph, line)
}

func (p *lineParser) closeParagraph(events *[]Event) {
	if len(p.paragraph) == 0 {
		return
	}

	*events = append(*events, Event{Kind: EventEnterBlock, Block: BlockParagraph})
	for i, line := range p.paragraph {
		if i > 0 {
			*events = append(*events, Event{Kind: EventSoftBreak})
		}
		*events = append(*events, Event{Kind: EventText, Text: strings.TrimSpace(line)})
	}
	*events = append(*events, Event{Kind: EventExitBlock, Block: BlockParagraph})
	p.paragraph = p.paragraph[:0]
}

func (p *lineParser) isClosingFence(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < p.fenceLen {
		return false
	}
	for i := 0; i < p.fenceLen; i++ {
		if trimmed[i] != p.fence {
			return false
		}
	}
	return strings.TrimSpace(trimmed[p.fenceLen:]) == ""
}

func openingFence(line string) (byte, int, string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) < 3 {
		return 0, 0, "", false
	}
	marker := trimmed[0]
	if marker != '`' && marker != '~' {
		return 0, 0, "", false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == marker {
		n++
	}
	if n < 3 {
		return 0, 0, "", false
	}
	return marker, n, strings.TrimSpace(trimmed[n:]), true
}

func heading(line string) (int, string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
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
	text = strings.TrimRight(text, "#")
	return level, strings.TrimSpace(text), true
}

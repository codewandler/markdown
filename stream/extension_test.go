package stream

import (
	"strings"
	"testing"
)

type testInlineScanner struct{}

func (testInlineScanner) TriggerBytes() string { return ":" }

func (testInlineScanner) ScanInline(input string, _ InlineContext) (InlineScanResult, bool) {
	if !strings.HasPrefix(input, ":ok:") {
		return InlineScanResult{}, false
	}
	return InlineScanResult{
		Consume: len(":ok:"),
		Event: Event{
			Kind: EventInline,
			Inline: &InlineData{
				Type:         "test",
				Source:       ":ok:",
				Text:         "OK",
				DisplayWidth: 2,
			},
		},
	}, true
}

func TestInlineScannerEmitsInlineEvent(t *testing.T) {
	p := NewParser(WithInlineScanner(testInlineScanner{}))
	events, err := p.Write([]byte("hello :ok:\n"))
	if err != nil {
		t.Fatal(err)
	}
	flush, err := p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	events = append(events, flush...)

	var found bool
	for _, ev := range events {
		if ev.Kind == EventInline && ev.Inline != nil && ev.Inline.Type == "test" {
			found = true
			if ev.Inline.Text != "OK" || ev.Inline.DisplayWidth != 2 {
				t.Fatalf("unexpected inline data: %#v", ev.Inline)
			}
		}
	}
	if !found {
		t.Fatalf("missing inline event in %#v", events)
	}
}

func TestInlineScannerRespectsCodeSpanAndEmphasis(t *testing.T) {
	p := NewParser(WithInlineScanner(testInlineScanner{}))
	events, err := p.Write([]byte("`:ok:` **:ok:**\n"))
	if err != nil {
		t.Fatal(err)
	}
	flush, err := p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	events = append(events, flush...)

	inlineCount := 0
	var strongInline bool
	var codeLiteral bool
	for _, ev := range events {
		if ev.Kind == EventText && ev.Text == ":ok:" && ev.Style.Code {
			codeLiteral = true
		}
		if ev.Kind == EventInline {
			inlineCount++
			if ev.Style.Strong {
				strongInline = true
			}
		}
	}
	if !codeLiteral {
		t.Fatalf("code span shortcode was not preserved as code text: %#v", events)
	}
	if inlineCount != 1 || !strongInline {
		t.Fatalf("want one strong inline event, got count=%d strong=%v events=%#v", inlineCount, strongInline, events)
	}
}

func TestInlineScannerWorksInChunkedTable(t *testing.T) {
	p := NewParser(WithInlineScanner(testInlineScanner{}))
	if _, err := p.Write([]byte("| status |\n| --- |\n| :")); err != nil {
		t.Fatal(err)
	}
	events, err := p.Write([]byte("ok: |\n"))
	if err != nil {
		t.Fatal(err)
	}
	flush, err := p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	events = append(events, flush...)

	var inCell bool
	var found bool
	for _, ev := range events {
		if ev.Kind == EventEnterBlock && ev.Block == BlockTableCell {
			inCell = true
		}
		if ev.Kind == EventExitBlock && ev.Block == BlockTableCell {
			inCell = false
		}
		if inCell && ev.Kind == EventInline && ev.Inline != nil && ev.Inline.Type == "test" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing table inline event in %#v", events)
	}
}

package stream

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/codewandler/markdown/internal/commonmarktests"
)

func TestParserEventInvariants(t *testing.T) {
	samples := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "heading", in: "# Heading\n"},
		{name: "paragraph", in: "alpha\nbeta\n\n"},
		{name: "fenced code", in: "```go\npackage main\n```\n"},
		{name: "unfinished fenced code", in: "```go\npackage main\n"},
		{name: "list", in: "- one\n- two\n\n"},
		{name: "blockquote", in: "> quote\n\n"},
		{name: "table", in: "| a | b |\n| --- | :---: |\n| one | two |\n"},
		{name: "thematic break", in: "---\n"},
	}
	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			assertEventInvariants(t, parseAll(t, sample.in))
		})
	}
}

func TestCommonMarkCorpusEventInvariants(t *testing.T) {
	examples, err := commonmarktests.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range examples {
		t.Run(fmt.Sprintf("%03d/%s", ex.Example, ex.Section), func(t *testing.T) {
			assertEventInvariants(t, parseAll(t, ex.Markdown))
		})
	}
}

func TestParserNoEventsAfterFlush(t *testing.T) {
	p := NewParser()
	if _, err := p.Write([]byte("# heading\n")); err != nil {
		t.Fatal(err)
	}
	if events, err := p.Flush(); err != nil {
		t.Fatal(err)
	} else if len(events) == 0 {
		t.Fatal("expected flush to close the document")
	}
	if events, err := p.Write([]byte("after\n")); err != nil {
		t.Fatal(err)
	} else if len(events) != 0 {
		t.Fatalf("expected no events after flush write, got %#v", events)
	}
	if events, err := p.Flush(); err != nil {
		t.Fatal(err)
	} else if len(events) != 0 {
		t.Fatalf("expected no events after second flush, got %#v", events)
	}
}

func TestParserResetReturnsToCleanBehavior(t *testing.T) {
	p := NewParser()
	if _, err := p.Write([]byte("```go\npackage main\n")); err != nil {
		t.Fatal(err)
	}
	p.Reset()

	var got []Event
	events, err := p.Write([]byte("# reset\n"))
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, events...)
	events, err = p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, events...)

	want := viewEvents(parseAll(t, "# reset\n"))
	if gotView := viewEvents(got); !equalEventViews(gotView, want) {
		t.Fatalf("reset parse mismatch\nwant: %#v\n got: %#v", want, gotView)
	}
}

func assertEventInvariants(t *testing.T, events []Event) {
	t.Helper()
	if len(events) == 0 {
		return
	}
	if events[0].Kind != EventEnterBlock || events[0].Block != BlockDocument {
		t.Fatalf("first event must enter document, got %#v", events[0])
	}
	if events[len(events)-1].Kind != EventExitBlock || events[len(events)-1].Block != BlockDocument {
		t.Fatalf("last event must exit document, got %#v", events[len(events)-1])
	}

	var stack []BlockKind
	for i, ev := range events {
		switch ev.Kind {
		case EventEnterBlock:
			stack = append(stack, ev.Block)
		case EventExitBlock:
			if len(stack) == 0 {
				t.Fatalf("event %d exits %s with empty stack", i, ev.Block)
			}
			top := stack[len(stack)-1]
			if top != ev.Block {
				t.Fatalf("event %d exits %s while %s is open", i, ev.Block, top)
			}
			stack = stack[:len(stack)-1]
		case EventText, EventSoftBreak, EventLineBreak:
			if len(stack) == 0 {
				t.Fatalf("event %d appears outside document: %#v", i, ev)
			}
		default:
			t.Fatalf("event %d has unknown kind %q", i, ev.Kind)
		}
	}
	if len(stack) != 0 {
		t.Fatalf("unclosed block stack: %#v", stack)
	}
}

func equalEventViews(a, b []eventView) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			if a[i].List == nil || b[i].List == nil || a[i].Table == nil || b[i].Table == nil {
				return false
			}
			av, bv := *a[i].List, *b[i].List
			at, bt := *a[i].Table, *b[i].Table
			aa, bb := a[i], b[i]
			aa.List, bb.List = nil, nil
			aa.Table, bb.Table = nil, nil
			if aa != bb || av != bv || !reflect.DeepEqual(at, bt) {
				return false
			}
		}
	}
	return true
}

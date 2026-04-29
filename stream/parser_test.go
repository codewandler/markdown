package stream

import (
	"reflect"
	"strings"
	"testing"
)

type eventView struct {
	Kind  EventKind
	Block BlockKind
	Text  string
	Style InlineStyle
	Level int
	Info  string
	List  *ListData
	Table *TableData
}

func TestParserBlocks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []eventView
	}{
		{
			name: "heading",
			in:   "# Hello *world*\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockHeading, Level: 1},
				{Kind: EventText, Text: "Hello "},
				{Kind: EventText, Text: "world", Style: InlineStyle{Emphasis: true}},
				{Kind: EventExitBlock, Block: BlockHeading, Level: 1},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "paragraph",
			in:   "alpha\nbeta\n\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockParagraph},
				{Kind: EventText, Text: "alpha"},
				{Kind: EventSoftBreak},
				{Kind: EventText, Text: "beta"},
				{Kind: EventExitBlock, Block: BlockParagraph},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "fenced code streams",
			in:   "```go\npackage main\n```\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockFencedCode, Info: "go"},
				{Kind: EventText, Text: "package main"},
				{Kind: EventLineBreak},
				{Kind: EventExitBlock, Block: BlockFencedCode},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "unfinished fenced code flush",
			in:   "```go\npackage main\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockFencedCode, Info: "go"},
				{Kind: EventText, Text: "package main"},
				{Kind: EventLineBreak},
				{Kind: EventExitBlock, Block: BlockFencedCode},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "thematic break",
			in:   "alpha\n\n---\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockParagraph},
				{Kind: EventText, Text: "alpha"},
				{Kind: EventExitBlock, Block: BlockParagraph},
				{Kind: EventEnterBlock, Block: BlockThematicBreak},
				{Kind: EventExitBlock, Block: BlockThematicBreak},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "unordered list",
			in:   "- one\n- two\n\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockList, List: &ListData{Marker: "-", Tight: true}},
				{Kind: EventEnterBlock, Block: BlockListItem, List: &ListData{Marker: "-", Tight: true}},
				{Kind: EventEnterBlock, Block: BlockParagraph},
				{Kind: EventText, Text: "one"},
				{Kind: EventExitBlock, Block: BlockParagraph},
				{Kind: EventExitBlock, Block: BlockListItem},
				{Kind: EventEnterBlock, Block: BlockListItem, List: &ListData{Marker: "-", Tight: true}},
				{Kind: EventEnterBlock, Block: BlockParagraph},
				{Kind: EventText, Text: "two"},
				{Kind: EventExitBlock, Block: BlockParagraph},
				{Kind: EventExitBlock, Block: BlockListItem},
				{Kind: EventExitBlock, Block: BlockList, List: &ListData{Marker: "-", Tight: true}},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "blockquote",
			in:   "> quote\n\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockBlockquote},
				{Kind: EventEnterBlock, Block: BlockParagraph},
				{Kind: EventText, Text: "quote"},
				{Kind: EventExitBlock, Block: BlockParagraph},
				{Kind: EventExitBlock, Block: BlockBlockquote},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
		{
			name: "table",
			in:   "| a | b | c |\n|:---|:---:|---:|\n| 1 | 2 | 3 |\n",
			want: []eventView{
				{Kind: EventEnterBlock, Block: BlockDocument},
				{Kind: EventEnterBlock, Block: BlockTable, Table: &TableData{Align: []TableAlign{TableAlignLeft, TableAlignCenter, TableAlignRight}}},
				{Kind: EventEnterBlock, Block: BlockTableRow},
				{Kind: EventEnterBlock, Block: BlockTableCell},
				{Kind: EventText, Text: "a"},
				{Kind: EventExitBlock, Block: BlockTableCell},
				{Kind: EventEnterBlock, Block: BlockTableCell},
				{Kind: EventText, Text: "b"},
				{Kind: EventExitBlock, Block: BlockTableCell},
				{Kind: EventEnterBlock, Block: BlockTableCell},
				{Kind: EventText, Text: "c"},
				{Kind: EventExitBlock, Block: BlockTableCell},
				{Kind: EventExitBlock, Block: BlockTableRow},
				{Kind: EventEnterBlock, Block: BlockTableRow},
				{Kind: EventEnterBlock, Block: BlockTableCell},
				{Kind: EventText, Text: "1"},
				{Kind: EventExitBlock, Block: BlockTableCell},
				{Kind: EventEnterBlock, Block: BlockTableCell},
				{Kind: EventText, Text: "2"},
				{Kind: EventExitBlock, Block: BlockTableCell},
				{Kind: EventEnterBlock, Block: BlockTableCell},
				{Kind: EventText, Text: "3"},
				{Kind: EventExitBlock, Block: BlockTableCell},
				{Kind: EventExitBlock, Block: BlockTableRow},
				{Kind: EventExitBlock, Block: BlockTable, Table: &TableData{Align: []TableAlign{TableAlignLeft, TableAlignCenter, TableAlignRight}}},
				{Kind: EventExitBlock, Block: BlockDocument},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAll(t, tt.in)
			if !reflect.DeepEqual(viewEvents(got), tt.want) {
				t.Fatalf("events mismatch\nwant: %#v\n got: %#v", tt.want, viewEvents(got))
			}
		})
	}
}

func TestParserResponsiveness(t *testing.T) {
	p := NewParser()
	events, err := p.Write([]byte("```go\npackage main\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := viewEvents(events)
	want := []eventView{
		{Kind: EventEnterBlock, Block: BlockDocument},
		{Kind: EventEnterBlock, Block: BlockFencedCode, Info: "go"},
		{Kind: EventText, Text: "package main"},
		{Kind: EventLineBreak},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("responsiveness mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestParserHoldsIncompleteLine(t *testing.T) {
	p := NewParser()
	events, err := p.Write([]byte("# held"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events for incomplete line, got %#v", events)
	}
	events, err = p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	if got := viewEvents(events); len(got) == 0 || got[0].Kind != EventEnterBlock {
		t.Fatalf("expected flush events, got %#v", got)
	}
}

func TestInlineSubset(t *testing.T) {
	events := parseAll(t, "A **strong** and `code` plus [link](https://example.com)\n")
	got := viewEvents(events)
	assertContains(t, got, eventView{Kind: EventText, Text: "strong", Style: InlineStyle{Strong: true}})
	assertContains(t, got, eventView{Kind: EventText, Text: "code", Style: InlineStyle{Code: true}})
	assertContains(t, got, eventView{Kind: EventText, Text: "link", Style: InlineStyle{HasLink: true, Link: "https://example.com"}})
}

func TestTaskListSubset(t *testing.T) {
	events := viewEvents(parseAll(t, "- [ ] todo\n- [x] done\n1. [X] ordered\n"))
	assertContains(t, events, eventView{Kind: EventEnterBlock, Block: BlockListItem, List: &ListData{Marker: "-", Tight: true, Task: true, Checked: false}})
	assertContains(t, events, eventView{Kind: EventEnterBlock, Block: BlockListItem, List: &ListData{Marker: "-", Tight: true, Task: true, Checked: true}})
	assertContains(t, events, eventView{Kind: EventEnterBlock, Block: BlockListItem, List: &ListData{Ordered: true, Start: 1, Marker: ".", Tight: true, Task: true, Checked: true}})
	assertContains(t, events, eventView{Kind: EventText, Text: "todo"})
	assertContains(t, events, eventView{Kind: EventText, Text: "done"})
	assertContains(t, events, eventView{Kind: EventText, Text: "ordered"})
}

func TestGFMInlineExtensions(t *testing.T) {
	events := viewEvents(parseAll(t, "~~gone~~ https://example.com, www.example.org/path and foo@bar.example.com.\n"))
	assertContains(t, events, eventView{Kind: EventText, Text: "gone", Style: InlineStyle{Strike: true}})
	assertContains(t, events, eventView{Kind: EventText, Text: "https://example.com", Style: InlineStyle{HasLink: true, Link: "https://example.com"}})
	assertContains(t, events, eventView{Kind: EventText, Text: "www.example.org/path", Style: InlineStyle{HasLink: true, Link: "http://www.example.org/path"}})
	assertContains(t, events, eventView{Kind: EventText, Text: "foo@bar.example.com", Style: InlineStyle{HasLink: true, Link: "mailto:foo@bar.example.com"}})
}

func TestImageSubset(t *testing.T) {
	inline := parseAll(t, "![foo](https://example.com/logo.png)\n")
	got := viewEvents(inline)
	assertContains(t, got, eventView{Kind: EventText, Text: "foo", Style: InlineStyle{Link: "https://example.com/logo.png", HasLink: true, Image: true}})
	ref := parseAll(t, "[bar]: /bar.png\n\n![bar]\n")
	got = viewEvents(ref)
	assertContains(t, got, eventView{Kind: EventText, Text: "bar", Style: InlineStyle{Link: "/bar.png", HasLink: true, Image: true}})
}

func TestEscapedImageStartsAsText(t *testing.T) {
	events := parseAll(t, "\\![foo]\n")
	got := viewEvents(events)
	assertContains(t, got, eventView{Kind: EventText, Text: "![foo]"})
}

func TestEmphasisSubset(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		check func(*testing.T, []eventView)
	}{
		{
			name: "single star",
			in:   "*foo*\n",
			check: func(t *testing.T, events []eventView) {
				assertContains(t, events, eventView{Kind: EventText, Text: "foo", Style: InlineStyle{Emphasis: true}})
			},
		},
		{
			name: "single underscore",
			in:   "_foo bar_\n",
			check: func(t *testing.T, events []eventView) {
				assertContains(t, events, eventView{Kind: EventText, Text: "foo bar", Style: InlineStyle{Emphasis: true}})
			},
		},
		{
			name: "double star",
			in:   "**foo bar**\n",
			check: func(t *testing.T, events []eventView) {
				assertContains(t, events, eventView{Kind: EventText, Text: "foo bar", Style: InlineStyle{Strong: true}})
			},
		},
		{
			name: "double underscore",
			in:   "__foo bar__\n",
			check: func(t *testing.T, events []eventView) {
				assertContains(t, events, eventView{Kind: EventText, Text: "foo bar", Style: InlineStyle{Strong: true}})
			},
		},
		{
			name: "nested triple stars",
			in:   "***foo***\n",
			check: func(t *testing.T, events []eventView) {
				assertContains(t, events, eventView{Kind: EventText, Text: "foo", Style: InlineStyle{Emphasis: true, Strong: true}})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, viewEvents(parseAll(t, tt.in)))
		})
	}
}

func TestLinkReferenceDefinitions(t *testing.T) {
	t.Run("multiline definition resolves later reference", func(t *testing.T) {
		events := viewEvents(parseAll(t, "[foo]:\n/url\n  \"title\"\n\n[foo]\n"))
		assertContains(t, events, eventView{Kind: EventText, Text: "foo", Style: InlineStyle{HasLink: true, Link: "/url", LinkTitle: "title"}})
	})

	t.Run("invalid pending definition falls back to paragraph text", func(t *testing.T) {
		events := viewEvents(parseAll(t, "[foo]:\n\n[foo]\n"))
		assertContains(t, events, eventView{Kind: EventText, Text: "[foo]:"})
		assertContains(t, events, eventView{Kind: EventText, Text: "[foo]"})
	})

	t.Run("definition at flush emits no document events", func(t *testing.T) {
		if got := viewEvents(parseAll(t, "[foo]:\n/url")); len(got) != 0 {
			t.Fatalf("expected definition-only document to emit no events, got %#v", got)
		}
	})

	t.Run("next non-title line resolves after pending destination", func(t *testing.T) {
		events := viewEvents(parseAll(t, "[foo]: /url\n[foo]\n"))
		assertContains(t, events, eventView{Kind: EventText, Text: "foo", Style: InlineStyle{HasLink: true, Link: "/url"}})
	})
}

func parseAll(t *testing.T, in string) []Event {
	t.Helper()
	p := NewParser()
	var all []Event
	events, err := p.Write([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	all = append(all, events...)
	events, err = p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	all = append(all, events...)
	return all
}

func viewEvents(events []Event) []eventView {
	out := make([]eventView, 0, len(events))
	for _, ev := range events {
		var data *ListData
		if ev.List != nil {
			cp := *ev.List
			data = &cp
		}
		var table *TableData
		if ev.Table != nil {
			cp := *ev.Table
			if cp.Align != nil {
				cp.Align = append([]TableAlign(nil), cp.Align...)
			}
			table = &cp
		}
		out = append(out, eventView{
			Kind:  ev.Kind,
			Block: ev.Block,
			Text:  ev.Text,
			Style: ev.Style,
			Level: ev.Level,
			Info:  ev.Info,
			List:  data,
			Table: table,
		})
	}
	return out
}

func assertContains(t *testing.T, events []eventView, want eventView) {
	t.Helper()
	for _, event := range events {
		if reflect.DeepEqual(event, want) {
			return
		}
	}
	t.Fatalf("missing event %#v in %#v", want, events)
}

func TestNoPanicOnMalformedInline(t *testing.T) {
	for _, sample := range []string{
		"unterminated **strong\n",
		"unterminated `code\n",
		"[missing](\n",
		"<not-autolink>\n",
	} {
		t.Run(strings.TrimSpace(sample), func(t *testing.T) {
			_ = parseAll(t, sample)
		})
	}
}

package stream

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/codewandler/markdown/internal/commonmarktests"
)

type corpusStatus string

const (
	statusSupported   corpusStatus = "supported"
	statusKnownGap    corpusStatus = "known_gap"
	statusUnsupported corpusStatus = "unsupported"
)

func TestCommonMarkCorpusClassification(t *testing.T) {
	examples := loadCommonMarkCorpus(t)
	counts := map[corpusStatus]int{}
	sections := map[string]map[corpusStatus]int{}
	for _, ex := range examples {
		status := classifyCommonMarkExample(ex)
		counts[status]++
		if sections[ex.Section] == nil {
			sections[ex.Section] = map[corpusStatus]int{}
		}
		sections[ex.Section][status]++
	}

	if counts[statusSupported] == 0 {
		t.Fatal("CommonMark corpus has no supported examples")
	}
	if counts[statusKnownGap] == 0 {
		t.Fatal("CommonMark corpus has no known-gap examples")
	}
	if counts[statusUnsupported] == 0 {
		t.Fatal("CommonMark corpus has no unsupported examples")
	}

	names := make([]string, 0, len(sections))
	for name := range sections {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := sections[name]
		t.Logf("%s: supported=%d known_gap=%d unsupported=%d", name, c[statusSupported], c[statusKnownGap], c[statusUnsupported])
	}
}

func TestCommonMarkCorpusSplitEquivalence(t *testing.T) {
	examples := loadCommonMarkCorpus(t)
	for _, ex := range examples {
		t.Run(fmt.Sprintf("%03d/%s/%s", ex.Example, ex.Section, classifyCommonMarkExample(ex)), func(t *testing.T) {
			want := viewEvents(parseAll(t, ex.Markdown))
			for split := 0; split <= len(ex.Markdown); split++ {
				got := viewEvents(parseInTwoChunks(t, ex.Markdown, split))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("CommonMark %s example %d split %d mismatch\nmarkdown:\n%s\nwant: %#v\n got: %#v",
						commonmarktests.Version, ex.Example, split, ex.Markdown, want, got)
				}
			}
		})
	}
}

func TestCommonMarkSupportedExamples(t *testing.T) {
	examples := loadCommonMarkCorpus(t)
	byNumber := make(map[int]commonmarktests.Example, len(examples))
	for _, ex := range examples {
		byNumber[ex.Example] = ex
	}

	for number, assert := range supportedCommonMarkExamples {
		ex, ok := byNumber[number]
		if !ok {
			t.Fatalf("supported CommonMark example %d is missing from corpus", number)
		}
		t.Run(fmt.Sprintf("%03d/%s", ex.Example, ex.Section), func(t *testing.T) {
			assert(t, viewEvents(parseAll(t, ex.Markdown)))
		})
	}
}

func parseInTwoChunks(t *testing.T, in string, split int) []Event {
	t.Helper()
	p := NewParser()
	var all []Event
	events, err := p.Write([]byte(in[:split]))
	if err != nil {
		t.Fatal(err)
	}
	all = append(all, events...)
	events, err = p.Write([]byte(in[split:]))
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

func loadCommonMarkCorpus(t *testing.T) []commonmarktests.Example {
	t.Helper()
	examples, err := commonmarktests.Load()
	if err != nil {
		t.Fatal(err)
	}
	return examples
}

func classifyCommonMarkExample(ex commonmarktests.Example) corpusStatus {
	if _, ok := supportedCommonMarkExamples[ex.Example]; ok {
		return statusSupported
	}
	switch ex.Section {
	case "ATX headings",
		"Autolinks",
		"Blank lines",
		"Block quotes",
		"Code spans",
		"Fenced code blocks",
		"Indented code blocks",
		"List items",
		"Lists",
		"Paragraphs",
		"Soft line breaks",
		"Thematic breaks":
		return statusKnownGap
	default:
		return statusUnsupported
	}
}

var supportedCommonMarkExamples = map[int]func(*testing.T, []eventView){
	43:  expectBlocks(BlockThematicBreak, 3),
	44:  expectParagraphText("+++"),
	45:  expectParagraphText("==="),
	46:  expectParagraphText("--", "**", "__"),
	62:  expectHeadingLevels(1, 2, 3, 4, 5, 6),
	63:  expectParagraphText("####### foo"),
	119: expectFencedCode("", "<", " >"),
	120: expectFencedCode("", "<", " >"),
	122: expectFencedCode("", "aaa", "~~~"),
	123: expectFencedCode("", "aaa", "```"),
	142: expectFencedCode("ruby", "def foo(x)", "return 3", "end"),
	219: expectBlocks(BlockParagraph, 2),
	220: expectBlocks(BlockParagraph, 2),
	228: expectBlocks(BlockBlockquote, 1, BlockHeading, 1, BlockParagraph, 1),
	243: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	322: expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1),
	328: expectTextStyle("foo", InlineStyle{Code: true}),
	594: expectTextStyle("http://foo.bar.baz", InlineStyle{Link: "http://foo.bar.baz"}),
	595: expectTextStyle("https://foo.bar.baz/test?q=hello&id=22&boolean", InlineStyle{Link: "https://foo.bar.baz/test?q=hello&id=22&boolean"}),
}

func expectBlocks(pairs ...any) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		if len(pairs)%2 != 0 {
			t.Fatal("expectBlocks requires block/count pairs")
		}
		counts := countEnterBlocks(events)
		for i := 0; i < len(pairs); i += 2 {
			block, ok := pairs[i].(BlockKind)
			if !ok {
				t.Fatalf("expected BlockKind at pair %d, got %T", i/2, pairs[i])
			}
			want, ok := pairs[i+1].(int)
			if !ok {
				t.Fatalf("expected int count at pair %d, got %T", i/2, pairs[i+1])
			}
			if got := counts[block]; got != want {
				t.Fatalf("block %s count mismatch: want %d, got %d in %#v", block, want, got, events)
			}
		}
	}
}

func expectHeadingLevels(levels ...int) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		var got []int
		for _, ev := range events {
			if ev.Kind == EventEnterBlock && ev.Block == BlockHeading {
				got = append(got, ev.Level)
			}
		}
		if !reflect.DeepEqual(got, levels) {
			t.Fatalf("heading levels mismatch: want %#v, got %#v in %#v", levels, got, events)
		}
	}
}

func expectParagraphText(parts ...string) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		if got := countEnterBlocks(events)[BlockParagraph]; got == 0 {
			t.Fatalf("expected paragraph, got %#v", events)
		}
		assertTextParts(t, events, parts...)
	}
}

func expectFencedCode(info string, parts ...string) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		found := false
		for _, ev := range events {
			if ev.Kind == EventEnterBlock && ev.Block == BlockFencedCode && ev.Info == info {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing fenced code block info %q in %#v", info, events)
		}
		assertTextParts(t, events, parts...)
	}
}

func expectTextStyle(text string, style InlineStyle) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		for _, ev := range events {
			if ev.Kind == EventText && ev.Text == text && sameStyle(ev.Style, style) {
				return
			}
		}
		t.Fatalf("missing styled text %q (%#v) in %#v", text, style, events)
	}
}

func countEnterBlocks(events []eventView) map[BlockKind]int {
	counts := map[BlockKind]int{}
	for _, ev := range events {
		if ev.Kind == EventEnterBlock {
			counts[ev.Block]++
		}
	}
	return counts
}

func assertTextParts(t *testing.T, events []eventView, parts ...string) {
	t.Helper()
	text := eventText(events)
	for _, part := range parts {
		if !strings.Contains(text, part) {
			t.Fatalf("missing text %q in %q from %#v", part, text, events)
		}
	}
}

func eventText(events []eventView) string {
	var b strings.Builder
	for _, ev := range events {
		if ev.Kind == EventText {
			b.WriteString(ev.Text)
		}
	}
	return b.String()
}

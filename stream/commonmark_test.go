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
	wantCounts := map[corpusStatus]int{
		statusSupported:   255,
		statusKnownGap:    152,
		statusUnsupported: 245,
	}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("CommonMark corpus classification changed\nwant: %#v\n got: %#v", wantCounts, counts)
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
		"Backslash escapes",
		"Blank lines",
		"Block quotes",
		"Code spans",
		"Entity and numeric character references",
		"Fenced code blocks",
		"Hard line breaks",
		"Indented code blocks",
		"Inlines",
		"Links",
		"List items",
		"Lists",
		"Paragraphs",
		"Precedence",
		"Setext headings",
		"Soft line breaks",
		"Tabs",
		"Textual content",
		"Thematic breaks":
		return statusKnownGap
	default:
		return statusUnsupported
	}
}

var supportedCommonMarkExamples = map[int]func(*testing.T, []eventView){
	1:   expectBlocks(BlockIndentedCode, 1),
	2:   expectBlocks(BlockIndentedCode, 1),
	3:   expectBlocks(BlockIndentedCode, 1),
	8:   expectBlocks(BlockIndentedCode, 1),
	10:  expectHeadingLevels(1),
	11:  expectBlocks(BlockThematicBreak, 1),
	12:  expectTextParts("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"),
	13:  expectTextParts("\\\t\\A\\a\\ \\3\\φ\\«"),
	14:  expectTextParts("*not emphasized*", "<br/> not a tag", "[not a link](/foo)", "`not code`", "1. not a list", "* not a list", "# not a heading", "[foo]: /url \"not a reference\"", "&ouml; not a character entity"),
	15:  expectTextParts("\\", "emphasis"),
	16:  expectLineBreaks(1),
	17:  expectTextStyle("\\[\\`", InlineStyle{Code: true}),
	18:  expectBlocks(BlockIndentedCode, 1),
	19:  expectFencedCode("", "\\[\\]"),
	20:  expectTextStyle("https://example.com?find=\\*", InlineStyle{Link: "https://example.com?find=\\*"}),
	24:  expectFencedCode("foo+bar", "foo"),
	25:  expectTextParts("  & © Æ Ď", "¾ ℋ ⅆ", "∲ ≧̸"),
	26:  expectTextParts("# Ӓ Ϡ �"),
	27:  expectTextParts("\" ആ ಫ"),
	28:  expectTextParts("&nbsp &x; &#; &#x;", "&#87654321;", "&#abcdef0;", "&ThisIsNotDefined; &hi?;"),
	29:  expectTextParts("&copy"),
	30:  expectTextParts("&MadeUpEntity;"),
	34:  expectFencedCode("föö", "foo"),
	35:  expectTextStyle("f&ouml;&ouml;", InlineStyle{Code: true}),
	36:  expectTextParts("f&ouml;f&ouml;"),
	37:  expectTextParts("*foo*", "foo"),
	38:  expectBlocks(BlockParagraph, 2, BlockList, 1, BlockListItem, 1),
	39:  expectTextParts("foo\n\nbar"),
	40:  expectTextParts("\tfoo"),
	42:  expectBlocks(BlockList, 1, BlockListItem, 2),
	43:  expectBlocks(BlockThematicBreak, 3),
	44:  expectParagraphText("+++"),
	45:  expectParagraphText("==="),
	46:  expectParagraphText("--", "**", "__"),
	47:  expectBlocks(BlockThematicBreak, 3),
	48:  expectBlocks(BlockIndentedCode, 1),
	49:  expectParagraphText("Foo", "***"),
	50:  expectBlocks(BlockThematicBreak, 1),
	51:  expectBlocks(BlockThematicBreak, 1),
	52:  expectBlocks(BlockThematicBreak, 1),
	53:  expectBlocks(BlockThematicBreak, 1),
	54:  expectBlocks(BlockThematicBreak, 1),
	55:  expectBlocks(BlockParagraph, 3),
	56:  expectTextStyle("-", InlineStyle{Emphasis: true}),
	57:  expectBlocks(BlockList, 2, BlockListItem, 2, BlockThematicBreak, 1),
	58:  expectBlocks(BlockParagraph, 2, BlockThematicBreak, 1),
	60:  expectBlocks(BlockList, 2, BlockListItem, 2, BlockThematicBreak, 1),
	62:  expectHeadingLevels(1, 2, 3, 4, 5, 6),
	63:  expectParagraphText("####### foo"),
	64:  expectBlocks(BlockParagraph, 2),
	65:  expectParagraphText("## foo"),
	66:  expectTextParts("foo ", "bar", "*baz*"),
	67:  expectTextParts("foo"),
	68:  expectHeadingLevels(3, 2, 1),
	69:  expectBlocks(BlockIndentedCode, 1),
	70:  expectParagraphText("foo", "# bar"),
	71:  expectHeadingLevels(2, 3),
	72:  expectHeadingLevels(1, 5),
	73:  expectHeadingLevels(3),
	74:  expectTextParts("foo ### b"),
	75:  expectTextParts("foo#"),
	76:  expectTextParts("foo ###", "foo ###", "foo #"),
	77:  expectBlocks(BlockThematicBreak, 2, BlockHeading, 1),
	78:  expectBlocks(BlockParagraph, 2, BlockHeading, 1),
	79:  expectHeadingLevels(2, 1, 3),
	80:  expectHeadingLevels(1, 2),
	81:  expectHeadingLevels(1),
	82:  expectHeadingLevels(1),
	83:  expectHeadingLevels(2, 1),
	84:  expectHeadingLevels(2, 2, 1),
	85:  expectBlocks(BlockIndentedCode, 1, BlockThematicBreak, 1),
	86:  expectHeadingLevels(2),
	87:  expectParagraphText("Foo", "---"),
	88:  expectBlocks(BlockParagraph, 2, BlockThematicBreak, 1),
	89:  expectHeadingLevels(2),
	90:  expectHeadingLevels(2),
	91:  expectBlocks(BlockHeading, 2, BlockParagraph, 2),
	92:  expectBlocks(BlockBlockquote, 1, BlockParagraph, 1, BlockThematicBreak, 1),
	93:  expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	94:  expectBlocks(BlockList, 1, BlockListItem, 1, BlockThematicBreak, 1),
	95:  expectHeadingLevels(2),
	96:  expectBlocks(BlockThematicBreak, 1, BlockHeading, 2, BlockParagraph, 1),
	97:  expectParagraphText("===="),
	98:  expectBlocks(BlockThematicBreak, 2),
	99:  expectBlocks(BlockList, 1, BlockListItem, 1, BlockThematicBreak, 1),
	100: expectBlocks(BlockIndentedCode, 1, BlockThematicBreak, 1),
	101: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1, BlockThematicBreak, 1),
	102: expectHeadingLevels(2),
	103: expectBlocks(BlockParagraph, 2, BlockHeading, 1),
	104: expectBlocks(BlockParagraph, 2, BlockThematicBreak, 1),
	105: expectBlocks(BlockParagraph, 2, BlockThematicBreak, 1),
	106: expectBlocks(BlockParagraph, 1),
	107: expectBlocks(BlockIndentedCode, 1),
	110: expectBlocks(BlockIndentedCode, 1),
	111: expectBlocks(BlockIndentedCode, 1),
	112: expectBlocks(BlockIndentedCode, 1),
	113: expectParagraphText("Foo", "bar"),
	114: expectBlocks(BlockIndentedCode, 1, BlockParagraph, 1),
	115: expectBlocks(BlockHeading, 2, BlockIndentedCode, 2, BlockThematicBreak, 1),
	116: expectBlocks(BlockIndentedCode, 1),
	117: expectBlocks(BlockIndentedCode, 1),
	118: expectBlocks(BlockIndentedCode, 1),
	119: expectFencedCode("", "<", " >"),
	120: expectFencedCode("", "<", " >"),
	122: expectFencedCode("", "aaa", "~~~"),
	123: expectFencedCode("", "aaa", "```"),
	124: expectFencedCode("", "aaa", "```"),
	125: expectFencedCode("", "aaa", "~~~"),
	126: expectBlocks(BlockFencedCode, 1),
	129: expectBlocks(BlockFencedCode, 1),
	130: expectBlocks(BlockFencedCode, 1),
	131: expectFencedCode("", "aaa"),
	132: expectFencedCode("", "aaa"),
	133: expectFencedCode("", "aaa", " aaa"),
	134: expectBlocks(BlockIndentedCode, 1),
	135: expectFencedCode("", "aaa"),
	136: expectFencedCode("", "aaa"),
	137: expectFencedCode("", "aaa", "    ```"),
	140: expectBlocks(BlockParagraph, 2, BlockFencedCode, 1),
	142: expectFencedCode("ruby", "def foo(x)", "return 3", "end"),
	143: expectFencedCode("ruby startline=3 $%@#$", "def foo(x)", "return 3", "end"),
	144: expectFencedCode(";", ""),
	147: expectFencedCode("", "``` aaa"),
	219: expectBlocks(BlockParagraph, 2),
	220: expectBlocks(BlockParagraph, 2),
	221: expectBlocks(BlockParagraph, 2),
	222: expectParagraphText("aaa", "bbb"),
	223: expectParagraphText("aaa", "bbb", "ccc"),
	224: expectParagraphText("aaa", "bbb"),
	225: expectBlocks(BlockIndentedCode, 1, BlockParagraph, 1),
	226: expectLineBreaks(1),
	227: expectBlocks(BlockParagraph, 1, BlockHeading, 1),
	228: expectBlocks(BlockBlockquote, 1, BlockHeading, 1, BlockParagraph, 1),
	229: expectBlocks(BlockBlockquote, 1, BlockHeading, 1, BlockParagraph, 1),
	230: expectBlocks(BlockBlockquote, 1, BlockHeading, 1, BlockParagraph, 1),
	231: expectBlocks(BlockIndentedCode, 1),
	232: expectBlocks(BlockBlockquote, 1, BlockHeading, 1, BlockParagraph, 1),
	233: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	234: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1, BlockThematicBreak, 1),
	238: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	239: expectBlocks(BlockBlockquote, 1, BlockParagraph, 0),
	240: expectBlocks(BlockBlockquote, 1, BlockParagraph, 0),
	241: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	242: expectBlocks(BlockBlockquote, 2, BlockParagraph, 2),
	243: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	244: expectBlocks(BlockBlockquote, 1, BlockParagraph, 2),
	245: expectBlocks(BlockParagraph, 2, BlockBlockquote, 1),
	246: expectBlocks(BlockBlockquote, 2, BlockParagraph, 2, BlockThematicBreak, 1),
	247: expectBlocks(BlockBlockquote, 1, BlockParagraph, 1),
	248: expectBlocks(BlockBlockquote, 1, BlockParagraph, 2),
	249: expectBlocks(BlockBlockquote, 1, BlockParagraph, 2),
	261: expectBlocks(BlockParagraph, 2),
	265: expectOrderedList(123456789, 1),
	266: expectParagraphText("1234567890. not ok"),
	267: expectOrderedList(0, 1),
	268: expectOrderedList(3, 1),
	269: expectParagraphText("-1. not ok"),
	272: expectBlocks(BlockIndentedCode, 2, BlockParagraph, 1),
	275: expectBlocks(BlockParagraph, 2),
	280: expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1),
	281: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 2),
	282: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 2),
	283: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 2),
	284: expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 0),
	285: expectBlocks(BlockParagraph, 2),
	301: expectBlocks(BlockList, 2, BlockListItem, 3),
	302: expectBlocks(BlockList, 2, BlockListItem, 3),
	304: expectBlocks(BlockParagraph, 1, BlockList, 0),
	305: expectBlocks(BlockParagraph, 2, BlockList, 1, BlockListItem, 1),
	322: expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1),
	327: expectTextStyle("hi", InlineStyle{Code: true}),
	328: expectTextStyle("foo", InlineStyle{Code: true}),
	329: expectTextStyle("foo ` bar", InlineStyle{Code: true}),
	330: expectTextStyle("``", InlineStyle{Code: true}),
	331: expectTextStyle(" `` ", InlineStyle{Code: true}),
	332: expectTextStyle(" a", InlineStyle{Code: true}),
	333: expectTextStyle(" b ", InlineStyle{Code: true}),
	334: expectTextStyle("  ", InlineStyle{Code: true}),
	335: expectTextStyle("foo bar   baz", InlineStyle{Code: true}),
	336: expectTextStyle("foo ", InlineStyle{Code: true}),
	337: expectTextStyle("foo   bar  baz", InlineStyle{Code: true}),
	338: expectTextStyle("foo\\", InlineStyle{Code: true}),
	339: expectTextStyle("foo`bar", InlineStyle{Code: true}),
	340: expectTextStyle("foo `` bar", InlineStyle{Code: true}),
	343: expectTextStyle("<a href=\"", InlineStyle{Code: true}),
	345: expectTextStyle("<https://foo.bar.", InlineStyle{Code: true}),
	346: expectTextStyle("https://foo.bar.`baz", InlineStyle{Link: "https://foo.bar.`baz"}),
	347: expectParagraphText("```foo``"),
	348: expectParagraphText("`foo"),
	349: expectTextStyle("bar", InlineStyle{Code: true}),
	482: expectTextStyle("link", InlineStyle{Link: "/uri", LinkTitle: "title"}),
	483: expectTextStyle("link", InlineStyle{Link: "/uri"}),
	484: expectTextStyle("", InlineStyle{Link: "./target.md"}),
	485: expectTextStyle("link", InlineStyle{Link: ""}),
	486: expectTextStyle("link", InlineStyle{Link: ""}),
	487: expectTextStyle("", InlineStyle{Link: ""}),
	488: expectParagraphText("[link](/my uri)"),
	489: expectTextStyle("link", InlineStyle{Link: "/my uri"}),
	490: expectParagraphText("[link](foo", "bar)"),
	491: expectParagraphText("[link](<foo", "bar>)"),
	492: expectTextStyle("a", InlineStyle{Link: "b)c"}),
	493: expectParagraphText("[link](<foo>)"),
	495: expectTextStyle("link", InlineStyle{Link: "(foo)"}),
	496: expectTextStyle("link", InlineStyle{Link: "foo(and(bar))"}),
	497: expectParagraphText("[link](foo(and(bar))"),
	498: expectTextStyle("link", InlineStyle{Link: "foo(and(bar)"}),
	499: expectTextStyle("link", InlineStyle{Link: "foo(and(bar)"}),
	500: expectTextStyle("link", InlineStyle{Link: "foo):"}),
	501: expectTextStyle("link", InlineStyle{Link: "#fragment"}),
	502: expectTextStyle("link", InlineStyle{Link: "foo\\bar"}),
	503: expectTextStyle("link", InlineStyle{Link: "foo%20bä"}),
	504: expectTextStyle("link", InlineStyle{Link: "\"title\""}),
	505: expectTextStyle("link", InlineStyle{Link: "/url", LinkTitle: "title"}),
	506: expectTextStyle("link", InlineStyle{Link: "/url", LinkTitle: "title \"\""}),
	507: expectTextStyle("link", InlineStyle{Link: "/url \"title\""}),
	508: expectParagraphText("[link](/url \"title \"and\" title\")"),
	509: expectTextStyle("link", InlineStyle{Link: "/url", LinkTitle: "title \"and\" title"}),
	510: expectTextStyle("link", InlineStyle{Link: "/uri", LinkTitle: "title"}),
	511: expectParagraphText("[link] (/uri)"),
	512: expectTextStyle("link [foo [bar]]", InlineStyle{Link: "/uri"}),
	594: expectTextStyle("http://foo.bar.baz", InlineStyle{Link: "http://foo.bar.baz"}),
	595: expectTextStyle("https://foo.bar.baz/test?q=hello&id=22&boolean", InlineStyle{Link: "https://foo.bar.baz/test?q=hello&id=22&boolean"}),
	596: expectTextStyle("irc://foo.bar:2233/baz", InlineStyle{Link: "irc://foo.bar:2233/baz"}),
	597: expectTextStyle("MAILTO:FOO@BAR.BAZ", InlineStyle{Link: "MAILTO:FOO@BAR.BAZ"}),
	598: expectTextStyle("a+b+c:d", InlineStyle{Link: "a+b+c:d"}),
	599: expectTextStyle("made-up-scheme://foo,bar", InlineStyle{Link: "made-up-scheme://foo,bar"}),
	600: expectTextStyle("https://../", InlineStyle{Link: "https://../"}),
	601: expectTextStyle("localhost:5001/foo", InlineStyle{Link: "localhost:5001/foo"}),
	602: expectParagraphText("<https://foo.bar/baz bim>"),
	603: expectTextStyle("https://example.com/\\[\\", InlineStyle{Link: "https://example.com/\\[\\"}),
	604: expectTextStyle("foo@bar.example.com", InlineStyle{Link: "mailto:foo@bar.example.com"}),
	605: expectTextStyle("foo+special@Bar.baz-bar0.com", InlineStyle{Link: "mailto:foo+special@Bar.baz-bar0.com"}),
	606: expectParagraphText("<foo+@bar.example.com>"),
	607: expectParagraphText("<>"),
	608: expectParagraphText("< https://foo.bar >"),
	609: expectParagraphText("<m:abc>"),
	610: expectParagraphText("<foo.bar.baz>"),
	611: expectParagraphText("https://example.com"),
	612: expectParagraphText("foo@bar.example.com"),
	633: expectLineBreaks(1),
	634: expectLineBreaks(1),
	635: expectLineBreaks(1),
	636: expectLineBreaks(1),
	637: expectLineBreaks(1),
	640: expectTextStyle("code   span", InlineStyle{Code: true}),
	641: expectTextStyle("code\\ span", InlineStyle{Code: true}),
	644: expectParagraphText("foo\\"),
	645: expectLineBreaks(0),
	646: expectTextParts("foo\\"),
	647: expectTextParts("foo"),
	648: expectSoftBreaks(1),
	649: expectSoftBreaks(1),
	650: expectParagraphText("hello $.;'there"),
	651: expectParagraphText("Foo χρῆν"),
	652: expectParagraphText("Multiple     spaces"),
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

func expectTextParts(parts ...string) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
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

func expectOrderedList(start int, items int) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		found := false
		for _, ev := range events {
			if ev.Kind == EventEnterBlock && ev.Block == BlockList && ev.List != nil && ev.List.Ordered && ev.List.Start == start {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing ordered list start %d in %#v", start, events)
		}
		expectBlocks(BlockList, 1, BlockListItem, items)(t, events)
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

func expectSoftBreaks(want int) func(*testing.T, []eventView) {
	return expectEventKind(EventSoftBreak, want)
}

func expectLineBreaks(want int) func(*testing.T, []eventView) {
	return expectEventKind(EventLineBreak, want)
}

func expectEventKind(kind EventKind, want int) func(*testing.T, []eventView) {
	return func(t *testing.T, events []eventView) {
		t.Helper()
		got := 0
		for _, ev := range events {
			if ev.Kind == kind {
				got++
			}
		}
		if got != want {
			t.Fatalf("event kind %s count mismatch: want %d, got %d in %#v", kind, want, got, events)
		}
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

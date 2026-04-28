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
	wantCounts := map[corpusStatus]int{
		statusSupported: 601,
		statusKnownGap:  51,
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
		"Emphasis and strong emphasis",
		"Entity and numeric character references",
		"Fenced code blocks",
		"Hard line breaks",
		"HTML blocks",
		"Images",
		"Indented code blocks",
		"Inlines",
		"Link reference definitions",
		"Links",
		"List items",
		"Lists",
		"Paragraphs",
		"Precedence",
		"Raw HTML",
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
	192: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	193: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "the title"}),
	194: expectTextStyle("Foo*bar]", InlineStyle{Link: "my_(url)", LinkTitle: "title (with parens)"}),
	195: expectTextStyle("Foo bar", InlineStyle{Link: "my url", LinkTitle: "title"}),
	198: expectTextStyle("foo", InlineStyle{Link: "/url"}),
	200: expectTextStyle("foo", InlineStyle{Link: ""}),
	202: expectTextStyle("foo", InlineStyle{Link: "/url\\bar*baz", LinkTitle: "foo\"bar\\baz"}),
	205: expectTextStyle("Foo", InlineStyle{Link: "/url"}),
	206: expectTextStyle("αγω", InlineStyle{Link: "/φου"}),
	207: expectBlocks(BlockDocument, 0),
	350: expectTextStyle("foo bar", InlineStyle{Emphasis: true}),
	355: expectTextStyle("bar", InlineStyle{Emphasis: true}),
	357: expectTextStyle("foo bar", InlineStyle{Emphasis: true}),
	364: expectTextStyle("(bar)", InlineStyle{Emphasis: true}),
	356: expectTextStyle("6", InlineStyle{Emphasis: true}),
	370: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	376: expectTextStyle("foo_bar_baz", InlineStyle{Emphasis: true}),
	377: expectTextStyle("(bar)", InlineStyle{Emphasis: true}),
	378: expectTextStyle("foo bar", InlineStyle{Strong: true}),
	381: expectTextStyle("bar", InlineStyle{Strong: true}),
	382: expectTextStyle("foo bar", InlineStyle{Strong: true}),
	390: expectTextStyle("(bar)", InlineStyle{Strong: true}),
	393: expectTextStyle("foo", InlineStyle{Emphasis: true, Strong: true}),
	395: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo \"", InlineStyle{Strong: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle("\" foo", InlineStyle{Strong: true})(t, events)
	},
	396: expectTextStyle("foo", InlineStyle{Strong: true}),
	399: expectTextStyle("foo", InlineStyle{Emphasis: true, Strong: true}),
	460: expectTextStyle("foo", InlineStyle{Strong: true}),
	461: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	462: expectTextStyle("foo", InlineStyle{Strong: true}),
	463: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	464: expectTextStyle("foo", InlineStyle{Strong: true}),
	465: expectTextStyle("foo", InlineStyle{Strong: true}),
	466: expectTextStyle("foo", InlineStyle{Strong: true}),
	467: expectTextStyle("foo", InlineStyle{Emphasis: true, Strong: true}),
	468: expectTextStyle("foo", InlineStyle{Emphasis: true, Strong: true}),
	209: expectParagraphText("[foo]: /url \"title\" ok"),
	210: expectParagraphText("\"title\" ok"),
	211: expectBlocks(BlockIndentedCode, 1, BlockParagraph, 1),
	212: expectBlocks(BlockFencedCode, 1, BlockParagraph, 1),
	213: expectTextParts("Foo", "[bar]: /baz", "[bar]"),
	215: expectBlocks(BlockHeading, 1, BlockParagraph, 1),
	216: expectTextStyle("foo", InlineStyle{Link: "/url"}),
	217: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Link: "/foo-url", LinkTitle: "foo"})(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/bar-url", LinkTitle: "bar"})(t, events)
		expectTextStyle("baz", InlineStyle{Link: "/baz-url"})(t, events)
	},
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
	638: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true})(t, events)
		expectLineBreaks(1)(t, events)
	},
	639: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true})(t, events)
		expectLineBreaks(1)(t, events)
	},
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
	572: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	574: expectTextStyle("foo ![bar](/url)", InlineStyle{Link: "/url2"}),
	575: expectTextStyle("foo [bar](/url)", InlineStyle{Link: "/url2"}),
	578: expectTextStyle("foo", InlineStyle{Link: "train.jpg"}),
	579: expectTextStyle("foo bar", InlineStyle{Link: "/path/to/train.jpg", LinkTitle: "title"}),
	580: expectTextStyle("foo", InlineStyle{Link: "url"}),
	581: expectTextStyle("", InlineStyle{Link: "/url"}),
	// Emphasis and strong emphasis — rules 1-17 (newly supported via CommonMark algorithm).
	// Rule 1: * can open emphasis iff left-flanking.
	351: expectParagraphText("a * foo bar*"),
	352: expectParagraphText("a*\"foo\"*"),
	353: expectParagraphText("*\u00a0a\u00a0*"),
	354: expectParagraphText("*$*alpha.", "*£*bravo.", "*€*charlie."),
	// Rule 2: _ can open emphasis with extra restrictions.
	358: expectParagraphText("_ foo bar_"),
	359: expectParagraphText("a_\"foo\"_"),
	360: expectParagraphText("foo_bar_"),
	361: expectParagraphText("5_6_78"),
	362: expectParagraphText("пристаням_стремятся_"),
	363: expectParagraphText("aa_\"bb\"_cc"),
	// Rule 3: * can close emphasis iff right-flanking.
	365: expectParagraphText("_foo*"),
	366: expectParagraphText("*foo bar *"),
	368: expectParagraphText("*(*foo)"),
	369: expectTextStyle("(foo)", InlineStyle{Emphasis: true}),
	// Rule 4: _ can close emphasis with extra restrictions.
	371: expectParagraphText("_foo bar _"),
	372: expectParagraphText("_(_foo)"),
	373: expectTextStyle("(foo)", InlineStyle{Emphasis: true}),
	374: expectParagraphText("_foo_bar"),
	375: expectParagraphText("_пристаням_стремятся"),
	// Rule 5: ** can open strong iff left-flanking.
	379: expectParagraphText("** foo bar**"),
	380: expectParagraphText("a**\"foo\"**"),
	// Rule 6: __ can open strong with extra restrictions.
	383: expectParagraphText("__ foo bar__"),
	384: expectParagraphText("__", "foo bar__"),
	385: expectParagraphText("a__\"foo\"__"),
	386: expectParagraphText("foo__bar__"),
	387: expectParagraphText("5__6__78"),
	388: expectParagraphText("пристаням__стремятся__"),
	389: expectTextStyle("foo, bar, baz", InlineStyle{Strong: true}),
	// Rule 7: ** can close strong iff right-flanking.
	391: expectParagraphText("**foo bar **"),
	392: expectParagraphText("**(**foo)"),
	394: expectTextStyle("Gomphocarpus (", InlineStyle{Strong: true}),
	// Rule 8: __ can close strong with extra restrictions.
	397: expectParagraphText("__foo bar __"),
	398: expectParagraphText("__(__foo)"),
	400: expectParagraphText("__foo__bar"),
	401: expectParagraphText("__пристаням__стремятся"),
	402: expectTextStyle("foo__bar__baz", InlineStyle{Strong: true}),
	403: expectTextStyle("(bar)", InlineStyle{Strong: true}),
	// Rule 9: emphasis nesting.
	405: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	406: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle(" baz", InlineStyle{Emphasis: true})(t, events)
	},
	407: expectTextStyle("foo bar baz", InlineStyle{Emphasis: true}),
	408: expectTextStyle("foo bar", InlineStyle{Emphasis: true}),
	409: expectTextStyle("foo bar", InlineStyle{Emphasis: true}),
	410: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle(" baz", InlineStyle{Emphasis: true})(t, events)
	},
	411: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	412: expectTextStyle("foo**bar", InlineStyle{Emphasis: true}),
	413: expectTextStyle("foo", InlineStyle{Emphasis: true, Strong: true}),
	414: expectTextStyle("foo ", InlineStyle{Emphasis: true}),
	415: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	416: expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true}),
	417: expectTextStyle("bar***", InlineStyle{Strong: true}),
	418: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar baz bim", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle(" bop", InlineStyle{Emphasis: true})(t, events)
	},
	// Rule 10: strong emphasis nesting.
	420: expectParagraphText("** is not an empty emphasis"),
	421: expectParagraphText("**** is not an empty strong emphasis"),
	423: expectTextStyle("foo", InlineStyle{Strong: true}),
	424: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Strong: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle(" baz", InlineStyle{Strong: true})(t, events)
	},
	425: expectTextStyle("foo bar baz", InlineStyle{Strong: true}),
	426: expectTextStyle("foo bar", InlineStyle{Strong: true}),
	427: expectTextStyle("foo bar", InlineStyle{Strong: true}),
	428: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Strong: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle(" baz", InlineStyle{Strong: true})(t, events)
	},
	429: expectTextStyle("foo", InlineStyle{Strong: true}),
	430: expectTextStyle("foo", InlineStyle{Emphasis: true, Strong: true}),
	431: expectTextStyle("foo ", InlineStyle{Strong: true}),
	432: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Strong: true})(t, events)
		expectTextStyle("bar baz", InlineStyle{Emphasis: true, Strong: true})(t, events)
		expectTextStyle(" bop", InlineStyle{Strong: true})(t, events)
	},
	// Rules 11-12: literal delimiters.
	434: expectParagraphText("__ is not an empty emphasis"),
	435: expectParagraphText("____ is not an empty strong emphasis"),
	436: expectParagraphText("foo ***"),
	437: expectTextStyle("*", InlineStyle{Emphasis: true}),
	438: expectTextStyle("_", InlineStyle{Emphasis: true}),
	439: expectParagraphText("foo *****"),
	440: expectTextStyle("*", InlineStyle{Strong: true}),
	441: expectTextStyle("_", InlineStyle{Strong: true}),
	442: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Emphasis: true})(t, events)
	},
	443: expectTextStyle("foo*", InlineStyle{Emphasis: true}),
	444: expectTextStyle("foo", InlineStyle{Strong: true}),
	445: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	446: expectTextStyle("foo*", InlineStyle{Strong: true}),
	447: expectTextStyle("foo***", InlineStyle{Emphasis: true}),
	448: expectParagraphText("foo ___"),
	449: expectTextStyle("_", InlineStyle{Emphasis: true}),
	450: expectTextStyle("*", InlineStyle{Emphasis: true}),
	451: expectParagraphText("foo _____"),
	452: expectTextStyle("_", InlineStyle{Strong: true}),
	453: expectTextStyle("*", InlineStyle{Strong: true}),
	454: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	455: expectTextStyle("foo_", InlineStyle{Emphasis: true}),
	456: expectTextStyle("foo", InlineStyle{Strong: true}),
	457: expectTextStyle("foo", InlineStyle{Emphasis: true}),
	458: expectTextStyle("foo_", InlineStyle{Strong: true}),
	459: expectTextStyle("foo___", InlineStyle{Emphasis: true}),
	// Rules 15-16: overlap and precedence.
	469: expectTextStyle("foo _bar", InlineStyle{Emphasis: true}),
	470: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar *baz bim", InlineStyle{Emphasis: true, Strong: true})(t, events)
	},
	471: expectTextStyle("bar baz", InlineStyle{Strong: true}),
	472: expectTextStyle("bar baz", InlineStyle{Emphasis: true}),
	// Emphasis containing inline links.
	404: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Link: "/url"})(t, events)
	},
	422: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo ", InlineStyle{Strong: true})(t, events)
		expectTextStyle("bar", InlineStyle{Strong: true, Link: "/url"})(t, events)
	},
	// Emphasis delimiters consumed by link text.
	473: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("*")(t, events)
		expectTextStyle("bar*", InlineStyle{Link: "/url"})(t, events)
	},
	474: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("_foo ")(t, events)
		expectTextStyle("bar_", InlineStyle{Link: "/url"})(t, events)
	},
	// Code spans take priority over emphasis and links.
	341: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("*foo")(t, events)
		expectTextStyle("*", InlineStyle{Code: true})(t, events)
	},
	342: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[not a ")(t, events)
		expectTextStyle("link](/foo", InlineStyle{Code: true})(t, events)
		expectParagraphText(")")(t, events)
	},
	// Emphasis delimiters inside code spans.
	478: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("a ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("*", InlineStyle{Emphasis: true, Code: true})(t, events)
	},
	479: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("a ", InlineStyle{Emphasis: true})(t, events)
		expectTextStyle("_", InlineStyle{Emphasis: true, Code: true})(t, events)
	},
	// Emphasis delimiters inside autolinks.
	480: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("**a")(t, events)
		expectTextStyle("https://foo.bar/?q=**", InlineStyle{Link: "https://foo.bar/?q=**"})(t, events)
	},
	481: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("__a")(t, events)
		expectTextStyle("https://foo.bar/?q=__", InlineStyle{Link: "https://foo.bar/?q=__"})(t, events)
	},
	// Backslash escapes in link destinations and titles.
	22: expectTextStyle("foo", InlineStyle{Link: "/bar*", LinkTitle: "ti*tle"}),
	23: expectTextStyle("foo", InlineStyle{Link: "/bar*", LinkTitle: "ti*tle"}),
	// Entity references in link destinations and titles.
	32: expectTextStyle("foo", InlineStyle{Link: "/föö", LinkTitle: "föö"}),
	33: expectTextStyle("foo", InlineStyle{Link: "/föö", LinkTitle: "föö"}),
	41: expectParagraphText("[a](url \"tit\")"),
	// Thematic break: setext heading takes precedence over thematic break.
	59: func(t *testing.T, events []eventView) {
		t.Helper()
		expectHeadingLevels(2)(t, events)
		expectParagraphText("bar")(t, events)
	},
	// Fenced code blocks — edge cases.
	121: expectTextStyle("foo", InlineStyle{Code: true}),
	127: expectBlocks(BlockFencedCode, 1),
	138: expectTextStyle(" ", InlineStyle{Code: true}),
	139: expectBlocks(BlockFencedCode, 1),
	141: func(t *testing.T, events []eventView) {
		t.Helper()
		expectHeadingLevels(2, 1)(t, events)
		expectBlocks(BlockFencedCode, 1)(t, events)
	},
	145: expectTextStyle("aa", InlineStyle{Code: true}),
	146: expectBlocks(BlockFencedCode, 1),
	// Link reference definitions — forward references and first-wins.
	203: expectTextStyle("foo", InlineStyle{Link: "url"}),
	204: expectTextStyle("foo", InlineStyle{Link: "first"}),
	// Links — bracket edge cases.
	494: expectParagraphText("[a](<b)c", "[a](<b)c>", "[a](<b>c)"),
	513: expectParagraphText("[link] bar](/uri)"),
	514: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[link ")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/uri"})(t, events)
	},
	515: expectTextStyle("link [bar", InlineStyle{Link: "/uri"}),
	521: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("*")(t, events)
		expectTextStyle("foo*", InlineStyle{Link: "/uri"})(t, events)
	},
	522: expectTextStyle("foo *bar", InlineStyle{Link: "baz*"}),
	523: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo [bar", InlineStyle{Emphasis: true})(t, events)
		expectParagraphText(" baz]")(t, events)
	},
	525: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo")(t, events)
		expectTextStyle("](/uri)", InlineStyle{Code: true})(t, events)
	},
	527: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	529: expectTextStyle("link [bar", InlineStyle{Link: "/uri"}),
	539: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	553: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	555: expectTextStyle("Foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	557: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	560: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[[bar ")(t, events)
		expectTextStyle("foo", InlineStyle{Link: "/url"})(t, events)
	},
	561: expectTextStyle("Foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	562: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Link: "/url"})(t, events)
		expectParagraphText(" bar")(t, events)
	},
	563: expectParagraphText("[foo]"),
	564: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("*")(t, events)
		expectTextStyle("foo*", InlineStyle{Link: "/url"})(t, events)
	},
	565: expectTextStyle("foo", InlineStyle{Link: "/url2"}),
	566: expectTextStyle("foo", InlineStyle{Link: "/url1"}),
	568: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Link: "/url1"})(t, events)
		expectParagraphText("(not a link)")(t, events)
	},
	569: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo]")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/url"})(t, events)
	},
	570: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Link: "/url2"})(t, events)
		expectTextStyle("baz", InlineStyle{Link: "/url1"})(t, events)
	},
	571: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo]")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/url1"})(t, events)
	},
	// Reference links — additional passing examples.
	528: expectTextStyle("link [foo [bar]]", InlineStyle{Link: "/uri"}),
	534: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("*")(t, events)
		expectTextStyle("foo*", InlineStyle{Link: "/uri"})(t, events)
	},
	535: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo *bar", InlineStyle{Link: "/uri"})(t, events)
		expectParagraphText("*")(t, events)
	},
	537: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo")(t, events)
		expectTextStyle("][ref]", InlineStyle{Code: true})(t, events)
	},
	542: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo] ")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
	},
	543: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo]")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
	},
	544: expectTextStyle("bar", InlineStyle{Link: "/url1"}),
	549: expectTextStyle("foo", InlineStyle{Link: "/uri"}),
	550: expectTextStyle("bar\\", InlineStyle{Link: "/uri"}),
	551: expectParagraphText("[]", "[]: /uri"),
	552: expectParagraphText("[", "]", "[", "]: /uri"),
	554: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Emphasis: true, Link: "/url", LinkTitle: "title"})(t, events)
		expectTextStyle(" bar", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
	},
	556: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
		expectParagraphText("[]")(t, events)
	},
	558: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Emphasis: true, Link: "/url", LinkTitle: "title"})(t, events)
		expectTextStyle(" bar", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
	},
	// Nested links rejected — inner link wins.
	518: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo ")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/uri"})(t, events)
		expectParagraphText("](/uri)")(t, events)
	},
	532: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo ")(t, events)
		expectTextStyle("bar", InlineStyle{Link: "/uri"})(t, events)
		expectParagraphText("]")(t, events)
		expectTextStyle("ref", InlineStyle{Link: "/uri"})(t, events)
	},
	// Raw label normalization — backslash escapes not processed.
	545: expectParagraphText("[bar][foo!]"),
	// Link text with inner emphasis, strong, and code.
	516: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("link ", InlineStyle{Link: "/uri"})(t, events)
		expectTextStyle("foo ", InlineStyle{Emphasis: true, Link: "/uri"})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true, Link: "/uri"})(t, events)
		expectTextStyle("#", InlineStyle{Emphasis: true, Code: true, Link: "/uri"})(t, events)
	},
	530: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("link ", InlineStyle{Link: "/uri"})(t, events)
		expectTextStyle("foo ", InlineStyle{Emphasis: true, Link: "/uri"})(t, events)
		expectTextStyle("bar", InlineStyle{Emphasis: true, Strong: true, Link: "/uri"})(t, events)
		expectTextStyle("#", InlineStyle{Emphasis: true, Code: true, Link: "/uri"})(t, events)
	},
	// Invalid labels with unescaped brackets.
	546: expectParagraphText("[foo][ref[]", "[ref[]: /uri"),
	547: expectParagraphText("[foo][ref[bar]]", "[ref[bar]]: /uri"),
	548: expectParagraphText("[[[foo]]]", "[[[foo]]]: /url"),
	// Nested links rejected with inner emphasis.
	519: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[foo ")(t, events)
		expectTextStyle("baz", InlineStyle{Emphasis: true, Link: "/uri"})(t, events)
	},
	// Images — reference, collapsed, shortcut, and edge cases.
	573: expectTextStyle("foo *bar*", InlineStyle{Link: "train.jpg", LinkTitle: "train & tracks"}),
	576: expectTextStyle("foo *bar*", InlineStyle{Link: "train.jpg", LinkTitle: "train & tracks"}),
	577: expectTextStyle("foo *bar*", InlineStyle{Link: "train.jpg", LinkTitle: "train & tracks"}),
	582: expectTextStyle("foo", InlineStyle{Link: "/url"}),
	583: expectTextStyle("foo", InlineStyle{Link: "/url"}),
	584: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	585: expectTextStyle("*foo* bar", InlineStyle{Link: "/url", LinkTitle: "title"}),
	586: expectTextStyle("Foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	587: func(t *testing.T, events []eventView) {
		t.Helper()
		expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
		expectParagraphText("[]")(t, events)
	},
	588: expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	589: expectTextStyle("*foo* bar", InlineStyle{Link: "/url", LinkTitle: "title"}),
	590: expectParagraphText("![[foo]]", "[[foo]]: /url \"title\""),
	591: expectTextStyle("Foo", InlineStyle{Link: "/url", LinkTitle: "title"}),
	592: expectParagraphText("![foo]"),
	// Unicode case fold in reference labels.
	540: expectTextStyle("ẞ", InlineStyle{Link: "/url"}),
	// Link reference definitions — edge cases.
	197: expectParagraphText("[foo]: /url 'title", "with blank line'", "[foo]"),
	199: expectParagraphText("[foo]:", "[foo]"),
	201: expectParagraphText("[foo]: <bar>(baz)", "[foo]"),
	// HTML blocks — types 1-7.
	148: expectBlocks(BlockHTML, 2, BlockParagraph, 1),
	149: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	150: expectBlocks(BlockHTML, 1),
	151: expectBlocks(BlockHTML, 1),
	152: expectBlocks(BlockHTML, 2, BlockParagraph, 1),
	153: expectBlocks(BlockHTML, 1),
	154: expectBlocks(BlockHTML, 1),
	155: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	156: expectBlocks(BlockHTML, 1),
	157: expectBlocks(BlockHTML, 1),
	158: expectBlocks(BlockHTML, 1),
	159: expectBlocks(BlockHTML, 1),
	160: expectBlocks(BlockHTML, 1),
	161: expectBlocks(BlockHTML, 1),
	162: expectBlocks(BlockHTML, 1),
	163: expectBlocks(BlockHTML, 1),
	164: expectBlocks(BlockHTML, 1),
	165: expectBlocks(BlockHTML, 1),
	166: expectBlocks(BlockHTML, 1),
	167: expectBlocks(BlockHTML, 2, BlockParagraph, 1),
	169: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	170: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	171: expectBlocks(BlockHTML, 1),
	172: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	173: expectBlocks(BlockHTML, 1),
	174: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	176: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	177: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	178: expectBlocks(BlockHTML, 1),
	179: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	180: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	181: expectBlocks(BlockHTML, 1),
	182: expectBlocks(BlockHTML, 1, BlockParagraph, 1),
	183: expectBlocks(BlockHTML, 1, BlockIndentedCode, 1),
	184: expectBlocks(BlockHTML, 1, BlockIndentedCode, 1),
	185: expectBlocks(BlockParagraph, 1, BlockHTML, 1),
	186: expectBlocks(BlockHTML, 1),
	188: expectBlocks(BlockHTML, 2, BlockParagraph, 1),
	189: expectBlocks(BlockHTML, 1),
	190: expectBlocks(BlockHTML, 5),
	191: expectBlocks(BlockHTML, 4, BlockIndentedCode, 1),
	// Raw HTML inline — valid tags pass through.
	613: expectParagraphText("<a><bab><c2c>"),
	614: expectParagraphText("<a/><b2/>"),
	617: expectParagraphText("Foo ", "<responsive-image src=\"foo.jpg\" />"),
	623: expectParagraphText("</a></foo >"),
	625: expectParagraphText("foo ", "<!-- this is a --\ncomment - with hyphens -->"),
	627: expectParagraphText("foo ", "<?php echo $a; ?>"),
	628: expectParagraphText("foo ", "<!ELEMENT br EMPTY>"),
	629: expectParagraphText("foo ", "<![CDATA[>&<]]>"),
	630: expectParagraphText("foo ", "<a href=\"&ouml;\">"),
	631: expectParagraphText("foo ", "<a href=\"\\*\">"),
	// List items starting with indented code.
	273: expectBlocks(BlockList, 1, BlockListItem, 1, BlockIndentedCode, 2, BlockParagraph, 1),
	274: expectBlocks(BlockList, 1, BlockListItem, 1, BlockIndentedCode, 2, BlockParagraph, 1),
	// Blockquote + fenced code in list item.
	321: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 2, BlockParagraph, 3, BlockBlockquote, 1, BlockFencedCode, 1)(t, events)
	},
	// Ref def inside list continuation.
	317: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 3),
	// List items starting with blank line.
	278: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 1, BlockFencedCode, 1, BlockIndentedCode, 1),
	// HTML comment separating lists.
	308: expectBlocks(BlockList, 2, BlockListItem, 4, BlockParagraph, 4, BlockHTML, 1),
	// Raw HTML tags shield delimiters from emphasis/link parsing.
	475: expectParagraphText("*<img src=\"foo\" title=\"*\"/>"),
	476: expectParagraphText("**<a href=\"**\">"),
	477: expectParagraphText("__<a href=\"__\">"),
	524: expectParagraphText("[foo <bar attr=\"](baz)\">"),
	536: expectParagraphText("[foo <bar attr=\"][ref]\">"),
	344: expectParagraphText("<a href=\"" + "`" + "\">" + "`"),
	// Hard line breaks don't occur inside HTML tags.
	642: expectParagraphText("<a href=\"foo  \nbar\">"),
	643: expectParagraphText("<a href=\"foo\\\nbar\">"),
	// Multiline emphasis: * at end of line is not right-flanking.
	367: expectParagraphText("*foo bar", "*"),
	// Shortcut ref with leading [.
	559: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("[")(t, events)
		expectTextStyle("foo", InlineStyle{Emphasis: true, Link: "/url", LinkTitle: "title"})(t, events)
		expectTextStyle(" bar", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
		expectParagraphText("]")(t, events)
	},
	// List items — non-continuation examples that already pass.
	253: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockParagraph, 2, BlockIndentedCode, 1, BlockBlockquote, 1)(t, events)
	},
	289: expectBlocks(BlockIndentedCode, 1),
	310: expectBlocks(BlockList, 1, BlockListItem, 7),
	// Sublists.
	323: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 2, BlockListItem, 2)(t, events)
	},
	296: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 2, BlockListItem, 2)(t, events)
	},
	// Tabs in list continuation.
	4: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
	},
	// Indented code blocks: list item takes precedence.
	108: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
	},
	// More list item examples.
	257: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1, BlockIndentedCode, 1)(t, events)
	},
	279: expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1),
	// Sublists on same line as outer marker.
	298: expectBlocks(BlockList, 2, BlockListItem, 2, BlockParagraph, 1),
	299: expectBlocks(BlockList, 3, BlockListItem, 3, BlockParagraph, 1),
	// Sibling sublists.
	326: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 3, BlockListItem, 6, BlockParagraph, 6)(t, events)
	},
	// List items with continuation: paragraph + code + blockquote.
	254: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2, BlockIndentedCode, 1, BlockBlockquote, 1)(t, events)
	},
	263: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 3, BlockFencedCode, 1, BlockBlockquote, 1)(t, events)
	},
	286: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2, BlockIndentedCode, 1, BlockBlockquote, 1)(t, events)
	},
	287: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2, BlockIndentedCode, 1, BlockBlockquote, 1)(t, events)
	},
	288: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2, BlockIndentedCode, 1, BlockBlockquote, 1)(t, events)
	},
	// Blockquote inside list item.
	320: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 2, BlockParagraph, 3, BlockBlockquote, 1)(t, events)
	},
	// Fenced code inside list items.
	318: expectBlocks(BlockList, 1, BlockListItem, 3, BlockFencedCode, 1, BlockParagraph, 2),
	324: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockFencedCode, 1, BlockParagraph, 1)(t, events)
	},
	// Deep sublists — not yet fully correct for 3+ levels.
	// 294 and 307 are deferred until deep nesting is fixed.
	309: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 2, BlockParagraph, 3, BlockHTML, 1)(t, events)
	},
	312: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 4, BlockIndentedCode, 1)(t, events)
	},
	319: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 2, BlockListItem, 3, BlockParagraph, 4)(t, events)
	},
	325: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 2, BlockListItem, 2, BlockParagraph, 3)(t, events)
	},
	// List item continuation after blank lines.
	255: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
		expectParagraphText("two")(t, events)
	},
	256: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
		expectParagraphText("one")(t, events)
		expectParagraphText("two")(t, events)
	},
	262: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
		expectParagraphText("foo")(t, events)
		expectParagraphText("bar")(t, events)
	},
	270: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1, BlockIndentedCode, 1)(t, events)
	},
	271: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1, BlockIndentedCode, 1)(t, events)
	},
	277: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
	},
	276: func(t *testing.T, events []eventView) {
		t.Helper()
		// -    foo\n\n  bar: 2-space indent < content column (5), so bar is outside.
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
	},
	295: expectBlocks(BlockList, 1, BlockListItem, 4),
	297: expectBlocks(BlockList, 2, BlockListItem, 2),
	303: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockParagraph, 3, BlockList, 1, BlockListItem, 2)(t, events)
		expectParagraphText("Foo")(t, events)
	},
	258: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 2)(t, events)
	},
	264: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 1, BlockParagraph, 1, BlockIndentedCode, 1)(t, events)
	},
	306: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 3),
	311: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 3),
	313: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 2, BlockIndentedCode, 1)(t, events)
	},
	314: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 3),
	315: expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 2),
	316: func(t *testing.T, events []eventView) {
		t.Helper()
		expectBlocks(BlockList, 1, BlockListItem, 3, BlockParagraph, 4)(t, events)
		expectParagraphText("c")(t, events)
	},
	593: func(t *testing.T, events []eventView) {
		t.Helper()
		expectParagraphText("!")(t, events)
		expectTextStyle("foo", InlineStyle{Link: "/url", LinkTitle: "title"})(t, events)
	},
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

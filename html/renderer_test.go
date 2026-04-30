package html

import (
	"strings"
	"testing"

	"github.com/codewandler/markdown/stream"
)

func ev(kind stream.EventKind, block stream.BlockKind) stream.Event {
	return stream.Event{Kind: kind, Block: block}
}

func enterDoc() stream.Event  { return ev(stream.EventEnterBlock, stream.BlockDocument) }
func exitDoc() stream.Event   { return ev(stream.EventExitBlock, stream.BlockDocument) }
func enterPara() stream.Event { return ev(stream.EventEnterBlock, stream.BlockParagraph) }
func exitPara() stream.Event  { return ev(stream.EventExitBlock, stream.BlockParagraph) }

func text(s string) stream.Event {
	return stream.Event{Kind: stream.EventText, Text: s}
}

func styledText(s string, style stream.InlineStyle) stream.Event {
	return stream.Event{Kind: stream.EventText, Text: s, Style: style}
}

func heading(level int) stream.Event {
	return stream.Event{Kind: stream.EventEnterBlock, Block: stream.BlockHeading, Level: level}
}

func exitHeading(level int) stream.Event {
	return stream.Event{Kind: stream.EventExitBlock, Block: stream.BlockHeading, Level: level}
}

func TestParagraph(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(), text("Hello, world!"), exitPara(), exitDoc(),
	}
	got, err := RenderString(events)
	if err != nil {
		t.Fatal(err)
	}
	want := "<p>Hello, world!</p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHeadings(t *testing.T) {
	for level := 1; level <= 6; level++ {
		events := []stream.Event{
			enterDoc(), heading(level), text("Title"), exitHeading(level), exitDoc(),
		}
		got, _ := RenderString(events)
		want := "<h" + string(rune('0'+level)) + ">Title</h" + string(rune('0'+level)) + ">\n"
		if got != want {
			t.Errorf("h%d: got %q, want %q", level, got, want)
		}
	}
}

func TestThematicBreak(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockThematicBreak},
		{Kind: stream.EventExitBlock, Block: stream.BlockThematicBreak},
		exitDoc(),
	}
	got, _ := RenderString(events)
	if got != "<hr />\n" {
		t.Errorf("got %q, want %q", got, "<hr />\n")
	}
	got, _ = RenderString(events, WithHTML5())
	if got != "<hr>\n" {
		t.Errorf("html5: got %q, want %q", got, "<hr>\n")
	}
}

func TestBlockquote(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		ev(stream.EventEnterBlock, stream.BlockBlockquote),
		enterPara(), text("quoted"), exitPara(),
		ev(stream.EventExitBlock, stream.BlockBlockquote),
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<blockquote>\n<p>quoted</p>\n</blockquote>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFencedCode(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode, Info: "go"},
		text("package main\n"),
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<pre><code class=\"language-go\">package main\n</code></pre>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFencedCodeNoInfo(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode},
		text("hello\n"),
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<pre><code>hello\n</code></pre>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFencedCodeEscaping(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockFencedCode},
		text("<div>&amp;</div>\n"),
		{Kind: stream.EventExitBlock, Block: stream.BlockFencedCode},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<pre><code>&lt;div&gt;&amp;amp;&lt;/div&gt;\n</code></pre>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIndentedCode(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockIndentedCode},
		text("code line\n"),
		{Kind: stream.EventExitBlock, Block: stream.BlockIndentedCode},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<pre><code>code line\n</code></pre>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUnorderedList(t *testing.T) {
	listData := &stream.ListData{Marker: "-", Tight: true}
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: listData},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: listData},
		enterPara(), text("one"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: listData},
		enterPara(), text("two"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList, List: listData},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<ul>\n<li>one</li>\n<li>two</li>\n</ul>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOrderedList(t *testing.T) {
	listData := &stream.ListData{Ordered: true, Start: 1, Marker: ".", Tight: true}
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: listData},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: listData},
		enterPara(), text("first"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList, List: listData},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<ol>\n<li>first</li>\n</ol>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOrderedListStart(t *testing.T) {
	listData := &stream.ListData{Ordered: true, Start: 3, Marker: ".", Tight: true}
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: listData},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: listData},
		enterPara(), text("third"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList, List: listData},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<ol start=\"3\">\n<li>third</li>\n</ol>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLooseList(t *testing.T) {
	enterList := &stream.ListData{Marker: "-", Tight: true}
	exitList := &stream.ListData{Marker: "-", Tight: false}
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: enterList},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: enterList},
		enterPara(), text("one"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: enterList},
		enterPara(), text("two"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList, List: exitList},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<ul>\n<li>\n<p>one</p>\n</li>\n<li>\n<p>two</p>\n</li>\n</ul>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSoftBreak(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		text("line1"),
		{Kind: stream.EventSoftBreak},
		text("line2"),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p>line1\nline2</p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLineBreak(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		text("line1"),
		{Kind: stream.EventLineBreak},
		text("line2"),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p>line1<br />\nline2</p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	got, _ = RenderString(events, WithHTML5())
	want = "<p>line1<br>\nline2</p>\n"
	if got != want {
		t.Errorf("html5: got %q, want %q", got, want)
	}
}

func TestInlineStrong(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("bold", stream.InlineStyle{Strong: true}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><strong>bold</strong></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineEmphasis(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("italic", stream.InlineStyle{Emphasis: true}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><em>italic</em></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineCode(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("x := 1", stream.InlineStyle{Code: true}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><code>x := 1</code></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineCodeWrappedInEmphasis(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("code", stream.InlineStyle{Code: true, Strong: true, Emphasis: true, EmphasisDepth: 1, StrongDepth: 1}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><em><strong><code>code</code></strong></em></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineStrike(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("gone", stream.InlineStyle{Strike: true}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><del>gone</del></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineLink(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("click", stream.InlineStyle{LinkData: &stream.LinkData{HasLink: true, Link: "https://example.com"}}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><a href=\"https://example.com\">click</a></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineLinkWithTitle(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("click", stream.InlineStyle{LinkData: &stream.LinkData{HasLink: true, Link: "/url", LinkTitle: "a title"}}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><a href=\"/url\" title=\"a title\">click</a></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineImage(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("alt text", stream.InlineStyle{Image: true, LinkData: &stream.LinkData{HasLink: true, Link: "/img.png"}}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><img src=\"/img.png\" alt=\"alt text\" /></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	got, _ = RenderString(events, WithHTML5())
	want = "<p><img src=\"/img.png\" alt=\"alt text\"></p>\n"
	if got != want {
		t.Errorf("html5: got %q, want %q", got, want)
	}
}

func TestInlineImageWithTitle(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("alt", stream.InlineStyle{Image: true, LinkData: &stream.LinkData{HasLink: true, Link: "/img.png", LinkTitle: "my title"}}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><img src=\"/img.png\" alt=\"alt\" title=\"my title\" /></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCombinedStyles(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(),
		styledText("text", stream.InlineStyle{Strong: true, Emphasis: true}),
		exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p><em><strong>text</strong></em></p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHTMLBlockUnsafe(t *testing.T) {
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockHTML},
		text("<div>\nhello\n</div>\n"),
		{Kind: stream.EventExitBlock, Block: stream.BlockHTML},
		exitDoc(),
	}
	got, _ := RenderString(events, WithUnsafe())
	want := "<div>\nhello\n</div>\n"
	if got != want {
		t.Errorf("unsafe: got %q, want %q", got, want)
	}
	got, _ = RenderString(events)
	if strings.Contains(got, "<div>") {
		t.Errorf("safe mode should escape HTML, got %q", got)
	}
}

func TestTextEscaping(t *testing.T) {
	events := []stream.Event{
		enterDoc(), enterPara(), text("a < b & c > d"), exitPara(), exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<p>a &lt; b &amp; c &gt; d</p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEmptyDocument(t *testing.T) {
	got, _ := RenderString(nil)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestTaskListItem(t *testing.T) {
	listData := &stream.ListData{Marker: "-", Tight: true}
	checked := &stream.ListData{Marker: "-", Tight: true, Task: true, Checked: true}
	unchecked := &stream.ListData{Marker: "-", Tight: true, Task: true, Checked: false}
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockList, List: listData},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: unchecked},
		enterPara(), text("todo"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventEnterBlock, Block: stream.BlockListItem, List: checked},
		enterPara(), text("done"), exitPara(),
		{Kind: stream.EventExitBlock, Block: stream.BlockListItem},
		{Kind: stream.EventExitBlock, Block: stream.BlockList, List: listData},
		exitDoc(),
	}
	got, _ := RenderString(events)
	if !strings.Contains(got, "<input disabled=\"\" type=\"checkbox\">") {
		t.Errorf("missing unchecked checkbox in %q", got)
	}
	if !strings.Contains(got, "<input checked=\"\" disabled=\"\" type=\"checkbox\">") {
		t.Errorf("missing checked checkbox in %q", got)
	}
}

func TestTable(t *testing.T) {
	align := []stream.TableAlign{stream.TableAlignNone, stream.TableAlignRight}
	events := []stream.Event{
		enterDoc(),
		{Kind: stream.EventEnterBlock, Block: stream.BlockTable, Table: &stream.TableData{Align: align}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow, TableRow: &stream.TableRowData{Header: true}},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		text("A"),
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		text("B"),
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		text("1"),
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventEnterBlock, Block: stream.BlockTableCell},
		text("2"),
		{Kind: stream.EventExitBlock, Block: stream.BlockTableCell},
		{Kind: stream.EventExitBlock, Block: stream.BlockTableRow},
		{Kind: stream.EventExitBlock, Block: stream.BlockTable},
		exitDoc(),
	}
	got, _ := RenderString(events)
	want := "<table>\n<thead>\n<tr>\n<th>A</th>\n<th align=\"right\">B</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>1</td>\n<td align=\"right\">2</td>\n</tr>\n</tbody>\n</table>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

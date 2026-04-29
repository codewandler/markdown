package html

import (
	"io"
	"strconv"
	"strings"

	"github.com/codewandler/markdown/stream"
)

var (
	headingOpen  = [7]string{"", "<h1>", "<h2>", "<h3>", "<h4>", "<h5>", "<h6>"}
	headingClose = [7]string{"", "</h1>\n", "</h2>\n", "</h3>\n", "</h4>\n", "</h5>\n", "</h6>\n"}
)

// Option configures the HTML renderer.
type Option func(*renderer)

// WithHTML5 produces void elements without the self-closing slash
// (<br>, <hr>, <img>). The default is XHTML style (<br />, <hr />,
// <img ... />), which matches the CommonMark spec test suite.
func WithHTML5() Option {
	return func(r *renderer) { r.html5 = true }
}

// WithUnsafe passes raw HTML blocks through without escaping.
// Required for full CommonMark compliance since spec examples
// contain raw HTML.
func WithUnsafe() Option {
	return func(r *renderer) { r.unsafe = true }
}

// Render writes HTML for the given events to w.
func Render(w io.Writer, events []stream.Event, opts ...Option) error {
	r := newRenderer(w, opts...)
	return r.render(events)
}

// RenderString returns the HTML for the given events as a string.
func RenderString(events []stream.Event, opts ...Option) (string, error) {
	var b strings.Builder
	if err := Render(&b, events, opts...); err != nil {
		return "", err
	}
	return b.String(), nil
}

type renderer struct {
	w            io.Writer
	html5        bool
	unsafe       bool
	tightMap        map[int]bool // EnterBlock list event index -> tight
	tightStack      []bool       // runtime stack for nested lists
	containerDepth  int          // depth of non-list containers (blockquote)
	containerSaved  []int        // saved containerDepth when entering a list
	inHeader     bool         // current table row is a header row
	inCode       bool         // inside fenced_code or indented_code
	inHTML       bool         // inside html block
	headingLevel int          // stashed from enter for exit
	openStyle    stream.InlineStyle // currently open inline tags
	tagStack     []inlineTag        // actual open tag stack for proper nesting
	tableCol     int          // current column index in table row
	tableAlign   []stream.TableAlign
	err          error // sticky error
}

type inlineTagKind int

const (
	tagEm inlineTagKind = iota
	tagStrong
	tagDel
	tagLink
)

type inlineTag struct {
	kind      inlineTagKind
	link      string
	linkTitle string
}

func newRenderer(w io.Writer, opts ...Option) *renderer {
	r := &renderer{w: w}
	for _, o := range opts {
		o(r)
	}
	return r
}

func (r *renderer) render(events []stream.Event) error {
	r.tightMap = prescanTight(events)
	for i, ev := range events {
		if r.err != nil {
			return r.err
		}
		switch ev.Kind {
		case stream.EventEnterBlock:
			r.enterBlock(i, ev, events)
		case stream.EventExitBlock:
			r.closeStyle()
			r.exitBlock(i, ev, events)
		case stream.EventText:
			// Strip trailing spaces from text immediately before
			// a block close or soft break (CommonMark: trailing
			// spaces at end of line are removed for rendering).
			// Skip code spans — their content preserves whitespace.
			if !r.inCode && !r.inHTML && !ev.Style.Code &&
				i+1 < len(events) {
				next := events[i+1]
				if next.Kind == stream.EventSoftBreak ||
					(next.Kind == stream.EventExitBlock &&
						(next.Block == stream.BlockParagraph ||
							next.Block == stream.BlockHeading)) {
					ev.Text = strings.TrimRight(ev.Text, " \t")
				}
			}
			r.text(ev)
		case stream.EventSoftBreak:
			// Close tags that need to close before the newline.
			if ns, ok := r.nextTextStyle(events, i); ok {
				cs := r.closingStyle(ns)
				r.transitionStyle(cs)
			} else if r.openStyle.HasLink {
				r.closeStyle()
			}
			r.write("\n")
		case stream.EventLineBreak:
			if r.inCode || r.inHTML {
				r.write("\n")
			} else {
				r.lineBreak()
			}
		}
	}
	return r.err
}

// prescanTight does a single O(n) pass to find the real Tight value
// for each list. Returns a map from the EnterBlock list event index
// to the Tight bool from the corresponding ExitBlock.
func prescanTight(events []stream.Event) map[int]bool {
	var m map[int]bool
	var stack []int
	for i, ev := range events {
		switch {
		case ev.Kind == stream.EventEnterBlock && ev.Block == stream.BlockList:
			stack = append(stack, i)
		case ev.Kind == stream.EventExitBlock && ev.Block == stream.BlockList:
			if len(stack) > 0 {
				enterIdx := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				tight := true
				if ev.List != nil {
					tight = ev.List.Tight
				}
				if m == nil {
					m = make(map[int]bool)
				}
				m[enterIdx] = tight
			}
		}
	}
	return m
}

func (r *renderer) enterBlock(idx int, ev stream.Event, events []stream.Event) {
	switch ev.Block {
	case stream.BlockDocument:
		// nothing
	case stream.BlockParagraph:
		if r.isTight() {
			return
		}
		r.write("<p>")
	case stream.BlockHeading:
		r.headingLevel = ev.Level
		if ev.Level >= 1 && ev.Level <= 6 {
			r.write(headingOpen[ev.Level])
		}
	case stream.BlockBlockquote:
		r.containerDepth++
		r.write("<blockquote>\n")
	case stream.BlockList:
		tight := r.tightMap[idx]
		r.tightStack = append(r.tightStack, tight)
		r.containerSaved = append(r.containerSaved, r.containerDepth)
		r.containerDepth = 0
		if ev.List != nil && ev.List.Ordered {
			if ev.List.Start != 1 {
				r.write("<ol start=\"" + strconv.Itoa(ev.List.Start) + "\">\n")
			} else {
				r.write("<ol>\n")
			}
		} else {
			r.write("<ul>\n")
		}
	case stream.BlockListItem:
		if ev.List != nil && ev.List.Task {
			if ev.List.Checked {
				r.write("<li><input type=\"checkbox\" checked=\"\" disabled=\"\" /> ")
			} else {
				r.write("<li><input type=\"checkbox\" disabled=\"\" /> ")
			}
		} else if r.listItemNeedsNewline(idx, events) {
			r.write("<li>\n")
		} else {
			r.write("<li>")
		}
	case stream.BlockFencedCode:
		if ev.Info != "" {
			// Info string: use first word as language.
			lang := ev.Info
			if sep := strings.IndexAny(lang, " \t"); sep >= 0 {
				lang = lang[:sep]
			}
			r.write("<pre><code class=\"language-" + escapeHTML(lang) + "\">")
		} else {
			r.write("<pre><code>")
		}
		r.inCode = true
	case stream.BlockIndentedCode:
		r.write("<pre><code>")
		r.inCode = true
	case stream.BlockThematicBreak:
		if r.html5 {
			r.write("<hr>\n")
		} else {
			r.write("<hr />\n")
		}
	case stream.BlockHTML:
		r.inHTML = true
	case stream.BlockTable:
		r.write("<table>\n")
		if ev.Table != nil {
			r.tableAlign = ev.Table.Align
		} else {
			r.tableAlign = nil
		}
	case stream.BlockTableRow:
		r.tableCol = 0
		if ev.TableRow != nil && ev.TableRow.Header {
			r.inHeader = true
			r.write("<thead>\n<tr>\n")
		} else {
			r.write("<tr>\n")
		}
	case stream.BlockTableCell:
		tag := "td"
		if r.inHeader {
			tag = "th"
		}
		align := stream.TableAlignNone
		if r.tableCol < len(r.tableAlign) {
			align = r.tableAlign[r.tableCol]
		}
		switch align {
		case stream.TableAlignLeft:
			r.write("<" + tag + " align=\"left\">")
		case stream.TableAlignCenter:
			r.write("<" + tag + " align=\"center\">")
		case stream.TableAlignRight:
			r.write("<" + tag + " align=\"right\">")
		default:
			r.write("<" + tag + ">")
		}
	}
}

func (r *renderer) exitBlock(idx int, ev stream.Event, events []stream.Event) {
	switch ev.Block {
	case stream.BlockDocument:
		// nothing
	case stream.BlockParagraph:
		if r.isTight() {
			// In tight lists, paragraphs don't get <p> tags.
			// Emit a newline only if the next event opens a block
			// (e.g. a sublist), not if it's </li>.
			if idx+1 < len(events) && events[idx+1].Kind == stream.EventEnterBlock {
				r.write("\n")
			}
			return
		}
		r.write("</p>\n")
	case stream.BlockHeading:
		lvl := ev.Level
		if lvl == 0 {
			lvl = r.headingLevel
		}
		if lvl >= 1 && lvl <= 6 {
			r.write(headingClose[lvl])
		}
		r.headingLevel = 0
	case stream.BlockBlockquote:
		r.containerDepth--
		r.write("</blockquote>\n")
	case stream.BlockList:
		if len(r.tightStack) > 0 {
			r.tightStack = r.tightStack[:len(r.tightStack)-1]
		}
		if len(r.containerSaved) > 0 {
			r.containerDepth = r.containerSaved[len(r.containerSaved)-1]
			r.containerSaved = r.containerSaved[:len(r.containerSaved)-1]
		}
		if ev.List != nil && ev.List.Ordered {
			r.write("</ol>\n")
		} else {
			r.write("</ul>\n")
		}
	case stream.BlockListItem:
		r.write("</li>\n")
	case stream.BlockFencedCode:
		r.write("</code></pre>\n")
		r.inCode = false
	case stream.BlockIndentedCode:
		r.write("</code></pre>\n")
		r.inCode = false
	case stream.BlockThematicBreak:
		// nothing — thematic break is self-closing on enter
	case stream.BlockHTML:
		r.inHTML = false
	case stream.BlockTable:
		r.write("</tbody>\n</table>\n")
		r.tableAlign = nil
	case stream.BlockTableRow:
		if r.inHeader {
			r.write("</tr>\n</thead>\n<tbody>\n")
			r.inHeader = false
		} else {
			r.write("</tr>\n")
		}
	case stream.BlockTableCell:
		if r.inHeader {
			r.write("</th>\n")
		} else {
			r.write("</td>\n")
		}
		r.tableCol++
	}
}

func (r *renderer) text(ev stream.Event) {
	if r.inHTML {
		if r.unsafe {
			r.write(ev.Text)
		} else {
			r.write(escapeHTML(ev.Text))
		}
		return
	}
	if r.inCode {
		r.write(escapeHTML(ev.Text))
		return
	}

	s := ev.Style

	// Inline raw HTML: pass through without escaping.
	// Close any open inline styles first so tags nest correctly.
	if s.RawHTML {
		r.closeStyle()
		r.write(ev.Text)
		return
	}

	// Code spans: emit <code> inline within the current open style.
	// Don't close emphasis/strong — code spans can appear inside them.
	if s.Code {
		r.write("<code>")
		r.write(escapeHTML(ev.Text))
		r.write("</code>")
		return
	}

	// Image: void element, self-contained.
	if s.Image && s.Link != "" {
		r.closeStyle()
		// Image inside a link: wrap in <a>.
		if s.ImageLink != "" {
			r.write("<a href=\"" + escapeAttrURL(s.ImageLink) + "\"")
			if s.ImageLinkTitle != "" {
				r.write(" title=\"" + escapeHTML(s.ImageLinkTitle) + "\"")
			}
			r.write(">")
		}
		r.write("<img src=\"" + escapeAttrURL(s.Link) + "\" alt=\"" + escapeHTML(ev.Text) + "\"")
		if s.LinkTitle != "" {
			r.write(" title=\"" + escapeHTML(s.LinkTitle) + "\"")
		}
		if r.html5 {
			r.write(">")
		} else {
			r.write(" />")
		}
		if s.ImageLink != "" {
			r.write("</a>")
		}
		return
	}

	// Transition inline style tags: close tags no longer active,
	// open tags newly active. This allows emphasis/strong/etc to
	// span across soft breaks and line breaks.
	r.transitionStyle(s)
	r.write(escapeHTML(ev.Text))
}

// transitionStyle manages open inline tags. When the style is
// identical to what's already open, nothing happens (common case
// for emphasis spanning across soft/line breaks). When the style
// changes, close tags that are being removed (innermost first),
// then open tags that are being added (outermost first).
//
// Nesting order (outermost to innermost):
//   strong > em > del > link
func (r *renderer) transitionStyle(s stream.InlineStyle) {
	if r.openStyle == s {
		return
	}

	sEm := s.EmphasisDepth
	if sEm == 0 && s.Emphasis { sEm = 1 }
	sSt := s.StrongDepth
	if sSt == 0 && s.Strong { sSt = 1 }
	sameLink := s.HasLink && r.openStyle.HasLink && s.Link == r.openStyle.Link && s.LinkTitle == r.openStyle.LinkTitle

	// Determine which tags need to be removed.
	needRemove := func(t inlineTag) bool {
		switch t.kind {
		case tagDel:
			return !s.Strike
		case tagEm:
			return false // handled by count
		case tagStrong:
			return false // handled by count
		case tagLink:
			return !s.HasLink || !sameLink
		}
		return false
	}

	// Count current em/strong in stack.
	curEm, curSt := 0, 0
	for _, t := range r.tagStack {
		if t.kind == tagEm { curEm++ }
		if t.kind == tagStrong { curSt++ }
	}

	// Close tags from innermost. We need to close:
	// - any tag that's being removed
	// - excess em/strong tags
	// - any tag that's above a tag being removed (to maintain proper nesting)
	// Find the deepest tag that needs removal.
	removeFrom := len(r.tagStack)
	extraEm := curEm - sEm
	extraSt := curSt - sSt
	seenEm, seenSt := 0, 0
	for i := len(r.tagStack) - 1; i >= 0; i-- {
		t := r.tagStack[i]
		if needRemove(t) {
			removeFrom = i
		}
		if t.kind == tagEm {
			seenEm++
			if seenEm <= extraEm { removeFrom = i }
		}
		if t.kind == tagStrong {
			seenSt++
			if seenSt <= extraSt { removeFrom = i }
		}
	}

	// Close from innermost to removeFrom.
	var reopen []inlineTag
	for i := len(r.tagStack) - 1; i >= removeFrom; i-- {
		r.writeCloseTag(r.tagStack[i])
	}
	// Collect tags to reopen (those above removeFrom that are still wanted).
	remEm, remSt := 0, 0
	for i := removeFrom; i < len(r.tagStack); i++ {
		t := r.tagStack[i]
		keep := false
		switch t.kind {
		case tagEm:
			remEm++
			keep = remEm > extraEm
		case tagStrong:
			remSt++
			keep = remSt > extraSt
		case tagLink:
			keep = s.HasLink && sameLink
		case tagDel:
			keep = s.Strike
		}
		if keep {
			reopen = append(reopen, t)
		}
	}
	r.tagStack = r.tagStack[:removeFrom]
	for _, t := range reopen {
		r.writeOpenTag(t)
		r.tagStack = append(r.tagStack, t)
	}

	// Open new tags.
	if s.HasLink && !sameLink {
		tag := inlineTag{kind: tagLink, link: s.Link, linkTitle: s.LinkTitle}
		r.writeOpenTag(tag)
		r.tagStack = append(r.tagStack, tag)
	}
	newSt := 0
	for _, t := range r.tagStack {
		if t.kind == tagStrong { newSt++ }
	}
	for i := newSt; i < sSt; i++ {
		tag := inlineTag{kind: tagStrong}
		r.writeOpenTag(tag)
		r.tagStack = append(r.tagStack, tag)
	}
	newEm := 0
	for _, t := range r.tagStack {
		if t.kind == tagEm { newEm++ }
	}
	for i := newEm; i < sEm; i++ {
		tag := inlineTag{kind: tagEm}
		r.writeOpenTag(tag)
		r.tagStack = append(r.tagStack, tag)
	}
	if s.Strike && !r.hasTag(tagDel) {
		tag := inlineTag{kind: tagDel}
		r.writeOpenTag(tag)
		r.tagStack = append(r.tagStack, tag)
	}

	r.openStyle = s
}

func (r *renderer) hasTag(kind inlineTagKind) bool {
	for _, t := range r.tagStack {
		if t.kind == kind { return true }
	}
	return false
}

func (r *renderer) writeOpenTag(t inlineTag) {
	switch t.kind {
	case tagEm:
		r.write("<em>")
	case tagStrong:
		r.write("<strong>")
	case tagDel:
		r.write("<del>")
	case tagLink:
		r.write("<a href=\"" + escapeAttrURL(t.link) + "\"")
		if t.linkTitle != "" {
			r.write(" title=\"" + escapeHTML(t.linkTitle) + "\"")
		}
		r.write(">")
	}
}

func (r *renderer) writeCloseTag(t inlineTag) {
	switch t.kind {
	case tagEm:
		r.write("</em>")
	case tagStrong:
		r.write("</strong>")
	case tagDel:
		r.write("</del>")
	case tagLink:
		r.write("</a>")
	}
}

// sameVisualStyle checks if two styles produce the same HTML tags.
// Ignores depth fields — only checks boolean/string fields that affect rendering.
func sameVisualStyle(a, b stream.InlineStyle) bool {
	aEm := a.EmphasisDepth
	if aEm == 0 && a.Emphasis { aEm = 1 }
	bEm := b.EmphasisDepth
	if bEm == 0 && b.Emphasis { bEm = 1 }
	aSt := a.StrongDepth
	if aSt == 0 && a.Strong { aSt = 1 }
	bSt := b.StrongDepth
	if bSt == 0 && b.Strong { bSt = 1 }
	return aEm == bEm && aSt == bSt &&
		a.Strike == b.Strike && a.Code == b.Code &&
		a.HasLink == b.HasLink && a.Link == b.Link && a.LinkTitle == b.LinkTitle &&
		a.Image == b.Image && a.RawHTML == b.RawHTML
}

// closingStyle returns a style that keeps only the tags common to both
// the current open style and the next style. This closes tags being removed
// without opening new ones.
func (r *renderer) closingStyle(next stream.InlineStyle) stream.InlineStyle {
	o := r.openStyle
	var s stream.InlineStyle
	// Keep emphasis/strong at the minimum of current and next depths.
	oEm := o.EmphasisDepth
	if oEm == 0 && o.Emphasis { oEm = 1 }
	nEm := next.EmphasisDepth
	if nEm == 0 && next.Emphasis { nEm = 1 }
	oSt := o.StrongDepth
	if oSt == 0 && o.Strong { oSt = 1 }
	nSt := next.StrongDepth
	if nSt == 0 && next.Strong { nSt = 1 }
	minEm := oEm
	if nEm < minEm { minEm = nEm }
	minSt := oSt
	if nSt < minSt { minSt = nSt }
	s.Emphasis = minEm > 0
	s.EmphasisDepth = minEm
	s.Strong = minSt > 0
	s.StrongDepth = minSt
	s.Strike = o.Strike && next.Strike
	// Always close links at soft breaks — separate links on
	// different lines must not merge.
	return s
}

// nextTextStyle returns the style of the next text event after position i.
func (r *renderer) nextTextStyle(events []stream.Event, i int) (stream.InlineStyle, bool) {
	for j := i + 1; j < len(events); j++ {
		switch events[j].Kind {
		case stream.EventText:
			return events[j].Style, true
		case stream.EventExitBlock:
			return stream.InlineStyle{}, false
		default:
			continue
		}
	}
	return stream.InlineStyle{}, false
}

// nextTextChangesLink checks if the next text event after position i
// has a different link style than the current open style.
func (r *renderer) nextTextChangesLink(events []stream.Event, i int) bool {
	for j := i + 1; j < len(events); j++ {
		switch events[j].Kind {
		case stream.EventText:
			next := events[j].Style
			cur := r.openStyle
			return cur.HasLink != next.HasLink || cur.Link != next.Link || cur.LinkTitle != next.LinkTitle
		case stream.EventExitBlock:
			return true
		default:
			continue
		}
	}
	return false
}

// closeStyle closes all currently open inline style tags.
func (r *renderer) closeStyle() {
	r.transitionStyle(stream.InlineStyle{})
}

func (r *renderer) lineBreak() {
	if r.html5 {
		r.write("<br>\n")
	} else {
		r.write("<br />\n")
	}
}

func (r *renderer) isTight() bool {
	// A paragraph inside a tight list item is suppressed, UNLESS
	// there's a non-list container (blockquote) between the list
	// item and the paragraph.
	return r.containerDepth == 0 && len(r.tightStack) > 0 && r.tightStack[len(r.tightStack)-1]
}

// listItemNeedsNewline returns true if the list item at index idx
// should emit a newline after <li>. This is the case when:
// - the list is loose (items contain <p> tags), OR
// - the item's first child is a block-level element (sublist,
//   blockquote, code block, heading, etc.) rather than inline text.
func (r *renderer) listItemNeedsNewline(idx int, events []stream.Event) bool {
	// Look at the next event after this EnterBlock list_item.
	next := idx + 1
	if next >= len(events) {
		return false
	}
	ev := events[next]
	// Empty item (next event is exit): no newline.
	if ev.Kind == stream.EventExitBlock && ev.Block == stream.BlockListItem {
		return false
	}
	if !r.isTight() {
		return true
	}
	if ev.Kind != stream.EventEnterBlock {
		return false
	}
	// A paragraph in a tight list is suppressed (no <p> tag),
	// so it doesn't need the newline. Any other block does.
	return ev.Block != stream.BlockParagraph
}

func (r *renderer) write(s string) {
	if r.err != nil {
		return
	}
	_, r.err = io.WriteString(r.w, s)
}

// escapeAttrURL percent-encodes a URL then HTML-escapes it for use
// in an HTML attribute (href, src). This ensures & in query strings
// becomes &amp; in the attribute value.
func escapeAttrURL(s string) string {
	return escapeHTML(escapeURL(s))
}

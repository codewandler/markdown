# Plan: HTML Renderer (Event -> HTML)

Status: **ready for implementation**
Created: 2026-04-29
Finalized: 2026-04-29

## Goal

Add an `html` package that converts `stream.Event` slices into
CommonMark-compliant HTML output, and expose it through the root
`markdown` package as a top-level convenience function. This enables:

1. **Compliance testing** -- run the official CommonMark/GFM spec suites
   against our parser by comparing HTML output directly.
2. **Competition pipeline** -- our variants gain a `RenderHTML` adapter,
   putting us on equal footing with goldmark/blackfriday/gomarkdown and
   removing the "event-level" footnote from COMPARISON.md.
3. **Broader utility** -- users who want HTML output from our streaming
   parser get it without goldmark.

## Non-Goals

- This is NOT a full-featured HTML rendering library with themes,
  sanitization, or template support.
- No streaming HTML output (the terminal renderer is the streaming
  path). HTML rendering operates on a complete event slice.
- No CSS class injection or custom attributes (keep it minimal).
- No HTML5 vs XHTML mode toggle in Phase 1 (default to XHTML for spec
  compliance; HTML5 mode can be added in Phase 3 if needed).

## Relationship to Other Plans

- **`design-competition.md`** -- the competition pipeline already has
  `RenderHTML` adapter slots wired for goldmark, blackfriday, and
  gomarkdown (see `competition/candidates.go`). Our variants currently
  have `RenderHTML: nil`. Phase 4 of this plan fills that slot, which
  is what `design-competition.md` Section 2 (Compliance Testing) needs
  to measure our compliance identically to competitors.
- **`roadmap-v1.0.md`** -- insert the HTML renderer at **v0.39.0**,
  before CommonMark gap work. Bump current v0.39.0 (gaps) to v0.40.0
  and shift everything else by one. Rationale: HTML-level verification
  for every gap fix is strictly better signal.
- **AGENTS.md product rule** -- currently says "HTML rendering output
  is out of scope unless explicitly added as a real incremental
  renderer." This plan IS the explicit addition. Once Phase 1 lands,
  update AGENTS.md to reflect that the `html` package exists and is
  in scope.

## Design Principles

1. **Spec-faithful** -- output must match CommonMark 0.31.2 expected
   HTML exactly (modulo whitespace normalization). This is the primary
   acceptance criterion.
2. **Event-driven** -- the renderer walks `[]stream.Event` linearly.
   It must not re-parse Markdown syntax. (Product rule: "Renderer code
   must not parse Markdown syntax.")
3. **Zero dependencies** -- only stdlib + `stream` package.
4. **Testable in isolation** -- given events, produce HTML. No parser
   coupling in the renderer itself.
5. **Consistent API** -- follow the patterns established by
   `markdown.Parse`, `markdown.RenderString`, and `terminal.Renderer`.
6. **Performance** -- minimize allocations. Pre-scan over buffering.
   Propagate parser knowledge via event fields rather than inferring
   it in the renderer.

---

## Resolved Design Decisions

### D1: Tight list — pre-scan the event slice

The parser sets `Tight: true` optimistically on `EnterBlock list`.
The real value is only known at `ExitBlock list` where
`data.Tight = !p.listLoose`. The HTML renderer needs the real value
to decide whether to emit `<p>` tags inside `<li>`.

**Decision:** Pre-scan. Before rendering, do a single O(n) linear
pass over the event slice to build a tight/loose map indexed by the
event index of each `EnterBlock list`. The render pass reads from
the map. No buffering, no parser change, no API leak. The pre-scan
is internal to the renderer — callers don't see it.

**Why not buffer:** Buffering list content until ExitBlock would
require allocating and storing potentially large nested structures.
Pre-scan is O(n) time, O(lists) space.

**Why not fix the parser:** The parser emits `EnterBlock list`
immediately when it sees the first list item. Deferring emission
would break the streaming contract. The tight/loose determination
is inherently retrospective — it's the renderer's job to handle
that when it has the full slice.

### D2: Table thead/tbody — add `Header bool` to table row events

The parser emits `BlockTableRow` for both header and body rows.
HTML needs `<thead><tr><th>` vs `<tbody><tr><td>`.

**Decision:** Add `Header bool` to the `Event` struct (or a new
`TableRowData`). The parser already knows which row is the header —
it emits it from `tryStartTable` after parsing the separator line.
Set `Header: true` on that row's `EnterBlock` event. One bool per
row, zero extra allocations, explicit, benefits all renderers.

**Parser change:** In `emitTableRow`, accept a `header bool`
parameter. `tryStartTable` passes `true` for the header row;
`processActiveTableLine` passes `false`. The `Event` carries the
flag.

**Where to put the field:** Reuse the existing `List *ListData`
pattern — but for tables. Options:

- Add `Header bool` directly to `Event` (simple, but `Header` is
  only meaningful for table rows).
- Add a `TableRowData` struct with `Header bool` and put it on
  `Event` as `TableRow *TableRowData` (consistent with `List`
  and `Table` patterns, nil for non-table-row events).

**Chosen:** `TableRow *TableRowData` with `Header bool`. Consistent
with existing patterns. The pointer is nil for non-table-row events
(zero cost). For table rows, one small allocation per row — but
table rows are rare and this matches the `List *ListData` pattern
the codebase already uses.

### D3: Image — `Image bool` on `InlineStyle`

**Decision:** Add `Image bool` to `InlineStyle`. Minimal, consistent
with how emphasis/strong/strike/code are all flags on text events.
The text content becomes the alt text; `Style.Link` carries the URL.
Zero extra allocations.

**Parser change:** Set `Image: true` in `parseInlineImage`,
`parseReferenceImage`, and `parseInlineImageAsLink`. ~15 minutes.

### D4: Compliance test — root package

**Decision:** Spec compliance test lives in root `markdown_test.go`.
Matches the existing pattern where the root package tests the full
pipeline. No circular dependency issues (root imports `html`,
`stream`, and `internal/commonmarktests`).

### D5: Compliance gate — hard gate from Phase 1

**Decision:** Set exact pass count after first Phase 1 implementation,
then lock it in as a regression gate. Same pattern as `wantCounts` in
`TestCommonMarkCorpusClassification`.

### D6: Roadmap — HTML renderer before CommonMark gaps

**Decision:** Insert at v0.39.0, before gap-closing work. Every gap
fix gets HTML-level verification immediately. The 2-3 session cost
pays for itself in gap-closing confidence.

---

## Public API

### Root package (`markdown.go`)

Top-level convenience functions, matching the existing pattern:

```go
// HTMLString renders Markdown source to an HTML string.
//
//   out, err := markdown.HTMLString("# Hello\n\nWorld")
//   out, err := markdown.HTMLString(src, html.WithUnsafe())
func HTMLString(src string, opts ...html.Option) (string, error)

// HTMLBytes renders Markdown source to HTML bytes.
//
//   out, err := markdown.HTMLBytes([]byte(src))
func HTMLBytes(src []byte, opts ...html.Option) ([]byte, error)
```

### `html` package

```go
package html

import (
    "io"
    "github.com/codewandler/markdown/stream"
)

// Render writes HTML for the given events to w.
func Render(w io.Writer, events []stream.Event, opts ...Option) error

// RenderString returns the HTML as a string.
func RenderString(events []stream.Event, opts ...Option) (string, error)

// Option configures the HTML renderer.
type Option func(*renderer)

// WithHTML5 produces void elements without self-closing slash.
// Default is XHTML, which matches the CommonMark spec test suite.
func WithHTML5() Option

// WithUnsafe passes raw HTML blocks through without escaping.
// Required for full CommonMark compliance.
func WithUnsafe() Option
```

### Implementation sketch

```go
type renderer struct {
    w          io.Writer
    html5      bool
    unsafe     bool
    tightMap   map[int]bool // event index -> tight (from pre-scan)
    listDepth  int
    tightStack []bool       // runtime stack mirroring tightMap lookups
    inHeader   bool         // current table row is header
    err        error        // sticky error
}

func (r *renderer) render(events []stream.Event) error {
    r.tightMap = prescanTight(events)
    for i, ev := range events {
        if r.err != nil {
            return r.err
        }
        switch ev.Kind {
        case stream.EventEnterBlock:
            r.enterBlock(i, ev)
        case stream.EventExitBlock:
            r.exitBlock(ev)
        case stream.EventText:
            r.text(ev)
        case stream.EventSoftBreak:
            r.write("\n")
        case stream.EventLineBreak:
            r.lineBreak()
        }
    }
    return r.err
}

// prescanTight does a single O(n) pass to find the real Tight
// value for each list. Returns a map from the EnterBlock list
// event index to the Tight bool from the corresponding ExitBlock.
func prescanTight(events []stream.Event) map[int]bool {
    m := make(map[int]bool)
    var stack []int // indices of EnterBlock list events
    for i, ev := range events {
        if ev.Kind == stream.EventEnterBlock && ev.Block == stream.BlockList {
            stack = append(stack, i)
        }
        if ev.Kind == stream.EventExitBlock && ev.Block == stream.BlockList {
            if len(stack) > 0 {
                enterIdx := stack[len(stack)-1]
                stack = stack[:len(stack)-1]
                tight := true
                if ev.List != nil {
                    tight = ev.List.Tight
                }
                m[enterIdx] = tight
            }
        }
    }
    return m
}
```

---

## Parser Prerequisites (Pre-Phase 2)

Two small parser changes, both propagating knowledge the parser
already has:

### 1. `Image bool` on `InlineStyle`

```go
// stream/event.go
type InlineStyle struct {
    Emphasis  bool
    Strong    bool
    Strike    bool
    Code      bool
    Link      string
    LinkTitle string
    Image     bool   // true for ![alt](url) and ![alt][ref]
}
```

Set in `parseInlineImage`, `parseReferenceImage`,
`parseInlineImageAsLink`.

### 2. `TableRowData` with `Header bool`

```go
// stream/event.go
type TableRowData struct {
    Header bool
}

// Event gains a new field:
type Event struct {
    Kind     EventKind
    Block    BlockKind
    Text     string
    Style    InlineStyle
    Level    int
    Info     string
    Span     Span
    List     *ListData
    Table    *TableData
    TableRow *TableRowData  // NEW
}
```

In `emitTableRow`, accept `header bool`:

```go
func (p *parser) emitTableRow(text string, line lineInfo, events *[]Event, header bool) {
    rowSpan := Span{Start: line.start, End: line.end}
    var rowData *TableRowData
    if header {
        rowData = &TableRowData{Header: true}
    }
    *events = append(*events, Event{
        Kind:     EventEnterBlock,
        Block:    BlockTableRow,
        Span:     rowSpan,
        TableRow: rowData,
    })
    // ... cells ...
}
```

Call sites:
- `tryStartTable`: `p.emitTableRow(..., true)`
- `processActiveTableLine`: `p.emitTableRow(..., false)`

---

## Event -> HTML Mapping

| Event | HTML Output |
|-------|-------------|
| EnterBlock document | (nothing) |
| ExitBlock document | (nothing) |
| EnterBlock paragraph | `<p>` (suppressed in tight lists) |
| ExitBlock paragraph | `</p>\n` (suppressed in tight lists) |
| EnterBlock heading (level N) | `<hN>` |
| ExitBlock heading (level N) | `</hN>\n` |
| EnterBlock list (ordered, start=1) | `<ol>\n` |
| EnterBlock list (ordered, start=S) | `<ol start="S">\n` |
| EnterBlock list (unordered) | `<ul>\n` |
| ExitBlock list | `</ol>\n` or `</ul>\n` |
| EnterBlock list_item | `<li>` |
| ExitBlock list_item | `</li>\n` |
| EnterBlock list_item (task, checked) | `<li><input type="checkbox" checked="" disabled="" /> ` |
| EnterBlock list_item (task, unchecked) | `<li><input type="checkbox" disabled="" /> ` |
| EnterBlock blockquote | `<blockquote>\n` |
| ExitBlock blockquote | `</blockquote>\n` |
| EnterBlock fenced_code (info=lang) | `<pre><code class="language-lang">` |
| EnterBlock fenced_code (no info) | `<pre><code>` |
| ExitBlock fenced_code | `</code></pre>\n` |
| EnterBlock indented_code | `<pre><code>` |
| ExitBlock indented_code | `</code></pre>\n` |
| EnterBlock thematic_break | `<hr />\n` (or `<hr>\n` with WithHTML5) |
| EnterBlock html | raw passthrough (if WithUnsafe) or escaped |
| EnterBlock table | `<table>\n` |
| EnterBlock table_row (Header) | `<thead>\n<tr>\n` |
| ExitBlock table_row (Header) | `</tr>\n</thead>\n<tbody>\n` |
| EnterBlock table_row (body) | `<tr>\n` |
| ExitBlock table_row (body) | `</tr>\n` |
| EnterBlock table_cell (in header) | `<th>` or `<th align="...">` |
| EnterBlock table_cell (in body) | `<td>` or `<td align="...">` |
| ExitBlock table_cell | `</th>\n` or `</td>\n` |
| ExitBlock table | `</tbody>\n</table>\n` |
| Text (plain) | HTML-escaped text |
| Text (strong) | `<strong>escaped</strong>` |
| Text (emphasis) | `<em>escaped</em>` |
| Text (code) | `<code>escaped</code>` |
| Text (strike) | `<del>escaped</del>` |
| Text (link) | `<a href="url" title="t">escaped</a>` |
| Text (image) | `<img src="url" alt="alt" title="t" />` |
| Text (combined) | nested tags in precedence order |
| SoftBreak | `\n` |
| LineBreak | `<br />\n` |

### Tight Lists

Pre-scan builds `tightMap[enterIdx] = bool`. On `EnterBlock list`,
push `tightMap[i]` onto `tightStack`. On `ExitBlock list`, pop.
On `EnterBlock paragraph`, check top of stack — if tight, skip
`<p>`. On `ExitBlock paragraph`, same.

### Inline Style Nesting

When a text event has multiple styles, wrap outermost first:
1. Link (`<a>`) or Image (`<img>`)
2. Strong (`<strong>`)
3. Emphasis (`<em>`)
4. Strikethrough (`<del>`)
5. Code (`<code>`)

**Code span precedence:** If `Code: true`, ignore other styles.

### HTML Escaping

```go
func escapeHTML(s string) string  // &, <, >, "
func escapeURL(s string) string   // percent-encode, preserve %XX
```

---

## Package Structure

```
html/
    renderer.go       # Render, RenderString, renderer type, block/inline logic
    renderer_test.go  # Unit tests (event slice -> expected HTML)
    escape.go         # escapeHTML, escapeURL
    escape_test.go    # Escape function tests
    doc.go            # Package documentation
```

Same module as root — no separate `go.mod`.

---

## Implementation Phases

### Phase 1: Core blocks + plain text

**Deliverables:**
- Document, paragraph, heading, thematic break
- Fenced code (with info string), indented code
- Blockquote
- Lists (ordered with start, unordered, tight/loose via pre-scan)
- Plain text with HTML escaping
- Soft break, hard line break
- Root package: `HTMLString(src)`, `HTMLBytes(src)`
- `html` package: `Render(w, events)`, `RenderString(events)`
- Compliance test in root package with hard gate

**Acceptance criteria:**
- All Phase 1 block types render correctly
- Tight list paragraph suppression works via pre-scan
- HTML escaping is correct
- Hard compliance gate set (exact number after first impl)
- `go test ./html ./stream ./terminal .` all pass
- No external dependencies

### Phase 2: Inline styles + images

**Prerequisites:** `Image bool` + `TableRowData` parser changes.

**Deliverables:**
- Strong, emphasis, code spans
- Links (href, title, URL escaping)
- Images (src, alt, title)
- Strikethrough (GFM)
- Combined/nested inline styles

**Acceptance criteria:**
- All inline styles render with correct nesting
- Images render as `<img>` with alt/src/title
- URL escaping preserves already-encoded sequences
- Compliance gate updated

### Phase 3: Edge cases + full compliance

**Deliverables:**
- Raw HTML passthrough (`WithUnsafe`)
- Task list items (GFM checkbox)
- Tables (GFM: thead/tbody via `TableRowData.Header`, alignment)
- HTML5 mode (`WithHTML5`)
- Entity passthrough, blank line handling

**Acceptance criteria:**
- Compliance reaches 96%+ (627+/652)
- HTML output matches CommonMark spec exactly (modulo whitespace)

### Phase 4: Competition integration

**Deliverables:**
- Wire `RenderHTML` in `competition/candidates.go`
- Remove event-level footnote from COMPARISON.md

**Acceptance criteria:**
- Our variants have `RenderHTML` adapter
- COMPARISON.md reflects HTML-level compliance

---

## Effort Estimate

| Phase | Effort | Blocker |
|-------|--------|---------|
| Parser changes (Image + TableRowData) | 30 min | None |
| Phase 1 | 1-2 sessions | None |
| Phase 2 | 1-2 sessions | Parser changes |
| Phase 3 | 2-3 sessions | Phase 2 |
| Phase 4 | 1 session | Phase 3 |

Total: ~5-8 focused sessions. Phase 1 can start immediately.

---

## Checklist: What Changes Outside `html/`

| File | Change | Phase |
|------|--------|-------|
| `stream/event.go` | Add `Image bool` to `InlineStyle` | Pre-Phase 2 |
| `stream/event.go` | Add `TableRowData` struct, `TableRow` field on `Event` | Pre-Phase 2 |
| `stream/parser_impl.go` | Set `Image: true` in image parsers | Pre-Phase 2 |
| `stream/parser_impl.go` | Pass `header bool` to `emitTableRow` | Pre-Phase 2 |
| `markdown.go` | Add `HTMLString`, `HTMLBytes` | Phase 1 |
| `markdown_test.go` | Integration + compliance tests | Phase 1 |
| `AGENTS.md` | Update product rule re: HTML renderer | Phase 1 |
| `roadmap-v1.0.md` | Insert v0.39.0 HTML renderer milestone | Phase 1 |
| `competition/candidates.go` | Wire `RenderHTML` for our variants | Phase 4 |
| `COMPARISON.md` | Remove event-level footnote | Phase 4 |

---

## Open Questions

None. All decisions resolved. Ready for implementation.

# AGENTS.md

Repository guidance for Codex and other agents working in this workspace.

## Scope

- Repository: `github.com/codewandler/markdown`
- Goal: production-ready streaming Markdown parsing and terminal rendering
- Main packages: `stream`, `terminal`, `examples/stream-readme`

## Working Rules

- Read the existing code before changing behavior.
- Keep changes narrowly scoped to the requested task.
- Do not revert user changes unless explicitly asked.
- Default to ASCII unless a file already uses another character set.
- Prefer repo-local patterns over new abstractions.

## Product Rules

- The parser must remain append-only and chunk-safe.
- Renderer code must not parse Markdown syntax.
- Custom inline syntax should use `stream.InlineScanner` and `EventInline`, not
  source preprocessing. Scanners must preserve CommonMark precedence: escapes,
  code spans, autolinks, and raw HTML bind before custom atoms.
- Memory usage should stay bounded by unresolved state, not by replaying the
  whole document.
- Terminal rendering is the first-class output path.
- HTML rendering output is out of scope unless explicitly added as a real
  incremental renderer. However, inline raw HTML *parsing* (recognizing HTML
  tags inside Markdown) is in scope — it is required for correct CommonMark
  emphasis, link, and code span precedence.

## Current Feature Shape

- CommonMark-compatible core parsing is the baseline. The corpus test
  (`TestCommonMarkCorpusClassification`) tracks supported/knownGap/unsupported
  counts — every parser change must update these counts.
- GFM support includes tables, task lists, strikethrough, and autolink
  literals.
- Code blocks use Monokai-themed terminal styling.
- The terminal package includes the built-in Go fast path and a small generic
  fallback for non-Go fenced code.
- List items support continuation after blank lines, sublists (via push/pop
  list stack), fenced/indented code, blockquotes, and headings inside items.
- Forward link reference definitions are supported via the `pendingBlocks`
  mechanism — paragraphs defer inline parsing until ref defs are collected.
- Inline extension support includes `stream.InlineScanner`, `EventInline`,
  `InlineData`, and terminal `WithInlineRenderer`/`WithWidthFunc`. `cmd/mdview`
  uses this path for emoji shortcodes so table widths can use `DisplayWidth`.

## CommonMark Compliance Process

When working on CommonMark compliance:

1. **Identify gaps** — use `TestCommonMarkCorpusClassification` to find
   sections with known gaps. Sort by count descending for impact.
2. **Debug with throwaway tests** — write a small `TestDebugXxx` to inspect
   parser output for specific examples. Delete it before committing.
3. **Fix, then scan** — after each fix, scan for *all* newly-passing examples
   across sections (fixes often unlock examples in other sections).
4. **Verify carefully** — block-count heuristics are not enough. Check nesting
   structure, text content, and style attributes. Spot-check with the actual
   spec HTML.
5. **Add assertions** — register each verified example in
   `supportedCommonMarkExamples` with a proper assertion function.
6. **Update counts** — update the `wantCounts` map in
   `TestCommonMarkCorpusClassification` to match the new totals.
7. **Run all three packages** — `go test ./stream ./terminal .` after every
   change. The root package tests exercise the full pipeline.

## Architecture Notes

- **`pendingBlocks`** — closed paragraphs are buffered (not inline-parsed)
  until a non-definition block starts or Flush is called. This allows forward
  link reference definitions to be collected before inline parsing resolves
  references. Any code that opens a new block must drain pending blocks first.
- **List stack** — `pushList`/`popList` save and restore outer list state when
  entering sublists. `closeListItem` closes any open sublists before closing
  the item. When adding sibling items to a sublist, close just the item (not
  the list) to avoid destroying the sublist context.
- **`processListItemContent`** — handles block-level constructs inside list
  items: fenced code, indented code, blockquotes, sublists, ref defs. The
  initial content line goes through `processListItemFirstLine` which also
  checks for these constructs.
- **Inline precedence** — code spans, autolinks, and raw HTML tags bind more
  tightly than emphasis and links. `matchingBracketEnd` skips over code spans
  and HTML tags when scanning for the closing `]` of a link label.
- **Inline scanners** — custom scanners run from `tokenizeInlineReuse` after
  escapes/code/autolinks/raw HTML and before emphasis delimiter handling.
  `TriggerBytes` must be narrow so the plain-text fast path remains cheap.
  Scanner events are represented as `inlineTokenEvent`, inherit emphasis/link
  style during `resolveEmphasisReuse`/`coalesceInlineTokensInto`, and should set
  `InlineData.DisplayWidth` when terminal width differs from byte/rune count.
- **Terminal inline rendering** — table rendering buffers cell events and tracks
  rendered text plus display width. Do not reintroduce width calculation from
  rendered ANSI strings for custom inline atoms.

## File Inventory

Key files by size (lines), for planning read strategies:

| File | Lines | Role |
| --- | ---: | --- |
| `stream/parser_impl.go` | 4,916 | Entire parser: block + inline + scanner hooks |
| `terminal/renderer.go` | 909 | Terminal ANSI renderer |
| `html/renderer.go` | 769 | HTML renderer |
| `stream/event.go` | 247 | Event/Block/Style/LinkData/InlineData types (public API) |
| `stream/parser.go` | 83 | Parser interface + config + InlineScanner API |
| `stream/bench_test.go` | 121 | Parser-only benchmarks |
| `competition/benchmarks/bench_test.go` | 286 | Cross-library comparison benchmarks |

Use `grep` to find functions before reading `parser_impl.go` — don't read
it linearly.

## Performance Context

### Current Gap vs Goldmark (Parse-Only, Spec Input)

| Metric | ours | goldmark | ratio |
| --- | ---: | ---: | ---: |
| Speed | 3.7ms | 1.9ms | 1.9x slower |
| Allocations | 14.0K | 11.4K | 1.2x more |
| Memory | 8.6 MB | 1.7 MB | 5.1x more |

Internal benchmark (this machine, `BenchmarkParserCommonMarkCorpus`):

```
~1.12ms    1.47MB    3,893 allocs
```

### Completed Optimizations (2026-04-30)

10 optimizations applied. See `docs/perf-analysis.md` for the full log
with before/after numbers for each change.

Key changes that future work should be aware of:

- **`parseInline` appends directly** into `*[]Event` instead of returning
  `[]Event`. Same for `coalesceInlineTokensInto` and `coalesceTextInPlace`.
- **Scratch slices on parser struct**: `inlineTokens`, `emphOut`, `tableCells`
  are reused across calls. Don't allocate new slices in the inline pipeline
  without checking if a scratch slice exists.
- **`EventKind`/`BlockKind` are `uint8`**, not `string`. Use `.String()`
  for formatting. Switch/comparison works unchanged.
- **`InlineStyle.LinkData`** is a `*LinkData` pointer, nil for non-link
  events. Use `GetLink()`, `GetHasLink()` etc. accessor methods, or check
  `LinkData != nil` before accessing fields directly.
- **`Event.Inline` is a `*InlineData` pointer**, nil for normal Markdown text.
  Use `EventInline` for custom atoms rather than adding fields to `InlineStyle`
  or rewriting source text.
- **Event struct is 152 bytes** (InlineStyle 24B + *LinkData + 4 pointers +
  Span 48B). InlineStyle is 24 bytes. Don't add fields to these structs without
  measuring impact.
- **No `sort` import** in parser_impl.go. Emphasis style events use
  hand-written insertion sort (slices are 1-4 elements).
- **`containsFold`/`indexFold`** are zero-alloc case-insensitive helpers.
  Use these instead of `strings.ToLower` + `strings.Contains`/`strings.Index`.

### Remaining Hot Paths (CPU profile, post-optimization)

1. **`tokenizeInlineReuse`** — 26% cum. Core inline tokenizer.
2. **`processLine`** — 53% cum. Main dispatch loop.
3. **GC** — 15% cum. Reduced from 34% but still significant.
4. **`nextAutolinkLiteralStart`** — 11% cum. Autolink candidate scanning.
5. **`strings.ToLower`** — 5% flat. Remaining in `unicodeCaseFold`,
   `detectHTMLBlockStart` tag matching.

### Struct Sizes

- `Event`: 152 bytes (InlineStyle 24B + *LinkData + 4 pointers + Span 48B)
- `inlineToken`: ~96 bytes (InlineStyle 24B + delimiter fields + Event for custom inline tokens)
- `InlineStyle`: 24 bytes (6 bools + 2 int16 + *LinkData pointer)
- `LinkData`: 72 bytes (4 strings + 1 bool)

See `docs/perf-analysis.md` for the full optimization log.

### Benchmark Commands

```bash
# Quick parse-only benchmark (internal, ~3s)
go test -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=3 -benchtime=500ms ./stream/

# CPU profile
go test -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=1 -benchtime=2s -cpuprofile=/tmp/cpu.prof ./stream/
go tool pprof -top -cum /tmp/cpu.prof

# Memory profile
go test -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=1 -benchtime=2s -memprofile=/tmp/mem.prof ./stream/
go tool pprof -top -alloc_space /tmp/mem.prof

# Full competition benchmarks (slow, ~5min)
task competition:bench
```

### Performance Rules

- Every optimization must preserve all CommonMark + GFM test results.
- Benchmark before and after every change with `-count=3`.
- Profile before guessing — use `pprof` to confirm the hot path.
- Prefer reducing allocation count over reducing allocation size.
- The streaming Event-slice architecture is a non-negotiable constraint.
  Memory parity with batch parsers (goldmark) is not a goal.

## Verification

Use focused tests first:

```bash
go test ./stream
go test ./terminal
go test .
```

For the example module:

```bash
cd examples/stream-readme && go test ./...
```

If a command needs network access or hits sandbox limits, stop and request the
appropriate escalation instead of working around it.

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

## File Inventory

Key files by size (lines), for planning read strategies:

| File | Lines | Role |
| --- | ---: | --- |
| `stream/parser_impl.go` | 4,579 | Entire parser: block + inline |
| `terminal/renderer.go` | ~750 | Terminal ANSI renderer |
| `html/renderer.go` | ~750 | HTML renderer |
| `stream/event.go` | 113 | Event/Block/Style types (public API) |
| `stream/parser.go` | 47 | Parser interface + config |
| `stream/bench_test.go` | 121 | Parser-only benchmarks |
| `competition/benchmarks/bench_test.go` | 286 | Cross-library comparison benchmarks |

Use `grep` to find functions before reading `parser_impl.go` — don't read
it linearly.

## Performance Context

### Current Gap vs Goldmark (Parse-Only, Spec Input)

| Metric | ours | goldmark | ratio |
| --- | ---: | ---: | ---: |
| Speed | 7.8ms | 1.7ms | 4.6x slower |
| Allocations | 22.3K | 11.4K | 2.0x more |
| Memory | 25.8 MB | 1.7 MB | 15.5x more |

Internal benchmark baseline (this machine, `BenchmarkParserCommonMarkCorpus`):

```
~2.04ms    5.6MB    7,668 allocs
```

### Hot Paths (CPU profile, CommonMark corpus)

1. **`drainPendingBlocks`** — 21% cum. Calls `parseInline` per paragraph,
   uses `append(*events, slice...)` pattern that creates temporary slices.
2. **`tokenizeInline`** — 17% cum. Allocates `[]inlineToken` (160B each).
3. **`resolveEmphasis`** — 4% flat. Allocates new `[]inlineToken` + maps.
4. **`coalesceInlineTokens`** — 5% cum. Allocates `[]Event` (248B each).
5. **GC** — 34% of total CPU. Driven by allocation volume.
6. **`strings.ToLower`** — 4% flat. Called in `normalizeReferenceLabel`.

### Top Allocation Sites (alloc_space)

1. `drainPendingBlocks` — 37% (2,214 MB)
2. `tokenizeInline` — 13% (767 MB)
3. `coalesceInlineTokens` — 13% (767 MB)
4. `processFenceLine` — 10% (599 MB)
5. `processHTMLBlockLine` — 7% (427 MB)
6. `resolveEmphasis` — 5% (321 MB)

### Struct Sizes

- `Event`: 248 bytes (InlineStyle 104B + 3 pointers + 2 strings + Span 48B)
- `inlineToken`: ~160 bytes (InlineStyle copy + delimiter fields)
- `InlineStyle`: 104 bytes (5 strings, 6 bools, 2 ints)

See `docs/perf-analysis.md` for the full optimization plan.

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

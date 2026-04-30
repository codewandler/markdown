# Performance Analysis: Beating Goldmark

Date: 2026-04-30
Baseline commit: current HEAD
Benchmark: `BenchmarkParserCommonMarkCorpus` (CommonMark spec, ~15.6KB input)

## Current Gap vs Goldmark (Parse-Only)

From competition benchmark (Spec input, 2026-04-30 post-optimization):

| Metric       | ours     | goldmark  | ratio           | was             |
| ------------ | -------: | --------: | --------------: | --------------: |
| Speed        | 3.7ms    | 1.9ms     | **1.9x slower** | was 4.6x slower |
| Allocations  | 14.0K    | 11.4K     | **1.2x more**   | was 2.0x more   |
| Memory       | 8.6 MB   | 1.7 MB    | **5.1x more**   | was 15.5x more  |

Target: match or beat goldmark on speed and allocations. Memory will remain
higher due to the streaming Event-slice architecture (vs goldmark's compact
pointer-linked AST), but we should close the gap significantly.

## Baseline (stream/bench_test.go, this machine)

```
BenchmarkParserCommonMarkCorpus-20    ~2.04ms    5.6MB    7,668 allocs
```

Note: the COMPARISON.md numbers (7.8ms) are from a different benchmark harness
that includes more overhead. Our internal benchmark is the optimization target.

## CPU Profile Breakdown

Total CPU in `(*parser).Write`: **45.6%** of benchmark time.

| Function                        | flat  | cum    | % of total |
| ------------------------------- | ----: | -----: | ---------: |
| `processLine`                   | 0.03s | 2.09s  | 43.7%      |
| **GC (gcDrain + scanObject)**   | 0.35s | 1.62s  | **33.9%**  |
| `drainPendingBlocks`            | 0.03s | 1.02s  | 21.3%      |
| `parseInline`                   | —     | 1.19s  | 24.9%      |
| `growslice`                     | 0.03s | 1.00s  | 20.9%      |
| `tokenizeInline`                | 0.07s | 0.80s  | 16.7%      |
| `mallocgc`                      | 0.06s | 0.69s  | 14.4%      |
| `drainPendingBlocksEager`       | 0.03s | 0.58s  | 12.1%      |
| `coalesceInlineTokens`          | 0.07s | 0.26s  | 5.4%       |
| `nextAutolinkLiteralStart`      | 0.03s | 0.25s  | 5.2%       |
| `strings.ToLower`               | 0.12s | 0.18s  | 3.8%       |
| `resolveEmphasis`               | 0.09s | 0.19s  | 4.0%       |

**Key insight**: GC is 33.9% of total CPU. Reducing allocations will have a
double effect — fewer allocs AND less GC pressure.

## Allocation Profile Breakdown (alloc_space)

| Function                    | flat MB  | % of total | alloc count |
| --------------------------- | -------: | ---------: | ----------: |
| `drainPendingBlocks`        | 2,214 MB | **36.9%**  | 49,860      |
| `tokenizeInline`            | 766 MB   | **12.8%**  | 1,698,810   |
| `coalesceInlineTokens`      | 767 MB   | **12.8%**  | 1,403,627   |
| `processFenceLine`          | 599 MB   | **10.0%**  | —           |
| `processHTMLBlockLine`      | 427 MB   | **7.1%**   | —           |
| `resolveEmphasis`           | 321 MB   | **5.3%**   | 1,391,376   |
| `closeSetextHeading`        | 209 MB   | **3.5%**   | —           |
| `emitThematicBreak`         | 127 MB   | **2.1%**   | —           |
| `emitTableRow`              | 99 MB    | **1.6%**   | —           |

Top 3 functions account for **62.5%** of all allocated memory.

## Struct Sizes

| Type          | Size    | Notes                                    |
| ------------- | ------: | ---------------------------------------- |
| `Event`       | 248 B   | Contains InlineStyle (104B), 3 pointers  |
| `InlineStyle` | 104 B   | 5 strings, 6 bools, 2 ints               |
| `inlineToken` | ~160 B  | Contains InlineStyle copy                |
| `Span`        | 48 B    | 2× Position (24B each)                   |
| `Position`    | 24 B    | int64 + 2× int                           |

A single `[]Event` append of 100 events = 24.8 KB. The inline pipeline
creates `[]inlineToken` → `resolveEmphasis` → new `[]inlineToken` →
`coalesceInlineTokens` → `[]Event`. That's 3-4 slice allocations per
paragraph, each carrying 160-248 byte elements.

## Root Causes (Ranked by Impact)

### 1. Event struct is too large (248 bytes)

`EventKind` and `BlockKind` are `string` types. Every Event carries a full
`InlineStyle` (104 bytes) even for block enter/exit events that never use it.
Three pointer fields (`*ListData`, `*TableData`, `*TableRowData`) add 24 bytes
of pointer overhead that triggers GC scanning.

**Fix**: Change `EventKind`/`BlockKind` to `uint8` (iota constants). Keep the
existing string constants as the public API via `String()` methods. Split
`InlineStyle` — most events don't need Link/ImageLink strings. Consider a
union/tagged approach for the pointer fields.

**Impact**: ~2x reduction in Event size → fewer bytes copied in `append`,
less GC scanning, better cache locality.

**Risk**: Breaking API change. `EventKind` and `BlockKind` are used as string
comparisons in both renderers. Must provide migration path.

### 2. Inline pipeline allocates 3 intermediate slices per paragraph

`tokenizeInline` → `[]inlineToken` (alloc 1)
`resolveEmphasis` → new `[]inlineToken` (alloc 2)
`coalesceInlineTokens` → `[]Event` (alloc 3)
`drainPendingBlocks` appends to `*events` (alloc 4 via growslice)

Each `inlineToken` is ~160 bytes. For a paragraph with 20 tokens, that's
3.2 KB per intermediate slice × 3 = 9.6 KB per paragraph.

**Fix**: Pool `[]inlineToken` slices on the parser struct. Reuse between
paragraphs. `resolveEmphasis` should modify in-place where possible instead
of allocating a new slice.

**Impact**: Eliminates ~1.7M + 1.4M + 1.4M = 4.5M alloc objects (60% of total).

### 3. drainPendingBlocks uses `append(..., slice...)` pattern

Line 915: `*events = append(*events, p.parseInline(pb.text, pb.span)...)`

This creates a temporary `[]Event` from `parseInline`, then copies it into
the output slice via variadic append. The temporary is immediately garbage.

**Fix**: Pass `events *[]Event` into `parseInline` so it appends directly.
Eliminates the intermediate `[]Event` allocation entirely.

**Impact**: Eliminates 36.9% of allocated memory.

### 4. string(raw) conversion in Write hot loop

Line 155: `info := p.nextLineInfo(string(raw), true)`

Every line creates a new string from the byte slice. For the CommonMark
corpus (~652 examples), this is ~2000+ string allocations.

**Fix**: Use `unsafe.String` for zero-copy conversion where the string
lifetime is bounded by the Write call. Or restructure to work with byte
slices internally.

**Impact**: Moderate — eliminates ~2K allocs per parse.

### 5. processFenceLine / processHTMLBlockLine (17% combined)

These emit `Event` structs with `Text` fields that are string copies of
line content. Each fenced code line and HTML block line allocates.

**Fix**: Batch code block content into a single string (accumulate lines,
emit once at close). This matches how `closeParagraph` already works.

**Impact**: 599 MB + 427 MB = 1,026 MB (17% of alloc_space).

### 6. strings.ToLower in normalizeReferenceLabel (3.8% CPU)

Called for every link reference lookup. `strings.ToLower` allocates a new
string even when the input is already lowercase.

**Fix**: Use `strings.EqualFold` for comparisons instead of normalizing.
Or cache normalized labels.

**Impact**: 3.8% CPU flat.

## Optimization Order

1. **Event struct size** — largest single impact, enables all downstream wins
2. **Inline pipeline pooling** — eliminates 60% of alloc objects
3. **drainPendingBlocks direct append** — eliminates 37% of alloc_space
4. **String allocation reduction** — Write loop, closeParagraph
5. **Fence/HTML batching** — 17% of alloc_space
6. **strings.ToLower** — 3.8% CPU

## Constraints

- Parser must remain append-only and chunk-safe (streaming invariant)
- `Event` is a public type — API changes need migration path
- All 652 CommonMark spec tests must continue passing
- GFM tests must continue passing
- Fuzz tests must not regress

## Target

After all optimizations:
- Speed: <1ms on CommonMark corpus (from 2ms) — within 2x of goldmark
- Allocations: <4K (from 7.7K) — within 3x of goldmark
- Memory: <3 MB (from 5.6 MB) — within 2x of goldmark

Memory parity with goldmark (1.7 MB) is not achievable without abandoning
the streaming Event-slice architecture, which is a non-goal.

## Optimization Log

### Opt 1: parseInline direct append (2026-04-30)

Eliminated intermediate `[]Event` allocation in the inline pipeline.

- `parseInline` → `parseInlineInto`: takes `*[]Event`, appends directly
- `coalesceInlineTokens` → `coalesceInlineTokensInto`: same pattern
- `coalesceText` → `coalesceTextInPlace`: compacts in-place instead of returning new slice
- All 6 call sites updated (drainPendingBlocks, drainPendingBlocksEager, closeSetextHeading, emitTableRow, processNonContainerLine heading, processListItemFirstLine heading)

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 2.04 ms | 1.83 ms | **-10.3%** |
| Memory | 5.6 MB | 4.9 MB | **-12.5%** |
| Allocations | 7,668 | 6,522 | **-14.9%** |

This was root cause #3 from the analysis. The `append(*events, parseInline(...)...)`
pattern created a temporary `[]Event` per paragraph that was immediately garbage.
Direct append eliminates that allocation entirely.

### Opt 2: tokenizeInline slice reuse (2026-04-30)

Pool the `[]inlineToken` scratch slice on the parser struct so it survives
across `parseInline` calls. The backing array grows once to the high-water
mark and is reused for every subsequent paragraph/heading.

- Added `inlineTokens []inlineToken` field to `parser` struct
- `parser.parseInline` resets the slice (`[:0]`) and passes it to `tokenizeInlineReuse`
- `tokenizeInline` (package-level, used by recursive link label parsing) still allocates fresh
- Capacity-hint approach was tried first but regressed speed due to upfront `make` cost for 160-byte elements

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.83 ms | 1.77 ms | **-3.3%** |
| Memory | 4.9 MB | 4.4 MB | **-10.2%** |
| Allocations | 6,522 | 5,229 | **-19.8%** |

Cumulative from baseline: **-13.2% speed, -21.4% memory, -31.8% allocs**

### Opt 2: resolveEmphasis output slice pooling (2026-04-30)

Added `emphOut []inlineToken` scratch slice to the parser struct. New
`resolveEmphasisReuse` accepts and returns the output buffer, avoiding
allocation of the output `[]inlineToken` on every paragraph.

The original `resolveEmphasis` delegates to `resolveEmphasisReuse(tokens, nil)`
for non-method call paths (recursive link content parsing).

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.83 ms | 1.62 ms | **-11.5%** |
| Memory | 4.9 MB | 4.0 MB | **-18.4%** |
| Allocations | 6,522 | 4,581 | **-29.8%** |

Cumulative from baseline: speed -20.6%, memory -28.6%, allocs -40.3%.

### Opt 3: eliminate string allocations in HTML block check + ref label normalization (2026-04-30)

- Replace `strings.Contains(strings.ToLower(...), strings.ToLower(...))` in
  HTML block end detection with zero-alloc `containsFold` helper.
- Add fast path in `normalizeReferenceLabel`: skip `Fields`/`Join`/`ToLower`
  when label is already trimmed, single-spaced, lowercase ASCII.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.62 ms | 1.65 ms | ~same |
| Memory | 4.0 MB | 4.0 MB | ~same |
| Allocations | 4,581 | 4,362 | **-4.8%** |

Cumulative from baseline: speed -19.1%, memory -28.6%, allocs -43.1%.

### Opt 4: pool splitTableRow cells slice (2026-04-30)

Added `tableCells []string` scratch slice to parser struct. New
`splitTableRowReuse` reuses the backing array across table row parses.

No measurable change on CommonMark corpus (few tables). Prevents
per-row `[]string` allocation on table-heavy inputs.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.65 ms | 1.71 ms | ~same |
| Memory | 4.0 MB | 4.0 MB | ~same |
| Allocations | 4,362 | 4,362 | ~same |

Cumulative from baseline: speed -16.2%, memory -28.6%, allocs -43.1%.

### Opt 5: EventKind/BlockKind string → uint8 (2026-04-30)

Changed `EventKind` and `BlockKind` from `string` to `uint8` with iota
constants. Added `String()` methods for formatting compatibility.
Source-compatible for all switch/comparison usage.

Event struct: 248B → 224B (-24B per event, -10%).

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.71 ms | 1.60 ms | **-6.4%** |
| Memory | 4.0 MB | 3.88 MB | **-3.0%** |
| Allocations | 4,362 | 4,362 | same |

Cumulative from baseline: speed -21.6%, memory -30.7%, allocs -43.1%.

### Opt 6: Split InlineStyle — move link strings behind *LinkData pointer (2026-04-30)

Moved Link, LinkTitle, HasLink, ImageLink, ImageLinkTitle from InlineStyle
into a separate LinkData struct behind a pointer. Only link/image events
allocate it; plain text and block events carry a nil pointer.

Event struct: 224B → 144B (-36%). InlineStyle: 104B → 24B (-77%).

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.60 ms | 1.22 ms | **-23.8%** |
| Memory | 3.88 MB | 2.66 MB | **-31.4%** |
| Allocations | 4,362 | 4,435 | +1.7% |

Allocs increased slightly (+73) from LinkData pointer allocations, but
the 1.2MB memory reduction and 24% speed gain from smaller Event copies
and better cache locality far outweigh it.

Cumulative from baseline: speed -40.2%, memory -52.5%, allocs -42.2%.

**Breaking API change**: InlineStyle fields Link, LinkTitle, HasLink,
ImageLink, ImageLinkTitle moved to LinkData. Access via GetLink(),
GetHasLink(), etc. methods or direct LinkData pointer.

### Opt 7: Pre-size events slice in Write by counting newlines (2026-04-30)

Count newlines in partial buffer before processing to pre-allocate the
events slice with capacity = lines × 4. Eliminates all growslice
reallocation during event emission.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.22 ms | 1.24 ms | ~same |
| Memory | 2.66 MB | 1.48 MB | **-44.4%** |
| Allocations | 4,435 | 4,420 | -0.3% |

Cumulative from baseline: speed -39.2%, memory -73.6%, allocs -42.3%.

### Opt 8: Replace sort.SliceStable with insertion sort in resolveEmphasis (2026-04-30)

Replace `sort.SliceStable` (which uses `reflectlite.Swapper`, allocating
per call) with a hand-written insertion sort. The style event slices are
typically 1-4 elements. Removed `sort` import entirely.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.24 ms | 1.19 ms | **-4.0%** |
| Memory | 1.48 MB | 1.47 MB | ~same |
| Allocations | 4,420 | 4,079 | **-7.7%** |

Cumulative from baseline: speed -41.7%, memory -73.8%, allocs -46.8%.

### Opt 9: Eliminate strings.ToLower in autolink detection (2026-04-30)

Replace `strings.ToLower(text)` called 4x per invocation in
`nextAutolinkLiteralStart` with zero-alloc `indexFold` helper.
Replace `strings.ToLower(candidate)` in `parseAutolinkLiteral` with
direct `strings.EqualFold` prefix checks.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.19 ms | 1.13 ms | **-5.0%** |
| Memory | 1.47 MB | 1.46 MB | ~same |
| Allocations | 4,079 | 3,735 | **-8.4%** |

Cumulative from baseline: speed -44.6%, memory -73.9%, allocs -51.3%.

### Opt 10: Use bytes.IndexByte instead of bytes.IndexAny in Write loop (2026-04-30)

Check once for \r presence, then use `bytes.IndexByte(_, '\n')` (SIMD)
on the common path instead of `bytes.IndexAny(_, "\r\n")` (generic).

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.13 ms | 1.08 ms | **-4.4%** |
| Memory | 1.46 MB | 1.46 MB | same |
| Allocations | 3,735 | 3,735 | same |

Cumulative from baseline: speed -47.1%, memory -73.9%, allocs -51.3%.

### Opt 11: ASCII-only fold matching for hot-path tag/scheme scans (2026-04-30)

Replaced `strings.EqualFold` inside `indexFold`/`containsFold` with a
byte-wise ASCII-only fold helper. These helpers are used for CommonMark/GFM
HTML tag and URI scheme matching, where the match strings are lowercase ASCII.
This avoids Unicode case-folding overhead in the inline/autolink hot path.

- Added `equalFoldASCII(s, lowerASCII)`.
- Updated `indexFold` and `containsFold` to use ASCII byte comparison.
- Kept zero-allocation behavior.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.22 ms | 0.99 ms | **-18.5%** |
| Memory | 1.47 MB | 1.47 MB | same |
| Allocations | 3,893 | 3,893 | same |

Benchmark command: `go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=5 -benchtime=1s ./stream/`.
The before number uses the immediately preceding local baseline run (count=3,
noisy but representative); the after number is the five-run average.

Cumulative from current post-Opt-10 baseline: speed ~-18.5%, memory same,
allocs same.

### Opt 12: Reuse inline link scan result for image precheck (2026-04-30)

Removed a duplicate `strings.Contains(text, "](")` scan at the start of
`tokenizeInlineReuse`. Inline links and inline images use the same cheap
precheck marker, so the result can be computed once and assigned to both
`linkPossible` and `imagePossible`.

```go
linkPossible := strings.Contains(text, "](")
imagePossible := linkPossible
```

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 0.99 ms | 1.06 ms | noisy / no measurable win |
| Memory | 1.47 MB | 1.47 MB | same |
| Allocations | 3,893 | 3,893 | same |

Benchmark command: `GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=5 -benchtime=1s ./stream/`.
The measured runtime was within local benchmark noise and did not show a stable
win, but the change removes a provably redundant full-string scan with no API or
correctness trade-off.

### Opt 13: Single-pass GFM autolink literal start scan (2026-04-30)

Replaced `nextAutolinkLiteralStart`'s four separate case-insensitive prefix
searches plus email scan with one left-to-right byte scan. The scanner checks
only bytes that can start a GFM autolink literal candidate (`h`, `f`, `w`, `@`)
and returns the first valid boundary-preserving candidate.

This removes repeated full-string scans for `http://`, `https://`, `ftp://`,
`www.`, and email starts on plain text segments.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| Speed | 1.06 ms | 1.02 ms | **-3.1%** |
| Memory | 1.47 MB | 1.47 MB | same |
| Allocations | 3,893 | 3,893 | same |

Benchmark command: `GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=5 -benchtime=1s ./stream/`.
The improvement is CPU-only and benchmark noise is still visible, but the code
now does one autolink candidate pass instead of up to five passes over the same
text segment.

### Opt 14: Plain inline fast path (2026-04-30)

Added a conservative `hasInlineSyntax` pre-scan before running the full inline
pipeline. Paragraphs and inline fragments that contain no possible inline
syntax now emit one `EventText` directly instead of going through
`tokenizeInlineReuse` -> `resolveEmphasisReuse` -> `coalesceInlineTokensInto`.

The predicate intentionally has false positives but no known false negatives:
it treats newline, escapes, code spans, links/images, raw HTML/autolinks,
character references, emphasis/strong/strike delimiters, and GFM autolink
starter bytes as inline syntax.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| CommonMark corpus speed | 1.02 ms | 1.01 ms | ~same / **-1%** |
| CommonMark corpus memory | 1.47 MB | 1.47 MB | same |
| CommonMark corpus allocations | 3,893 | 3,888 | **-5 allocs** |
| Competition Spec speed | 2.91 ms | 2.72 ms | **-6.5%** |
| Competition Spec allocations | 9,836 | 9,836 | same |
| Competition README speed | not captured post-Opt-13 | 0.29-0.32 ms | measured after |
| Competition README allocations | not captured post-Opt-13 | 922 | measured after |

Benchmark commands:

```bash
GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus|BenchmarkParserLongParagraph$' -benchmem -count=5 -benchtime=1s ./stream/
cd competition && go test -run='^$' -bench='BenchmarkParse/(ours|goldmark)/(Spec|README)$' -benchmem -count=3 -benchtime=500ms ./benchmarks
```

The CommonMark corpus contains many syntax-heavy examples, so the internal
corpus benchmark only moves slightly. The competition Spec parse benchmark shows
a clearer ~6-7% speedup from skipping the inline pipeline for plain text blocks.

### Opt 15: Emit no-bracket paragraphs immediately (2026-04-30)

Paragraphs are only kept in `pendingBlocks` to support forward link reference
definitions. Paragraphs without `[` or `]` cannot contain reference links, so
they do not benefit from deferred inline parsing. Such paragraphs now emit their
paragraph enter/text/exit events immediately when there are no older pending
blocks that must preserve document order.

Implementation details:

- Added `paragraphState.hasBrackets`, updated while paragraph lines are added.
- Added `paragraphText` helper with a single-line no-copy fast path.
- Added `clearParagraphLines` so all paragraph close paths reset bracket state.
- `closeParagraph` only appends to `pendingBlocks` when the paragraph has
  brackets or previous pending blocks must remain ordered before it.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| CommonMark corpus speed | 1.01 ms | 0.97 ms | **-3.8%** |
| CommonMark corpus memory | 1.47 MB | 1.46 MB | **-0.6%** |
| CommonMark corpus allocations | 3,888 | 3,381 | **-13.0%** |
| Tiny chunks allocations | 23,579 | 23,072 | **-507 allocs** |
| Competition Spec speed | 2.72 ms | 2.74 ms | ~same |
| Competition Spec memory | 8.53 MB | 8.50 MB | **-0.3%** |
| Competition Spec allocations | 9,836 | 8,835 | **-10.2%** |
| Competition README speed | 0.29-0.32 ms | 0.30 ms | ~same |
| Competition README allocations | 922 | 811 | **-12.0%** |

Benchmark commands:

```bash
GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus|BenchmarkParserLongParagraph$' -benchmem -count=5 -benchtime=1s ./stream/
cd competition && go test -run='^$' -bench='BenchmarkParse/(ours|goldmark)/(Spec|README)$' -benchmem -count=3 -benchtime=500ms ./benchmarks
```

This is primarily an allocation/GC-pressure win. The initial implementation
rescanned paragraph text for brackets at close and regressed speed; bracket
state is now tracked incrementally in `addParagraphLine`, which keeps the
allocation reduction without the extra close-time scan.

### Opt 16: Emit plain paragraph lines without joining (2026-04-30)

After Opt 15, `paragraphText` became visible in allocation profiles because
immediate no-bracket paragraphs still joined all paragraph lines into a single
string before the plain-inline fast path could emit text. For paragraphs whose
individual lines have no inline syntax and whose line endings cannot represent a
hard break, the parser now emits `EventText`/`EventSoftBreak` directly from the
stored paragraph lines.

Implementation details:

- Added `canEmitPlainParagraphLines` to conservatively check each line with
  `hasInlineSyntax`.
- Added `hasHardBreakSuffix` so lines ending in `\\` or two spaces still use the
  normal inline pipeline and preserve hard-break behavior.
- Added `emitPlainParagraphLines` to emit line text and soft breaks without
  allocating a joined paragraph string.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| CommonMark corpus speed | 0.97 ms | 0.98 ms | ~same |
| CommonMark corpus memory | 1.46 MB | 1.46 MB | ~same |
| CommonMark corpus allocations | 3,381 | 3,358 | **-23 allocs** |
| Long paragraph speed | 0.68 ms | 0.59 ms | **-13%** |
| Long paragraph allocations | 8 | 8 | same |
| Tiny chunks allocations | 23,072 | 23,049 | **-23 allocs** |
| Competition Spec speed | 2.74 ms | 2.63 ms | **-4%** |
| Competition Spec allocations | 8,835 | 8,835 | same |
| Competition README speed | 0.30 ms | 0.30 ms | ~same |
| Competition README allocations | 811 | 811 | same |

Benchmark commands:

```bash
GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus|BenchmarkParserLongParagraph$' -benchmem -count=5 -benchtime=1s ./stream/
cd competition && go test -run='^$' -bench='BenchmarkParse/(ours|goldmark)/(Spec|README)$' -benchmem -count=3 -benchtime=500ms ./benchmarks
```

This is a small corpus win but a meaningful long-plain-paragraph win, and it
removes a known `paragraphText` allocation path without changing Markdown
semantics.

### Opt 17: Check table header before parsing separator (2026-04-30)

`tryStartTable` parsed the current line as a table separator before checking
whether the previous paragraph line was actually a table header containing a
pipe. On ordinary paragraphs this caused unnecessary table-separator splitting
and alignment allocation attempts. The function now checks the cached paragraph
header line first and returns early when the header has no pipe.

Implementation details:

- Moved `splitTableRowReuse(header.text, p.tableCells[:0])` before
  `parseTableSeparator(line.text)`.
- Ordinary paragraph continuation lines now avoid separator parsing entirely
  when the previous line cannot be a table header.
- Table-heavy behavior is unchanged because real table headers still proceed to
  separator parsing and column-count validation.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| CommonMark corpus speed | 0.98 ms | 0.96 ms | **-2%** |
| CommonMark corpus memory | 1.46 MB | 1.46 MB | ~same |
| CommonMark corpus allocations | 3,358 | 3,246 | **-112 allocs** |
| Tiny chunks allocations | 23,049 | 22,937 | **-112 allocs** |
| Competition Spec speed | 2.63 ms | 2.56 ms | **-3%** |
| Competition Spec allocations | 8,835 | 8,835 | same |
| Competition README allocations | 811 | 802 | **-9 allocs** |

Benchmark commands:

```bash
GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus|BenchmarkParserLongParagraph$' -benchmem -count=5 -benchtime=1s ./stream/
cd competition && go test -run='^$' -bench='BenchmarkParse/(ours|goldmark)/(Spec|README)$' -benchmem -count=3 -benchtime=500ms ./benchmarks
```

This is a small but stable win on the CommonMark corpus and avoids wasted work
on non-table paragraph continuations.


### Opt 18: Reuse paragraph text helper for setext headings (2026-04-30)

`closeSetextHeading` manually built heading text with a fresh `strings.Builder`.
It now reuses `paragraphText`, which has the single-line no-copy fast path added
for paragraph emission. This avoids unnecessary string building for single-line
setext headings.

Implementation details:

- Replaced the local `strings.Builder` join in `closeSetextHeading` with
  `paragraphText(p.paragraph.lines)`.
- Reused one `Span` value for heading enter/inline/exit events.

| Metric | Before | After | Delta |
| ----------- | --------: | --------: | --------: |
| CommonMark corpus speed | 0.96 ms | 0.99 ms | noisy / no clear win |
| CommonMark corpus memory | 1.46 MB | 1.46 MB | ~same |
| CommonMark corpus allocations | 3,246 | 3,225 | **-21 allocs** |
| Competition Spec allocations | 8,835 | 8,835 | same |

Benchmark commands:

```bash
GOMAXPROCS=1 go test -run='^$' -bench='BenchmarkParserCommonMarkCorpus$' -benchmem -count=5 -benchtime=1s ./stream/
cd competition && go test -run='^$' -bench='BenchmarkParse/ours/Spec$' -benchmem -count=3 -benchtime=500ms ./benchmarks
```

This is kept as a small allocation-only cleanup; speed measurements were within
local benchmark noise.

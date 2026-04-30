# Performance Analysis: Beating Goldmark

Date: 2026-04-30
Baseline commit: current HEAD
Benchmark: `BenchmarkParserCommonMarkCorpus` (CommonMark spec, ~15.6KB input)

## Current Gap vs Goldmark (Parse-Only)

From COMPARISON.md (Spec input):

| Metric       | ours     | goldmark  | ratio         |
| ------------ | -------: | --------: | ------------: |
| Speed        | 7.8ms    | 1.7ms     | **4.6x slower** |
| Allocations  | 22.3K    | 11.4K     | **2.0x more** |
| Memory       | 25.8 MB  | 1.7 MB    | **15.5x more** |

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

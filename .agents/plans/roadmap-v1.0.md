# Roadmap: v1.0+ — Hardening, Demo, Documentation

Status: **in progress**
Created: 2026-04-28
Baseline: v0.35.0 (CommonMark 96.2%, GFM 100% valid, zero failures)

---

## 1. Fuzz Testing

**Priority: highest — production readiness gate**

Add Go native fuzz tests (`testing.F`) for the streaming parser to find
crashes, hangs, and panics with random/malformed input.

### Tasks

- [x] Add `FuzzParser` in `stream/fuzz_test.go` using `testing.F`
- [x] Seed with CommonMark + GFM corpus examples (652 + 672 = 1324 seeds)
- [x] Seed with pathological inputs: deeply nested lists/blockquotes,
      long delimiter runs, huge lines, empty input, binary data (40+ seeds)
- [x] Add `FuzzParserChunkBoundary` that splits input at random positions
      and verifies output equivalence
- [x] Add `FuzzParserMultiChunk` that splits input at multiple random
      positions and verifies output equivalence
- [x] Run fuzz for 5+ minutes, fix any findings
- [x] Add malformed UTF-8 seeds
- [ ] Verify bounded memory: no input causes unbounded allocation

### Findings fixed

1. **`closeBlockquote` didn't close nested lists** — when a blockquote
   contained nested lists (e.g. `>* *`), only the innermost list was
   closed. Fixed by looping over all lists in `closeBlockquote`.
2. **`closeListItem` emitted duplicate exit after `closeBlockquote`** —
   when `closeBlockquote` already closed the list item (because the list
   was inside the blockquote), `closeListItem` emitted a second exit.
   Fixed by checking `p.inListItem` after `closeBlockquote` returns.
3. **`parseTableAlign` panicked on single colon** — input like `|:` caused
   a slice bounds panic. Fixed by guarding against empty cell after
   stripping the left colon.
4. **`closeListItem`/`closeList` closed blockquote from wrong direction** —
   both functions unconditionally called `closeBlockquote`, even when the
   list was inside the blockquote (not the other way around). Fixed by
   checking `bqInsideListItem` before closing.
5. **Non-`>` lines inside blockquote treated as list continuation** —
   when a blockquote contained a list, subsequent non-`>` lines were
   matched as list item continuation before the blockquote could close.
   Fixed by closing the blockquote before checking list continuation.

### Definition of done

- [x] Fuzz runs for 20+ seconds per target with zero crashes (~3M total execs)
- [x] All findings fixed and regression-tested
- [ ] Extended 10-minute run (requires manual execution)

---

## 2. Stronger GFM Assertions

**Priority: high — regression safety for GFM extensions**

The current GFM test only checks "has document block". Add structural
assertions for GFM-specific extensions to catch regressions.

### Tasks

- [x] Add table assertions: header row, data rows, alignment, cell text
- [x] Add task list assertions: checked/unchecked markers on list items
- [x] Add strikethrough assertions: Strike style + cross-paragraph rejection
- [x] Add autolink extension assertions: www., http://, email detection
- [x] Add tag filter assertions (block structure + raw HTML text)
- [x] Cross-reference GFM examples with CommonMark examples — added
      empty-output handling for ref-def-only examples (176, 188)

### Definition of done

- [x] All 24 GFM extension examples have structural assertions beyond
  "has document block"
- [x] `TestGFMSupportedExamples` runs all 672 assertion functions
  against actual parser output

### Table parsing fixes (v0.36.1)

- Fixed example 199: pipe-less delimiter rows now recognized
- Fixed example 202: pipe-less continuation rows now accepted
- Fixed example 203: column-count mismatch now correctly rejected

---

## 3. Demo Application & Terminal Recording

**Priority: high — README eye-catcher**

Build a polished demo application that showcases the streaming parser
and terminal renderer with a curated set of Markdown examples. Record
the terminal output as an animated GIF/video for the README.

### Tasks

- [x] Create `examples/demo/` with curated `demo.md` covering: headings,
      emphasis, code blocks (Go, Rust, bash), lists (nested, task lists),
      blockquotes, tables, links, autolinks, thematic breaks, strikethrough
- [x] Build `examples/demo/main.go` with:
  - Embedded `demo.md` via `go:embed`
  - `--delay` flag (default 20ms)
  - `--chunk` flag (default 16 bytes)
  - `--record` flag (optimized: chunk=10, delay=25ms)
  - `--instant` flag (no streaming delay)
  - `--width` flag (wrap width override)
  - `--clear` flag (clear screen, default true)
  - File argument support for custom Markdown
- [x] Add `demo.tape` for vhs recording
- [x] Record a ~15-second terminal session showing streaming rendering
- [x] Convert to GIF for README embedding
- [x] Update README.md with demo section and current compliance stats

### Definition of done

- `go run ./examples/demo` produces a visually appealing terminal output
- Animated GIF embedded in README showing streaming Markdown rendering
- README reflects current CommonMark 96.2% and GFM 100% stats

---

## 4. Remaining CommonMark Gaps (25 examples)

**Priority: medium — diminishing returns**

Each fix is 1-3 examples and requires significant new features.

### Tasks

- [ ] Nested blockquote stack (#250, #251, #259, #260, #292, #293) — 6
- [ ] Multiline ref def title (#196) and label (#208) — 2
- [ ] Tab expansion in blockquotes/lists (#6, #7, #9) — 3
- [ ] Images inside links (#517, #520, #531) — 3
- [ ] Autolinks inside link labels (#526, #538) — 2
- [ ] Forward ref in heading (#214) — 1
- [ ] Ref def in blockquote (#218) — 1
- [ ] Fenced code `>` prefix in blockquote (#128) — 1
- [ ] Empty inline link destination (#567) — 1
- [ ] Nested ref link rejection (#533) — 1
- [ ] Misc edge cases (#175, #621, #626, #541) — 4

### Definition of done

- CommonMark compliance ≥ 98% (640+ / 652)

---

## 5. Performance Benchmarks

**Priority: medium — validates production claims**

### Tasks

- [ ] Benchmark large real-world documents (Linux kernel README,
      CommonMark spec itself, large changelogs)
- [ ] Benchmark pathological inputs (deeply nested, long delimiter runs)
- [ ] Benchmark different chunk sizes (1, 16, 64, 256, 1024, 4096 bytes)
- [ ] Memory profiling: verify bounded memory for streaming path
- [ ] Compare with goldmark for throughput baseline
- [ ] Add `go test -bench` targets to CI

### Definition of done

- Benchmark results documented
- No pathological input causes >10x slowdown vs normal input
- Memory stays bounded by unresolved state, not document size

---

## 6. Documentation

**Priority: medium — developer experience**

### Tasks

- [ ] Add godoc comments for all public types and functions in
      `stream` and `terminal` packages
- [ ] Update README.md with:
  - Current compliance numbers
  - Architecture overview
  - Usage examples (streaming, one-shot, custom renderer)
  - Demo GIF
  - API reference links
- [ ] Add `examples/stdin/` that reads from stdin and renders to terminal
- [ ] Add `examples/http/` that streams Markdown from an HTTP response

### Definition of done

- `go doc` produces clean output for all public API
- README is comprehensive and visually appealing

---

## 7. `cmd/mdview` — Terminal Markdown Viewer

**Priority: high — user-facing tool**

Build a standalone CLI tool for viewing Markdown files in the terminal,
using Bubble Tea v2 (charmbracelet/bubbletea) for a scrollable viewport
with keyboard navigation.

### Tasks

- [ ] Create `cmd/mdview/main.go` with Bubble Tea v2 viewport
- [ ] Accept file argument or stdin pipe
- [ ] Render Markdown through `stream.Parser` + `terminal.Renderer`
  into a string buffer, then display in viewport
- [ ] Keyboard: `j`/`k`/arrows for scroll, `q`/`Esc` to quit,
  `g`/`G` for top/bottom, `/` for search
- [ ] Mouse scroll support
- [ ] `--width` flag to override wrap width
- [ ] `--theme` flag (monokai default, plain for no color)
- [ ] `--no-pager` flag to dump output without viewport (like current demo)
- [ ] Add `task view` and `task view -- file.md` to Taskfile
- [ ] Add to README as primary usage example

### Definition of done

- `go run ./cmd/mdview README.md` opens a scrollable, styled Markdown view
- Keyboard and mouse navigation work
- Piping works: `cat README.md | go run ./cmd/mdview`
- Installable: `go install github.com/codewandler/markdown/cmd/mdview@latest`

---

## 8. Comparison with Other Renderers

**Priority: medium — credibility and positioning**

Publish a comparison of this library against other Go Markdown
renderers (goldmark, blackfriday, glamour) covering performance,
memory usage, and spec completeness.

### Tasks

- [ ] Benchmark throughput (MB/s) against goldmark and blackfriday
  on real-world documents (CommonMark spec, large changelogs, READMEs)
- [ ] Benchmark memory allocation per document (bytes/op, allocs/op)
- [ ] Benchmark streaming memory: peak RSS for 1MB+ documents
  (this library should stay flat; batch parsers grow linearly)
- [ ] Benchmark pathological inputs (deeply nested, long delimiter runs)
- [ ] Compare CommonMark spec pass rates across libraries
- [ ] Compare GFM extension coverage
- [ ] Compare feature matrix: streaming, terminal output, syntax
  highlighting, hyperlinks, word wrapping
- [ ] Write `COMPARISON.md` with tables, methodology, and reproduction
  commands
- [ ] Add comparison summary to README
- [ ] Add `task bench` and `task bench:compare` to Taskfile

### Definition of done

- `COMPARISON.md` with reproducible benchmarks and clear methodology
- README includes a summary table or link to comparison
- All benchmark code is in-repo and runnable via `task bench`

---

## Release Plan

| Version | Content |
|---------|---------|
| v0.35.1 | Fuzz testing + findings fixed |
| v0.36.0 | Stronger GFM assertions |
| v0.36.1 | GFM table parsing fixes |
| v0.37.0 | Demo application + README GIF |
| v0.38.0 | CommonMark gaps (target ≥98%) |
| v0.39.0 | Performance benchmarks + documentation |
| v0.40.0 | `cmd/mdview` terminal viewer |
| v0.41.0 | Comparison with other renderers |
| v1.0.0  | Stable API, full documentation, polished README |

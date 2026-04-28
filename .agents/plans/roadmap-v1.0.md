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

## 5. Competition (Benchmarks + Comparison + Competitor Research)

**Priority: medium — validates production claims + credibility**

Comprehensive competitive analysis: benchmarks, feature matrix,
competitor profiles, and syntax highlighting comparison.

### Deliverables

- `benchmarks/` — separate module with comparative benchmarks
- `COMPARISON.md` — full results with speedup ratios
- `docs/competitors.md` — detailed profiles of all Go Markdown libraries
- `benchmarks/cmd/benchcompare/` — tool to generate Markdown tables

### Competitors benchmarked

| Library | Parse | Terminal Render | Stream |
|---------|:-----:|:---------------:|:------:|
| **glamour** | via goldmark | yes | no |
| **go-term-markdown** | via blackfriday | yes | no |
| **goldmark** | yes | no | no |
| **blackfriday** | yes | no | no |
| **gomarkdown** | yes | no | no |

No other Go library supports streaming. We are unique.

### Tasks

- [x] Create `benchmarks/` module with all 5 competitors
- [x] Implement 9 input categories (spec, readme, github-top10,
  code-heavy, table-heavy, inline-heavy, pathological-nest,
  pathological-delim, large-flat)
- [x] Fetch 11 real-world READMEs from top GitHub projects
- [x] Benchmark: terminal render pipeline (us vs glamour vs go-term-md)
- [x] Benchmark: parse-only (us vs goldmark vs blackfriday vs gomarkdown)
- [x] Benchmark: chunk size sensitivity (us only)
- [x] Benchmark: Go syntax highlighting fast path vs Chroma
- [x] Build `benchcompare` tool for Markdown table generation
- [x] Write `COMPARISON.md` with speed, allocations, memory tables
- [x] Write `docs/competitors.md` with detailed library profiles
- [x] Feature matrix: streaming, highlighting, hyperlinks, wrapping,
  TTY detection, CommonMark compliance, GFM, dependencies
- [x] Add Taskfile tasks: bench, bench:render, bench:parse, bench:chunks
- [x] Add Performance section + summary to README

### Key results

- **Terminal rendering**: 1.2–56x faster than glamour, fewest allocations
- **Parse-only**: 2–4x slower than goldmark (expected: streaming trade-off)
- **Go highlighting**: 18x faster than Chroma, 6.7x fewer allocations
- **Streaming**: 4KB chunks faster than whole-doc; 1-byte only 1.1x slower

### Definition of done

- [x] `task bench:render` and `task bench:parse` produce comparison tables
- [x] `COMPARISON.md` with reproducible results and clear methodology
- [x] `docs/competitors.md` with all Go Markdown library profiles
- [x] README includes performance summary

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

## 8. Built-in Syntax Highlighters (Drop Chroma)

**Priority: medium — reduce dependencies, improve performance**

Extend the Go stdlib AST fast path approach to more languages. Each
built-in highlighter replaces Chroma for that language, making it
18x+ faster with zero external dependencies. Chroma remains as the
fallback for languages without a built-in highlighter.

### Approach

The `HybridHighlighter` already dispatches Go to the fast path and
everything else to Chroma. Adding a new language means:

1. Write `highlightXxxLine(line string) string` in `terminal/highlight.go`
2. Add the language to the `HybridHighlighter.HighlightLine` switch
3. Add a benchmark in `benchmarks/bench_test.go` comparing fast path vs Chroma
4. Verify output looks correct in the demo

The highlighters don't need to be perfect — they need to handle
keywords, strings, numbers, comments, and punctuation. Chroma handles
edge cases for users who need pixel-perfect highlighting.

### Languages (in priority order)

| Language | Why | Complexity |
|----------|-----|------------|
| JSON | Config files, API responses, ubiquitous | Low — strings, numbers, booleans, null, punctuation |
| YAML | Config files, Kubernetes, CI/CD | Low — keys, strings, comments, indentation |
| TOML | Go config files (cargo, pyproject) | Low — keys, strings, numbers, comments, sections |
| Bash/Shell | READMEs, CI scripts, examples | Medium — keywords, strings, comments, variables |
| Python | Popular in AI/ML docs | Medium — keywords, strings, comments, decorators |
| TypeScript | Web ecosystem | Medium — keywords, strings, types, comments |
| Rust | Systems programming | Medium — keywords, lifetimes, macros, strings |
| SQL | Database docs | Low — keywords, strings, comments |

### Tasks

- [ ] JSON highlighter (strings, numbers, booleans, null, braces)
- [ ] YAML highlighter (keys, values, comments, anchors)
- [ ] TOML highlighter (keys, values, comments, section headers)
- [ ] Bash highlighter (keywords, strings, comments, variables, pipes)
- [ ] Python highlighter (keywords, strings, comments, decorators)
- [ ] Benchmark each new highlighter vs Chroma
- [ ] Consider making Chroma an optional build tag so pure-stdlib
  builds are possible

### Definition of done

- JSON, YAML, TOML have built-in highlighters
- Each is benchmarked at 10x+ faster than Chroma
- Chroma remains as fallback for unlisted languages
- `go run ./examples/demo` renders all languages correctly

---

## 9. Theming

**Priority: medium — customization, user experience**

Replace hardcoded Monokai color constants with a `Theme` struct that
can be swapped at renderer construction time. Ship built-in themes
and allow users to define custom ones.

### Current state

Colors are hardcoded as `monokaiForeground`, `monokaiComment`, etc.
in `terminal/renderer.go`. The `styler` interface wraps them but
doesn't parameterize them — `ansiStyler` always uses Monokai.

### Design

```go
type Theme struct {
    Foreground  string // ANSI escape for default text
    Comment     string // dim/muted text (borders, metadata)
    Keyword     string // headings, bold markers
    String      string // inline code, string literals
    Function    string // links, emphasis
    Type        string // blockquote text, secondary
    Number      string // list markers, ordered list numbers
    Error       string // strikethrough, warnings
    Background  string // code block background (optional)
}
```

- `WithTheme(theme Theme)` renderer option
- `ansiStyler` reads colors from the theme instead of constants
- Built-in highlighters use theme colors instead of hardcoded Monokai
- Chroma style name derived from theme or configurable separately

### Built-in themes

| Theme | Description |
|-------|-------------|
| Monokai | Current default — warm, high contrast |
| Dracula | Popular dark theme |
| Nord | Cool, muted Scandinavian palette |
| Solarized Dark | Ethan Schoonover's classic |
| Solarized Light | Light variant |
| One Dark | Atom-inspired |
| Plain | No colors (current `plainStyler` behavior) |

### Tasks

- [ ] Define `Theme` struct with semantic color roles
- [ ] Create `WithTheme(theme Theme)` renderer option
- [ ] Refactor `ansiStyler` to read from `Theme` instead of constants
- [ ] Refactor built-in Go highlighter to use theme colors
- [ ] Ship Monokai, Dracula, Nord, Solarized Dark/Light, One Dark themes
- [ ] Add `--theme` flag to `examples/demo` and `cmd/mdview`
- [ ] Add `THEME_NAME` environment variable support
- [ ] Document theme API and custom theme creation in README
- [ ] Ensure Chroma style aligns with selected theme

### Definition of done

- `WithTheme(ThemeDracula)` produces Dracula-colored output
- Custom themes work: `WithTheme(Theme{Foreground: "...", ...})`
- Demo and mdview support `--theme` flag
- At least 5 built-in themes ship

---

## Release Plan

| Version | Content |
|---------|---------|
| v0.35.1 | Fuzz testing + findings fixed |
| v0.36.0 | Stronger GFM assertions |
| v0.36.1 | GFM table parsing fixes |
| v0.37.0 | Demo application + README GIF |
| v0.38.0 | Benchmarks + competition + drop goldmark |
| v0.39.0 | CommonMark gaps (target ≥98%) + documentation |
| v0.40.0 | `cmd/mdview` terminal viewer |
| v0.41.0 | Built-in highlighters (JSON, YAML, TOML) |
| v0.42.0 | Theming (Monokai, Dracula, Nord, Solarized, One Dark) |
| v1.0.0  | Stable API, full documentation, polished README |

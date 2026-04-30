<div align="center">

# markdown

**The fastest Go terminal Markdown renderer. The only one that streams.**

Parse incrementally. Render immediately. Keep memory bounded.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CommonMark](https://img.shields.io/badge/CommonMark-100%25-brightgreen)](https://spec.commonmark.org/)
[![GFM](https://img.shields.io/badge/GFM-97.1%25-brightgreen)](https://github.github.com/gfm/)

<img src="examples/demo/demo.gif" alt="demo" width="720">

</div>

## Why this exists

- **Only streaming Markdown renderer in Go** -- no other library
  supports chunk-by-chunk parsing and rendering
- **3-8x faster** than glamour on real-world documents, **fewer
  allocations than goldmark** on every input
  ([benchmarks](COMPARISON.md))
- **100% CommonMark** (652/652), **97.1% GFM** (707/728) -- higher compliance than
  goldmark (94.8%), blackfriday (34.6%), and gomarkdown (36.8%)
  ([measured](competition/benchmarks/gfmext_test.go),
  [details](docs/compliance.md))
- **2 core dependencies** -- parser is pure stdlib; the root module only uses
  Chroma for terminal syntax highlighting
- **18x faster Go highlighting** than Chroma via built-in stdlib
  AST fast path

## Performance

Benchmarked against 5 Go Markdown libraries. Full results in
[COMPARISON.md](COMPARISON.md). Competitor profiles in
[docs/competitors.md](docs/competitors.md).

### Terminal rendering (parse + render)

| Input | ours | glamour | go-term-md |
| --- | ---: | ---: | ---: |
| Spec (~120KB) | **9.2ms** | 40.9ms (4.4x slower) | 405ms (44x slower) |
| README (~10KB) | **1.2ms** | 6.2ms (5.3x slower) | 3.9ms (3.3x slower) |
| GitHub Top 10 (~130KB) | **32.9ms** | 37.2ms (1.1x slower) | 2,770ms (84x slower) |

### Spec compliance (measured, not claimed)

| Spec | ours | goldmark | blackfriday | gomarkdown |
| --- | ---: | ---: | ---: | ---: |
| CommonMark 0.31.2 | **652/652 (100%)** | 647/652 (99.2%) | 244/652 (37.4%) | 263/652 (40.3%) |
| GFM 0.29 spec.txt | **663/672 (98.7%)** | 655/672 (97.5%) | 247/672 (36.8%) | 263/672 (39.1%) |
| GFM extensions.txt | **16/30** | 21/30 | 1/30 | 1/30 |
| GFM regression.txt | **15/26** | 14/26 | 4/26 | 4/26 |
| **GFM total** | **707/728 (97.1%)** | 690/728 (94.8%) | 252/728 (34.6%) | 268/728 (36.8%) |

See [docs/compliance.md](docs/compliance.md) for details on remaining gaps.

### Feature matrix

| | ours | glamour | go-term-md | goldmark | blackfriday |
| --- | :---: | :---: | :---: | :---: | :---: |
| **Streaming** | **yes** | no | no | no | no |
| Terminal render | yes | yes | yes | no | no |
| Go fast path | **18x faster** | no | no | n/a | n/a |
| OSC 8 hyperlinks | yes | no | no | n/a | n/a |
| Inline extension events | yes | no | no | parser only | no |
| TTY auto-detect | yes | no | no | n/a | n/a |
| Core dependencies | **2** | ~20 | ~15 | 0 | 0 |

## Quick Start

```go
package main

import (
    "os"
    "github.com/codewandler/markdown/terminal"
)

func main() {
    r := terminal.NewStreamRenderer(os.Stdout)
    r.Write([]byte("# Hello\n\nThis is **streaming** Markdown.\n"))
    r.Flush()
}
```

For interactive terminals where tables should grow in-place as rows arrive,
use `terminal.NewLiveRenderer`. It implements the same `Write`/`Flush` shape as
`NewStreamRenderer`, but may emit cursor-control escape sequences while a table
is active.

```bash
go get github.com/codewandler/markdown
```

## Demo

```bash
go run ./examples/demo                         # stream the built-in showcase with live table redraws
go run ./examples/demo README.md               # stream any file
go run ./examples/demo --chunk 10 --delay 30ms # tune the effect
go run ./examples/demo --instant               # render all at once
go run ./examples/demo --live=false            # disable live table redraws
```

## mdview

`cmd/mdview` is a terminal viewer for local files or stdin. It supports normal
append-only rendering, fixed-width append-only table streaming, interactive
live table redraws, built-in themes, OSC 8 links for file references such as
`foo.go:18`, and a Bubble Tea pager.

```bash
go run ./cmd/mdview README.md
cat README.md | go run ./cmd/mdview --width 100

# Exercise chunked streaming behavior.
go run ./cmd/mdview --stream --chunk 20 --delay 200ms README.md

# Append-only table streaming: widths are fixed, overflow is clipped/ellipsized.
go run ./cmd/mdview --stream --table-mode fixed --table-widths 16,12,40 README.md

# Auto-width append-only table streaming: columns share a target table width.
go run ./cmd/mdview --stream --table-mode auto --table-max-width 100 README.md

# Interactive live table rendering: redraws the active table as widths grow.
go run ./cmd/mdview --live --stream --chunk 20 --delay 200ms README.md

# Use a built-in theme.
go run ./cmd/mdview --theme nord README.md

# Browse rendered Markdown in an interactive pager.
go run ./cmd/mdview --pager README.md
```

Use `--live` only for interactive terminal output. It redraws the active table
region using ANSI cursor controls; for pipes, logs, recordings, or Bubble Tea
viewports that should be append-only, use the default buffered mode or
`--table-mode fixed|auto`.

Use `--pager` for full-input browsing in an alternate-screen viewport. The first
pager implementation keeps source Markdown and rendered output in memory; it is
intended for normal documentation and generated reports, not multi-gigabyte files.

## Architecture

```text
chunks --> stream.Parser --> events --> terminal.Renderer/LiveRenderer --> output
```

| Package              | Role                                        |
| -------------------- | ------------------------------------------- |
| `stream`             | Incremental parser, append-only event model |
| `terminal`           | Terminal renderer over `stream.Event`       |
| `html`               | HTML renderer over `stream.Event`           |
| `bubbleview`         | Nested Bubble Tea module for Markdown viewports |
| `cmd/mdview`         | Nested terminal viewer CLI module           |
| `competition`        | Comparative benchmarks against 5 libraries  |
| `examples/demo`      | Streaming showcase with recording support   |

The parser emits structure. The renderer owns presentation. Neither
knows about the other's internals. Custom inline scanners can add semantic
inline atoms to the stream without preprocessing the source document.

## Dependencies

The core parser (`stream`) has **zero dependencies** -- pure Go stdlib.

The terminal renderer has one root-module dependency:

| Dependency | Why |
| --- | --- |
| [`chroma`](https://github.com/alecthomas/chroma) | Syntax highlighting for non-Go code (24-bit truecolor). Go uses a built-in stdlib AST fast path that is 18x faster. |

No framework, no goldmark, no blackfriday in the root library. Bubble Tea/Bubbles
are isolated in the nested `bubbleview` and `cmd/mdview` modules so applications
that only need parsing/rendering do not inherit TUI dependencies.

## Inline Extensions

Use `stream.WithInlineScanner` for small inline syntaxes that are not Markdown
itself, such as emoji shortcodes, issue references, or mentions. Scanners run
during inline tokenization after higher-precedence Markdown constructs such as
backslash escapes and code spans, so `:ok:` inside backticks remains literal
code while `**:ok:**` can produce a styled custom inline event.

```go
type emojiScanner struct{}

func (emojiScanner) TriggerBytes() string { return ":" }

func (emojiScanner) ScanInline(input string, _ stream.InlineContext) (stream.InlineScanResult, bool) {
    if !strings.HasPrefix(input, ":ok:") {
        return stream.InlineScanResult{}, false
    }
    return stream.InlineScanResult{
        Consume: len(":ok:"),
        Event: stream.Event{
            Kind: stream.EventInline,
            Inline: &stream.InlineData{
                Type:         "emoji",
                Source:       ":ok:",
                Text:         "✅",
                DisplayWidth: 2,
            },
        },
    }, true
}
```

Register the scanner directly on a parser:

```go
p := stream.NewParser(stream.WithInlineScanner(emojiScanner{}))
```

or through the terminal stream renderer:

```go
r := terminal.NewStreamRenderer(
    os.Stdout,
    terminal.WithParserOptions(stream.WithInlineScanner(emojiScanner{})),
)
```

Terminal renderers handle unknown inline atoms by rendering `Inline.Text` and
using `Inline.DisplayWidth` for tables and layout. For custom presentation,
register a renderer by inline type:

```go
renderer := terminal.NewRenderer(os.Stdout,
    terminal.WithInlineRenderer("emoji", func(ev stream.Event) (terminal.InlineRenderResult, bool) {
        return terminal.InlineRenderResult{Text: ev.Inline.Text, Width: ev.Inline.DisplayWidth}, true
    }),
)
```

Source preprocessors are intentionally not the main extension mechanism: they
can break source spans, CommonMark precedence, and chunk-safety. Prefer scanners
for inline syntax and renderer hooks for presentation.

## Bubble Tea Views

The nested `bubbleview` module provides reusable Bubble Tea components without
adding Bubble Tea dependencies to the root module:

```bash
go get github.com/codewandler/markdown/bubbleview
```

- `bubbleview.PagerModel` renders complete Markdown into a scrollable viewport.
- `bubbleview.StreamModel` accepts `MarkdownChunkMsg` and `MarkdownFlushMsg` for
  apps that receive Markdown incrementally, such as agent consoles or LLM UIs.
- Both models reuse `terminal.StreamRenderer`, parser options, themes, and custom
  inline renderers. They currently keep source and rendered output in memory so
  resize reflow can replay the source correctly.

## Terminal Renderer

- **Syntax highlighting** -- Go via stdlib AST (18x faster than Chroma),
  other languages via Chroma with 24-bit truecolor
- **OSC 8 hyperlinks** -- inline and reference links are clickable
- **Word wrapping** -- auto-detected terminal width or `WithWrapWidth`
- **TTY detection** -- ANSI escapes stripped when piped or redirected
- **Inline atoms** -- custom `EventInline` renderers with display-width-aware
  table layout
- **Table modes** -- buffered final-width tables, fixed/auto append-only
  streaming tables, or live redraws via `LiveRenderer`
- **Configurable** -- code block borders, padding, indentation, ANSI mode

## Testing

| Category | Coverage |
| --- | --- |
| Corpus classification | every CommonMark + GFM example tracked |
| Split equivalence | every example parsed at every chunk boundary |
| Structural assertions | 652 CommonMark + 24 GFM extension examples |
| Event invariants | balanced enter/exit, correct nesting |
| Fuzz testing | 3 `testing.F` targets, 1300+ seeds |
| Memory retention | completed blocks released promptly |

```bash
go test ./stream ./terminal .
task competition:compliance  # run spec compliance against all parsers
task competition:bench      # run benchmarks against all competitors
task competition:full       # full pipeline: metadata + compliance + bench + report
```

## Design Rules

1. Parser is **append-only** -- no backtracking or re-parsing
2. Events emit at **block boundaries** -- not deferred until flush
3. Memory bounded by **unresolved state** -- not document size
4. Renderer **never parses** Markdown syntax
5. Custom inline syntax uses **scanner events**, not source preprocessing
6. Terminal rendering is the **first-class output path**

## Roadmap

See [`roadmap-v1.0.md`](.agents/plans/roadmap-v1.0.md) for the full plan.

| Milestone | Status |
| --- | --- |
| Fuzz testing | :white_check_mark: v0.35.1 |
| GFM structural assertions | :white_check_mark: v0.36.0 |
| GFM table parsing fixes | :white_check_mark: v0.36.1 |
| Demo application + README | :white_check_mark: v0.37.0 |
| Benchmarks + competition | :white_check_mark: v0.38.0 |
| HTML renderer + 100% CommonMark | :white_check_mark: v0.39.0 |
| `cmd/mdview` terminal viewer | :white_check_mark: v0.42.0 |
| Terminal themes and file-reference links | :white_check_mark: v0.45.0 |
| Bubble Tea `bubbleview` module and `mdview --pager` | :white_check_mark: v0.46.0 |
| v1.0 stable API | planned |

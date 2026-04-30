<div align="center">

# markdown

**The fastest Go terminal Markdown renderer. The only one that streams.**

Parse incrementally. Render immediately. Keep memory bounded.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CommonMark](https://img.shields.io/badge/CommonMark-100%25-brightgreen)](https://spec.commonmark.org/)
[![GFM](https://img.shields.io/badge/GFM-98.7%25-brightgreen)](https://github.github.com/gfm/)

<img src="examples/demo/demo.gif" alt="demo" width="720">

</div>

## Why this exists

- **Only streaming Markdown renderer in Go** -- no other library
  supports chunk-by-chunk parsing and rendering
- **3-7x faster** than glamour on real-world documents, up to
  **121x faster** on pathological inputs
  ([benchmarks](COMPARISON.md))
- **100% CommonMark** (652/652), **98.7% GFM** (663/672) -- higher compliance than
  goldmark (99.2% / 97.5%), blackfriday (37.4%), and gomarkdown (40.3%)
  ([measured](competition/benchmarks/compliance_test.go))
- **2 dependencies** -- parser is pure stdlib; only Chroma for
  non-Go syntax highlighting
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
| CommonMark 0.31.2 | **652/652 (100.0%)** | 647/652 (99.2%) | 244/652 (37.4%) | 263/652 (40.3%) |
| GFM 0.29 | **663/672 (98.7%)** | 655/672 (97.5%) | 247/672 (36.8%) | 263/672 (39.1%) |

### Feature matrix

| | ours | glamour | go-term-md | goldmark | blackfriday |
| --- | :---: | :---: | :---: | :---: | :---: |
| **Streaming** | **yes** | no | no | no | no |
| Terminal render | yes | yes | yes | no | no |
| Go fast path | **18x faster** | no | no | n/a | n/a |
| OSC 8 hyperlinks | yes | no | no | n/a | n/a |
| TTY auto-detect | yes | no | no | n/a | n/a |
| Dependencies | **2** | ~20 | ~15 | 0 | 0 |

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

```bash
go get github.com/codewandler/markdown
```

## Demo

```bash
go run ./examples/demo                         # stream the built-in showcase
go run ./examples/demo README.md               # stream any file
go run ./examples/demo --chunk 10 --delay 30ms # tune the effect
go run ./examples/demo --instant               # render all at once
```

## Architecture

```text
chunks --> stream.Parser --> events --> terminal.Renderer --> output
```

| Package              | Role                                        |
| -------------------- | ------------------------------------------- |
| `stream`             | Incremental parser, append-only event model |
| `terminal`           | Terminal renderer over `stream.Event`       |
| `examples/demo`      | Streaming showcase with recording support   |
| `benchmarks`         | Comparative benchmarks against 5 libraries  |

The parser emits structure. The renderer owns presentation. Neither
knows about the other's internals.

## Dependencies

The core parser (`stream`) has **zero dependencies** -- pure Go stdlib.

The terminal renderer has one dependency:

| Dependency | Why |
| --- | --- |
| [`chroma`](https://github.com/alecthomas/chroma) | Syntax highlighting for non-Go code (24-bit truecolor). Go uses a built-in stdlib AST fast path that is 18x faster. |

No framework, no goldmark, no blackfriday. Written from scratch for
streaming.

## Terminal Renderer

- **Syntax highlighting** -- Go via stdlib AST (18x faster than Chroma),
  other languages via Chroma with 24-bit truecolor
- **OSC 8 hyperlinks** -- inline and reference links are clickable
- **Word wrapping** -- auto-detected terminal width or `WithWrapWidth`
- **TTY detection** -- ANSI escapes stripped when piped or redirected
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
5. Terminal rendering is the **first-class output path**

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
| `cmd/mdview` terminal viewer | planned |
| v1.0 stable API | planned |

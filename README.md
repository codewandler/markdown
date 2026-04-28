<div align="center">

# markdown

**Streaming Markdown parser and terminal renderer for Go**

Parse incrementally. Render immediately. Keep memory bounded.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CommonMark](https://img.shields.io/badge/CommonMark-96.2%25-brightgreen)](https://spec.commonmark.org/)
[![GFM](https://img.shields.io/badge/GFM-100%25-brightgreen)](https://github.github.com/gfm/)

<img src="examples/demo/demo.gif" alt="demo" width="720">

</div>

## Why this exists

- **Streaming-first** -- parse and render chunks as they arrive, no
  buffering the whole document
- **Append-only events** -- the parser never backtracks or re-parses
- **Bounded memory** -- only unresolved state is kept, not the full document
- **Terminal-native** -- Monokai palette, syntax highlighting, clickable
  links, word wrapping
- **Spec-compliant** -- 96.2% CommonMark, 100% GFM, fuzz-tested

## Features

Headings, paragraphs, blockquotes, ordered and unordered lists, task
lists, tables with alignment, fenced and indented code, emphasis,
strong, ~~strikethrough~~, `code spans`, inline/reference/auto links,
images, thematic breaks, setext headings, HTML blocks.

## Dependencies

The core parser (`stream`) has **zero dependencies** -- it is pure Go
standard library.

The terminal renderer (`terminal`) has one dependency:

| Dependency | Why |
| --- | --- |
| [`alecthomas/chroma`](https://github.com/alecthomas/chroma) | Syntax highlighting for non-Go fenced code blocks (24-bit truecolor). Go code uses a built-in stdlib AST fast path and does not need Chroma. |

No framework, no goldmark, no blackfriday. The parser and renderer are
written from scratch for the streaming use case.

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
go run ./examples/demo                    # stream the built-in showcase
go run ./examples/demo README.md          # stream any file
go run ./examples/demo --chunk 10 --delay 30ms  # tune the effect
go run ./examples/demo --instant          # render all at once
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

The parser emits structure. The renderer owns presentation. Neither
knows about the other's internals.

## Terminal Renderer

- **Syntax highlighting** -- Go via stdlib AST (fast path), other
  languages via Chroma with 24-bit truecolor
- **OSC 8 hyperlinks** -- inline and reference links are clickable
- **Word wrapping** -- auto-detected terminal width or `WithWrapWidth`
- **TTY detection** -- ANSI escapes stripped when piped or redirected
- **Configurable** -- code block borders, padding, indentation, ANSI mode

## Performance

Fastest Go terminal Markdown renderer. See [COMPARISON.md](COMPARISON.md)
for full benchmarks against glamour, go-term-markdown, goldmark,
blackfriday, and gomarkdown.

| vs glamour | Speed | Allocations |
| --- | ---: | ---: |
| Spec (~120KB) | **6.9x faster** | **5.5x fewer** |
| README (~10KB) | **6.5x faster** | **4.8x fewer** |
| GitHub Top 10 | **1.2x faster** | **8.6x fewer** |

Only streaming Markdown renderer in Go — no other library supports
chunk-by-chunk parsing and rendering.

## Conformance

| Spec              | Pass Rate | Examples |
| ----------------- | --------- | -------- |
| CommonMark 0.31.2 | **96.2%** | 627/652  |
| GFM 0.29          | **100%**  | 672/672  |

Every example is tested for split equivalence across all chunk
boundaries, structural correctness, and balanced event invariants.
The fuzz suite covers 1300+ seeds with 3 `testing.F` targets.

```bash
go test ./stream ./terminal .
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
| CommonMark gaps (target >= 98%) | planned |
| `cmd/mdview` terminal viewer | planned |
| Benchmarks + renderer comparison | planned |
| v1.0 stable API | planned |

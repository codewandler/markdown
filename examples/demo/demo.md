# Streaming Markdown

A production-ready **streaming parser** and **terminal renderer** for Markdown,
built in Go. Parse incrementally, render immediately, keep memory bounded.

> The parser owns structure. The renderer owns presentation.
> Both stay small enough to keep the streaming path responsive.

---

## Features

| Feature              | Status      | Notes                          |
| -------------------- | ----------- | ------------------------------ |
| ATX headings         | supported   | levels 1-6                     |
| Paragraphs           | supported   | soft wraps, hard breaks        |
| Fenced code          | supported   | Go fast path + Chroma fallback |
| Tables               | supported   | alignment, pipe-less rows      |
| Live tables          | supported   | redraw as later rows widen the columns during streaming |
| Task lists           | supported   | checked and unchecked items    |
| Blockquotes          | supported   | nested, with lazy continuation |
| Inline styles        | supported   | *emphasis*, **strong**, `code` |
| ~~Strikethrough~~    | supported   | GFM extension                  |
| Links                | supported   | inline, reference, autolinks   |

## Compliance

The parser passes **96.2%** of the CommonMark 0.31.2 spec and **100%** of the
GFM 0.29 spec. Every example is tested for split equivalence across all
possible chunk boundaries.

## Getting Started

- [x] Install Go 1.22+
- [x] Clone the repository
- [ ] Run the demo: `go run ./examples/demo`
- [ ] Try your own files: `go run ./examples/demo README.md`

## Go Example

```go
package main

import (
    "os"
    "github.com/codewandler/markdown/terminal"
)

func main() {
    r := terminal.NewStreamRenderer(os.Stdout)

    // Stream chunks as they arrive -- from an API, a pipe, or a file.
    chunks := []string{
        "# Hello\n\n",
        "This is **streaming** ",
        "Markdown rendering.\n",
    }
    for _, chunk := range chunks {
        r.Write([]byte(chunk))
    }
    r.Flush()
}
```

## Rust Example

```rust
use std::io::{self, Read};

fn stream_chunks(input: &[u8], chunk_size: usize) -> Vec<&[u8]> {
    input.chunks(chunk_size).collect()
}

fn main() -> io::Result<()> {
    let markdown = b"# Hello from Rust\n\nStreaming works everywhere.\n";
    for chunk in stream_chunks(markdown, 16) {
        io::stdout().write_all(chunk)?;
    }
    Ok(())
}
```

## Shell Pipeline

```bash
# Stream a file with visible chunking
cat README.md | go run ./examples/demo --chunk 24 --delay 30ms

# Render instantly
go run ./examples/demo --instant README.md

# Record-optimized settings for GIF capture
go run ./examples/demo --record
```

## Architecture

The pipeline is simple:

> **Chunks** --> `stream.Parser` --> **Events** --> `terminal.Renderer` --> **Output**

Each stage is independent:

1. The parser is **append-only** -- it never backtracks or re-parses
2. Events are emitted at **block boundaries**, not deferred until flush
3. Memory is bounded by **unresolved state**, not document size
4. The renderer consumes events without parsing Markdown syntax

## Links

Visit the project at https://github.com/codewandler/markdown for the source,
or check the [CommonMark spec](https://spec.commonmark.org/) and the
[GFM spec](https://github.github.com/gfm/) for the standards we target.

---

*Built with care for terminals, streams, and fast feedback loops.*

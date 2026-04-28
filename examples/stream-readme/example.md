# markdown

Streaming Markdown parsing and rendering for Codewandler projects.

This repository provides a small, production-oriented Markdown pipeline that
keeps parsing incremental and rendering terminal output as events arrive. It is
designed for AI streams, long-lived CLI tools, and other inputs that should not
wait for a full document before showing useful output.

> The parser owns Markdown structure.
> The terminal renderer owns presentation.
> Both stay small enough to keep the streaming path responsive.

## Supported Surface

| Area                   | Status | Notes |
|------------------------| --- | --- |
| Headings               | supported | ATX headings in streamed input |
| Paragraphs             | supported | soft wraps stay as spaces |
| Lists                  | supported | ordered, unordered, and task items |
| Tables                 | supported | pipe tables with alignment metadata |
| **Code blocks**        | supported | fenced and indented blocks |
| Quotes                 | supported | nested blockquotes stay readable |
| Inline styles          | supported | emphasis, strong, strike, code |
| Links                  | supported | inline and reference-style links |
| Autolinks              | supported | bare URLs and email addresses |
| HTML rendering         | planned | stays out of scope for now |
| Incremental DOM output | planned | future renderer target |

## What This Looks Like

The main use case is a terminal session that receives small chunks, renders
them immediately, and keeps memory bounded to unresolved parser state.

The default renderer uses a Monokai-inspired palette, a thin code border, and a
hybrid code highlighter that stays fast for Go and falls back for other
languages.

## Current Workflow

- [x] Stream Markdown into the parser in small chunks
- [x] Emit append-only events instead of rebuilding the document
- [x] Render headings, quotes, lists, tables, and code blocks in the terminal
- [x] Highlight Go code on the fast path
- [x] Fall back to generic highlighting for Rust, JavaScript, Python, and shell
- [ ] Add incremental DOM rendering
- [ ] Add HTML output that preserves validity without full re-rendering

## Example Content

The public API is centered around `terminal.NewStreamRenderer`, which combines
parsing and rendering behind a single `io.Writer`-style helper. The
[repository on GitHub](https://github.com/codewandler/markdown) shows the
current code, and bare links such as https://github.com/codewandler/markdown
should render as clickable autolinks too.

See the project [README][readme] for the package overview and the
[roadmap][roadmap] for what comes next.

You can also stream directly from a shell pipeline or a file descriptor, for
example when rendering agent output live:

```bash
cat README.md | go run ./examples/stream-readme -chunk 48 -delay 20ms
```

## Release Notes

Before shipping a release, the usual checklist is:

- [x] confirm parser and renderer tests pass
- [x] verify the streaming example still works
- [x] keep the changelog aligned with behavior changes
- [ ] cut the next release tag
- [ ] publish notes on GitHub

## Notes On Compatibility

CommonMark conformance stays the baseline. GFM extensions are included where
they fit the streaming model, especially tables, task lists, strikethrough, and
autolink literals.

HTML parsing is still intentionally out of scope here. If it comes later, it
needs to be incrementally valid instead of being rebuilt from scratch on every
chunk.

## Go Example

```go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/codewandler/markdown/terminal"
)

func main() {
	pipe := terminal.NewStreamRenderer(os.Stdout)
	input := strings.NewReader(strings.Join([]string{
		"# Streamed release notes",
		"",
		"- [x] parser events stay append-only",
		"- [ ] DOM rendering remains a future step",
		"",
		"| Area | Status |",
		"| --- | --- |",
		"| Tables | supported |",
		"| HTML | planned |",
		"",
		"See https://github.com/codewandler/markdown for the current repo.",
		"",
		"```go",
		"func render(ctx context.Context, r io.Reader) error {",
		"    _ = ctx",
		"    _ = r",
		"    return nil",
		"}",
		"```",
	}, "\n"))

	buf := make([]byte, 32)
	for {
		n, err := input.Read(buf)
		if n > 0 {
			if _, writeErr := pipe.Write(buf[:n]); writeErr != nil {
				panic(writeErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := pipe.Flush(); err != nil {
		panic(err)
	}
	fmt.Println()
}
```

## Rust Example

```rust
use std::collections::VecDeque;
use std::time::{Duration, Instant};

#[derive(Debug, Clone)]
struct Chunk {
    text: String,
    created_at: Instant,
}

struct Window {
    pending: VecDeque<Chunk>,
    max_age: Duration,
}

impl Window {
    fn new(max_age: Duration) -> Self {
        Self {
            pending: VecDeque::new(),
            max_age,
        }
    }

    fn push(&mut self, text: impl Into<String>) {
        self.pending.push_back(Chunk {
            text: text.into(),
            created_at: Instant::now(),
        });
        self.compact();
    }

    fn compact(&mut self) {
        while self.pending.len() > 128 {
            self.pending.pop_front();
        }
    }

    fn drain_ready(&mut self) -> Vec<String> {
        let mut ready = Vec::new();
        while let Some(front) = self.pending.front() {
            if front.created_at.elapsed() < self.max_age {
                break;
            }
            if let Some(chunk) = self.pending.pop_front() {
                ready.push(chunk.text);
            }
        }
        ready
    }
}

fn main() {
    let mut window = Window::new(Duration::from_millis(40));
    for line in [
        "streaming markdown stays responsive",
        "tables and task lists render incrementally",
        "generic highlighting handles non-Go code",
    ] {
        window.push(line);
    }

    for line in window.drain_ready() {
        println!("{line}");
    }
}
```

## Indented Code

    markdown.NewParser()
    terminal.NewStreamRenderer(os.Stdout)
    terminal.WithCodeBlockStyle(terminal.DefaultCodeBlockStyle())

## References

- [Project README][readme]
- [Project Roadmap][roadmap]

[readme]: ../../README.md
[roadmap]: ../../ROADMAP.md

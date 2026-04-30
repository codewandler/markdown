# bubbleview

Reusable Bubble Tea Markdown viewport components for `github.com/codewandler/markdown`.

This module intentionally keeps Bubble Tea/Bubbles dependencies out of the root
Markdown module.

## Components

- `StreamModel`: append Markdown chunks with `MarkdownChunkMsg`, then finalize
  with `MarkdownFlushMsg`.
- `PagerModel`: render complete Markdown input into a scrollable viewport.

Initial limitation: both models keep source Markdown and rendered viewport
content in memory. They are intended for interactive documents and streaming UI
responses, not multi-gigabyte files yet.

# markdown

Streaming Markdown parsing and rendering primitives for Codewandler projects.

This module is a rewrite of the streaming Markdown model. It owns its own
parser and renderer architecture instead of cloning `agentsdk/markdown`.

## What It Does

The core pipeline is:

```text
incoming chunks
  -> stream.Parser
  -> append-only parser events
  -> renderer-specific lowering
  -> terminal output
```

The parser emits Markdown structure, not terminal layout. The terminal
renderer consumes those events and owns presentation details such as spacing,
indentation, borders, colors, and code highlighting.

## Supported Surface

The supported product path is terminal-first and CommonMark-aware, with GFM
extensions where they fit the incremental model.

Supported blocks and inlines include:

- ATX headings
- paragraphs and soft line breaks
- thematic breaks
- fenced code and indented code
- blockquotes
- ordered and unordered lists
- pipe tables with alignment metadata
- paragraph-boundary inline parsing
- links, references, images, code spans, emphasis, and strong emphasis
- GFM task lists, strikethrough, and autolink literals

Unsupported Markdown is handled intentionally and should remain stable under
split input and `Flush`.

## Terminal Renderer

The terminal renderer stays low-level and dependency-light. It uses Monokai as
the default palette for Markdown structure and code fences. Fenced code blocks
render with a configurable left prefix, border, and padding.

Inline and reference links render as OSC 8 terminal hyperlinks. When the
renderer can determine a terminal width, it wraps visible text itself so
hyperlinks remain clickable across physical line breaks.

The renderer does not parse Markdown syntax. It only consumes parser events.
That keeps table layout, list prefixes, blockquote prefixes, and ANSI styling
local to presentation code. The built-in default highlighter keeps Go on the
stdlib fast path and applies a small generic fallback for other fenced code
languages such as Rust, JavaScript, Python, and shell.

## Packages

- `stream`: incremental parser API and canonical event model.
- `terminal`: terminal renderer over `stream.Event`.
- `examples/stream-readme`: separate example module that uses local `replace`
  directives.

## Example

Run the streaming README example with chunked input:

```bash
cd examples/stream-readme
go run . -chunk 32 -delay 20ms
```

The example demonstrates streaming rendering, table output, task lists,
strikethrough, autolinks, and fenced code highlighting.

## Conformance And Testing

The parser passes **627 / 652** CommonMark 0.31.2 spec examples (96.2%) and
**672 / 672** GFM 0.29 spec examples (100%). The test suite includes:

- **Corpus classification** — supported / known-gap / unsupported accounting
  for every CommonMark and GFM example.
- **Split equivalence** — every corpus example is parsed at every possible
  chunk boundary and verified to produce identical events.
- **Event invariants** — balanced enter/exit blocks, correct nesting, no
  orphan events.
- **Fuzz testing** — three `testing.F` targets (`FuzzParser`,
  `FuzzParserChunkBoundary`, `FuzzParserMultiChunk`) seeded with 1300+
  corpus examples and 40+ pathological inputs.
- **Responsiveness** — events are emitted at block boundaries, not deferred
  until flush.
- **Memory retention** — completed paragraphs and code lines are released
  promptly.

For local verification:

```bash
go test ./stream
go test ./terminal
go test .
cd examples/stream-readme && go test ./...
```

## Design Rules

- Keep the parser append-only.
- Keep renderer layout out of parser events.
- Keep memory bounded by unresolved state, not whole-document reparsing.
- Keep HTML rendering out of the terminal product path.
- Prefer explicit tests and corpus fixtures over heuristic behavior.

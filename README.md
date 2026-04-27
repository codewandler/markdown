# markdown

Markdown parsing and rendering primitives for Codewandler projects.

This is a rewrite of the streaming Markdown model, not a clone of
`agentsdk/markdown`. The agentsdk package is useful problem context, but this
module should own a cleaner parser and renderer architecture from the beginning.

## Goal

Build an efficient, failsafe, true incremental Markdown parser for streaming
AI-agent output.

The parser emits structured events. Renderers consume those events. The first
renderer target is terminal output. Browser and DOM rendering are intentionally
out of scope for the initial product path.

## Design Direction

The core boundary is:

```text
streaming bytes
  -> CommonMark-aware incremental parser
  -> canonical syntax events
  -> renderer-specific lowering
  -> terminal output
```

The parser should emit Markdown syntax events, not terminal layout operations.
That keeps CommonMark conformance testable. A renderer may create additional
internal render operations such as blank lines, quote prefixes, indentation, or
ANSI style transitions, but those operations belong after the parser event
stream.

This split matters because terminal rendering and future DOM rendering need
different layout behavior while sharing the same parser semantics.

## CommonMark Strategy

CommonMark conformance should be reached in layers:

1. Implement block parsing as a streaming state machine over complete lines.
2. Keep unfinished tails buffered and finalize them only when the stream proves
   they are stable or when `Flush` is called.
3. Start with paragraph-boundary inline parsing so supported output is correct
   before attempting an incremental inline delimiter stack.
4. Use the pinned CommonMark `0.31.2` JSON corpus for parser stability and
   explicit supported/known-gap/unsupported accounting.
5. Add structural event assertions for examples the current parser explicitly
   supports.
6. Add split-fuzz tests that feed the same Markdown through every chunk
   boundary and compare the final event sequence.

The parser is allowed to be conservative while streaming. It is not allowed to
panic or emit events that must later be retracted.

## Quality Gates

The parser test suite includes:

- full CommonMark `0.31.2` corpus split-equivalence checks
- exact supported/known-gap/unsupported CommonMark accounting, currently
  `169` supported, `88` known gaps, and `395` unsupported examples
- event invariants for balanced block enter/exit ordering
- responsiveness tests for stable streaming emission points
- memory-retention checks for large partial lines, fenced code, paragraphs, and
  reset behavior
- `-benchmem` benchmarks for long streams, tiny chunks, corpus parsing, and
  malformed delimiter-heavy input

## Renderer Strategy

The terminal renderer should be faithful to parser semantics. CommonMark
conformance is measured at the parser/event level with the official CommonMark
example corpus plus terminal behavior tests for the supported subset.

Renderer-specific extras should be internal render operations, not parser
events. For example, the terminal renderer may lower a blockquote event into
prefix operations, indentation state, and ANSI style changes. Those operations
should not be visible to other renderers or fed back into the parser event
stream.

The terminal renderer should stay low-level and dependency-light. Do not use
Glamour. Syntax highlighting should be line-oriented where possible so fenced
code can render incrementally. The initial color palette is Monokai and applies
to Markdown structure as well as code fences.

Broad language highlighting belongs behind an optional adapter. The
`adapters/chroma` submodule provides a Chroma-backed `CodeHighlighter` without
adding Chroma to the core module. Its hybrid highlighter keeps Go code on the
core renderer's fast stdlib path and falls back to Chroma for non-Go fences
such as Rust and JavaScript.

HTML rendering is out of scope for the terminal-first product path. If added
later, it must be a valid incremental HTML renderer over parser events rather
than a whole-document rerender helper.

Fenced-code markers are parser metadata and are not rendered literally by the
terminal renderer. Code blocks render with a configurable left prefix; by
default that is four spaces, a thin border, and one space before the highlighted
code.

## Packages

- `stream`: incremental parser API and canonical event model.
- `terminal`: terminal renderer over `stream.Event`.
- `adapters/chroma`: optional Chroma-backed code highlighting adapter.
- `examples/stream-readme`: separate example module that uses the core module
  and Chroma adapter through local `replace` directives.

## Try the Example

```bash
cd examples/stream-readme
go run . -chunk 32 -delay 20ms
```

Code block layout can be adjusted:

```bash
go run . -code-indent 2 -code-border=false
```

The example demonstrates the event pipeline, terminal renderer, fast Go
highlighting, and Chroma fallback for the Rust fence. It is a usage example,
not a conformance substitute.

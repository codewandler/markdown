# PLAN: True Incremental Markdown Parser

## Status

Production-target implementation plan. The repository currently has:

- a production-oriented `stream` parser foundation for the initial supported
  subset
- a low-level terminal renderer
- configurable terminal code-block layout
- a dependency-free Go highlighter
- an optional Chroma highlighter adapter in a submodule
- one runnable example module using local `replace` directives
- a pinned CommonMark `0.31.2` corpus loader
- full-corpus parser split-equivalence tests with explicit
  supported/known-gap/unsupported accounting
- parser event invariant tests
- parser responsiveness tests
- parser memory-retention tests for large partial lines, fenced code,
  paragraphs, and reset behavior
- `-benchmem` benchmarks for long streams, corpus parsing, tiny chunks, and
  malformed/pathological inline delimiter input
- exact CommonMark classification totals: `104` supported, `125` known gaps,
  and `423` unsupported examples
- complete ATX heading section coverage in the supported CommonMark corpus
- expanded fenced-code, indented-code, and code-span coverage
- complete paragraph, blank-line, and soft-line-break coverage in the
  supported CommonMark corpus
- complete autolink section coverage in the supported CommonMark corpus

The next implementation turns must expand conformance, performance,
responsiveness, memory, and agentsdk compatibility without weakening the
append-only streaming contract. No shortcuts, undocumented workarounds, or
"good enough for demo" behavior are acceptable.

## Outcome

Build a production-ready, high-performance, memory-efficient, responsive,
failsafe Markdown pipeline for streaming AI-agent responses:

```text
incoming chunks
  -> stream.Parser
  -> append-only Markdown events
  -> renderers
      -> terminal product renderer
      -> future incremental HTML renderer, out of current scope
```

The parser owns Markdown semantics. Renderers own presentation. Terminal
layout operations such as indentation, blank lines, borders, and ANSI styling
must not leak back into parser events.

## Priority Order

1. **Append-only correctness**: once emitted, events are never retracted.
2. **Chunk-boundary independence**: final events are the same for all chunk
   splits of supported input.
3. **Memory efficiency**: memory is bounded by unresolved parser state, not
   total streamed output.
4. **Performance**: each byte is scanned a bounded number of times; no
   accumulated-document reparsing.
5. **Responsiveness**: stable constructs emit as soon as correctness permits.
6. **CommonMark conformance**: supported behavior is tested against the
   official CommonMark example corpus at the parser/event and terminal levels.
7. **agentsdk compatibility**: integration preserves current public APIs long
   enough for a safe migration.

## Production Quality Bar

The project is not complete until these criteria are met:

- Every supported feature has tests for normal, malformed, unfinished, and
  chunk-split input.
- Unsupported features are documented and tested as literal fallback or
  explicit non-support; no silent accidental behavior.
- CommonMark compatibility is proven for the supported subset using the
  official CommonMark example corpus, structural parser assertions, split
  tests, and terminal behavior tests.
- Performance claims are backed by `-benchmem` benchmarks.
- Memory behavior for long streams is measured and has regression tests or
  benchmarks.
- Parser code does not rely on accumulated-document reparsing.
- Renderer code does not parse Markdown syntax.
- Public APIs have comments that state streaming and ownership contracts.
- All modules pass tests: core, Chroma adapter, and example.
- Known correctness bugs are release blockers, not backlog items.

## Resolved Decisions

These decisions close the current open questions for the first implementation
pass.

### Source Positions

Include source positions in events from the beginning.

Rationale:

- Useful for tests and debugging.
- Helps future DOM/incremental rendering experiments without changing every
  event type later.
- Cheap to maintain while scanning lines.

Implementation:

```go
type Position struct {
    Offset int64 // byte offset from stream start
    Line   int   // 1-based
    Column int   // 1-based byte column
}

type Span struct {
    Start Position
    End   Position
}
```

Add `Span Span` to `stream.Event`. Use zero values only for synthetic events
where no meaningful source span exists.

### Long Paragraph Latency

First parser is conformant and conservative: paragraphs are emitted only when
they are finalized by a blank line, block interruption, or `Flush`.

Do not split long paragraphs by default. Premature paragraph emission can break
CommonMark correctness for inline parsing and setext/link-reference related
behavior.

Add benchmarks for long paragraphs. Add an optional safety setting later only
after production correctness exists:

```go
WithMaxParagraphBytes(n int)
```

If added, the option must be explicitly documented as an AI-streaming latency
trade-off, not a CommonMark-conformance mode.

### GFM Scope

No GFM in the first production pass.

First target is a clearly documented CommonMark-compatible subset:

- ATX headings
- paragraphs
- blank lines
- thematic breaks
- fenced code
- indented code
- blockquotes
- ordered/unordered lists
- paragraph-boundary inlines

GFM tables, task lists, strikethrough, and extended autolinks come after this
core product is correct and measured.

### HTML Handling

HTML rendering is out of scope for the terminal-first product path.

Terminal renderer should show raw HTML source text for raw HTML constructs.
If/when an HTML renderer is added, it must be a real incremental renderer over
parser events that emits valid HTML as the stream advances. It must not be a
whole-document rerender helper and must not become the primary conformance
mechanism for terminal rendering.

### `agentsdk/markdown.Buffer`

Keep `agentsdk/markdown.Buffer` source-compatible during migration.

Migration strategy:

1. Add this module to agentsdk with a local replace during development.
2. Introduce an adapter that consumes `stream.Event`.
3. Keep `markdown.Buffer` as a compatibility wrapper or independent utility
   for at least one release.
4. Migrate terminal rendering first.
5. Deprecate or wrap `Buffer` only after downstream callers have moved.

## Non-goals

- DOM/browser rendering.
- HTML rendering in the current product path.
- Glamour or any all-in-one Markdown terminal renderer.
- Full CommonMark in the first implementation turn if it would compromise
  correctness of the supported subset.
- GFM before CommonMark-compatible core behavior.
- Public API freeze before conformance and split-fuzz tests exist.
- Incremental inline parsing before paragraph-boundary inline parsing is
  proven correct.

Non-goals are not permission for sloppy behavior. Any unsupported Markdown must
have an intentional, documented fallback.

## Module Shape

```text
github.com/codewandler/markdown
  stream
    event model
    parser options
    incremental block parser
    paragraph-boundary inline parser
    split tests
    benchmarks
  terminal
    renderer over stream.Event
    configurable code block layout
    dependency-free Go highlighter
  adapters/chroma
    optional code highlighter adapter in separate Go module
  examples/stream-readme
    single runnable example module using local replaces
```

## Public API Target

Keep the API small and explicit.

```go
package stream

type Parser interface {
    Write(chunk []byte) ([]Event, error)
    Flush() ([]Event, error)
    Reset()
}

type ParserOption func(*ParserConfig)

func NewParser(opts ...ParserOption) Parser
```

Initial options:

```go
type ParserConfig struct {
    InlineMode InlineMode
}

type InlineMode int

const (
    InlineParagraphBoundary InlineMode = iota
)
```

Do not add latency/safety options until tests show exactly where they are
needed.

API discipline:

- avoid exported types unless they are required by consumers
- add doc comments to all exported identifiers
- keep parser options deterministic and testable
- never expose mutable parser-owned buffers
- changing event semantics requires updating conformance and split tests

## Event Contract

Events are append-only and renderer-neutral.

Current event vocabulary can remain, but it needs enough metadata to represent
CommonMark blocks and lists without renderer heuristics.

Target shape:

```go
type Event struct {
    Kind  EventKind
    Block BlockKind
    Text  string
    Style InlineStyle
    Level int
    Info  string
    Span  Span

    List *ListData
}

type ListData struct {
    Ordered bool
    Start   int
    Marker  string
    Tight   bool
}
```

Rules:

- `EventEnterBlock` and `EventExitBlock` set `Block`.
- `EventText` sets `Text` and optional `Style`.
- `EventSoftBreak` and `EventLineBreak` represent Markdown line semantics, not
  terminal wrapping.
- Fenced-code language/info stays in `Info`.
- Lists use `ListData`; renderers should not parse list markers from text.
- Events must not contain terminal-specific layout fields.
- Event text is immutable from the caller’s perspective. Do not expose slices
  backed by mutable parser buffers.

## Parser Architecture

Use a line-oriented block parser with explicit state.

Parser state:

```text
parser
  offset, line, column
  partialLine []byte
  openBlocks []blockState
  paragraph paragraphState
  fence fenceState
  pendingEvents []Event
```

`Write` flow:

1. Append chunk bytes to `partialLine`.
2. Extract complete lines without converting the entire buffer repeatedly.
3. For each complete line, update source position counters.
4. If inside fenced code, handle closing fence or emit code text immediately.
5. Otherwise classify line and update block state.
6. Emit only stable events.
7. Keep incomplete trailing line in `partialLine`.

`Flush` flow:

1. Treat any partial line as a final complete line.
2. Finalize paragraph if open.
3. Close open code/list/blockquote/document blocks conservatively.
4. Return remaining events.
5. Leave parser reusable only after `Reset`; repeated `Flush` should not
   duplicate events.

Block recognition order:

1. blank line
2. fenced code close if in fence
3. fenced code open
4. indented code
5. blockquote marker
6. ATX heading
7. thematic break
8. list marker
9. paragraph continuation or paragraph start

CommonMark details to respect in the first pass:

- opening code fence may be indented up to three spaces
- fence length is at least three backticks or tildes
- closing fence must use same marker and at least opening length
- ATX headings require a space/tab after `#` unless heading is empty
- thematic breaks require three or more `*`, `-`, or `_` with optional spaces
- list markers include `-`, `+`, `*`, `1.`, `1)`
- ordered list start number should be captured
- blockquote marker is optional space after `>`

Known deferrals:

- setext headings may be deferred unless trivial to add in the block parser
- link reference definitions can initially be literal paragraph content
- full CommonMark list tight/loose detection can start conservative

Deferrals must be visible in tests. If a deferred construct appears in input,
the parser must either emit literal text consistently or return documented
unsupported behavior. It must not panic, corrupt state, or emit events that
later become wrong.

## Inline Architecture

First production pass uses paragraph-boundary inline parsing.

This means:

- paragraphs and headings buffer their text until finalized
- once finalized, inline parser emits text/style events for that block
- fenced code and indented code bypass inline parsing

Supported first inline subset:

- emphasis `*em*`, `_em_`
- strong `**strong**`, `__strong__`
- code spans
- autolinks `<https://example.com>` and `<user@example.com>`
- inline links `[text](url)`
- soft breaks
- hard breaks using two trailing spaces or backslash

Inline parser constraints:

- no panics on malformed delimiters
- unresolved delimiters degrade to literal text
- no O(n^2) delimiter scanning for long paragraphs
- no incremental inline stack until paragraph-boundary inline parsing is stable
  and measured

## Memory Efficiency Plan

Status: initial retention test coverage implemented in `v0.5.0`.

Parser must avoid retaining emitted content.

Rules:

- Fenced code lines are emitted immediately and not retained.
- Closed paragraphs are cleared after events are emitted.
- `Reset` releases or truncates large buffers.
- Partial-line buffer is the only unbounded byte buffer outside current block
  content.
- If a builder grows above a threshold, replace it instead of retaining a huge
  backing array.

Suggested thresholds:

- release paragraph buffer if capacity exceeds 64 KiB after close
- release partial-line buffer if capacity exceeds 64 KiB after line extraction
- benchmark before finalizing these numbers

Benchmarks:

- `BenchmarkParserLongFence`
- `BenchmarkParserLongParagraph`
- `BenchmarkParserTinyChunks`
- `BenchmarkParserCommonMarkCorpus`
- `BenchmarkParserCommonMarkCorpusTinyChunks`
- `BenchmarkParserMalformedInlineDelimiters`
- `BenchmarkParserPathologicalInlineDelimiters`

Benchmark expectations:

- long fence allocation count should scale with emitted events, not retained
  parser state
- tiny chunks should not cause quadratic runtime
- long paragraph should allocate for unresolved paragraph text once, not reparse
  accumulated document on each write

Memory regressions are release blockers. If benchmarks show retained memory
growing with emitted fenced-code output, fix the parser before adding features.

## Responsiveness Plan

Status: initial responsiveness test coverage implemented in `v0.5.0`.

Responsiveness target by construct:

- document start: first non-empty write starts document
- ATX heading: emit after heading line newline
- thematic break: emit after line newline
- fenced code open: emit block start after opening fence newline
- fenced code content: emit each code line after newline
- fenced code close: emit block end after closing fence newline
- paragraph: emit on blank line, block interruption, or `Flush`
- blockquote/list: emit when parser has enough continuation context

Do not emit paragraph text speculatively in the conformant production path.

Measure responsiveness with tests:

- write opening fence and one code line without closing fence; assert code line
  event is emitted
- write heading without newline; assert no heading event until newline or flush
- write paragraph without blank line; assert no paragraph text until flush
- write an interrupting block after a paragraph; assert the paragraph finalizes
  at the interruption point

## Conformance Plan

Add conformance in layers. The official CommonMark repository says the spec
contains embedded examples that serve as conformance tests, and its
`test/spec_tests.py --dump-tests` command emits the raw tests as JSON. Use that
test data as the corpus.

### Structural Event Tests

Create table-driven samples in `stream/parser_test.go`.

Sample categories:

- headings
- paragraphs
- blank lines
- thematic breaks
- fenced code
- unfinished fenced code
- indented code
- blockquotes
- unordered lists
- ordered lists
- malformed constructs

Each sample asserts event sequence after full write and after split writes.

### Split-Fuzz Tests

For every short sample:

1. render event sequence from one write
2. render event sequence for every possible single split
3. render event sequence for selected multi-splits
4. compare normalized event sequence

Normalization should ignore source spans only if source spans make split tests
too brittle. Preferred: source spans should still match.

### CommonMark Corpus Ingestion

Add a corpus import path that does not make HTML rendering part of the product:

```text
internal/commonmarktests
  testdata/commonmark-0.31.2.json
  loader.go
```

Source:

- official repository: `commonmark/commonmark-spec`
- generated with: `python3 test/spec_tests.py --dump-tests`
- current latest spec version from `spec.commonmark.org`: `0.31.2`

Use cases:

- split-fuzz every example, including unsupported examples
- classify examples by section
- assert supported sections structurally
- assert unsupported sections degrade consistently according to documented
  fallback behavior
- track pass/skip/unsupported counts in test output or a generated report

The JSON includes expected HTML. For now, do not compare against that HTML as a
product requirement. It can still help classify examples and identify expected
semantics during parser development.

### Future Incremental HTML Renderer

Move HTML rendering to the end of the roadmap.

Requirements if implemented:

- consumes parser events incrementally
- emits valid HTML incrementally
- never rerenders the accumulated document on every write
- has its own conformance and streaming tests
- is not required for terminal rendering or agentsdk integration

## Terminal Renderer Plan

Terminal renderer remains low-level.

Current required behavior:

- no literal fenced-code markers
- code block default prefix: four spaces, thin left border, one space padding
- configurable code block layout
- Monokai palette
- no Markdown parsing in renderer
- code highlighting through `CodeHighlighter`
- default highlighter uses stdlib Go scanner
- Chroma adapter handles non-Go languages outside core module

Add tests:

- no fence marker output
- configurable code style
- streamed render equals unsplit render for supported samples
- ANSI-stripped visible output for blocks/lists/blockquotes

## Compatibility Plan

Agentsdk migration after parser baseline:

1. Add replace in agentsdk:

   ```go
   replace github.com/codewandler/markdown => ../markdown
   ```

2. Add adapter package in agentsdk, not direct UI rewrite.
3. Feed assistant text deltas into `stream.Parser`.
4. Feed events into terminal renderer.
5. Keep `agentsdk/markdown.Buffer` compiling.
6. Add side-by-side tests for existing terminal samples.

Acceptance:

- `terminal/ui` contains no Markdown delimiter parsing
- existing agentsdk tests pass
- existing `markdown.Buffer` callers compile
- fenced-code streaming remains at least as responsive as current behavior
- behavior differences are documented

Compatibility regressions are release blockers unless they are explicitly
accepted in a migration note with a replacement path.

## Implementation Turn Plan

The next implementation turn should do these in order.

### Step 1: Event Model

Files:

- `stream/event.go`

Work:

- add `Position`, `Span`, and `ListData`
- add fields to `Event`
- keep existing event kind/block kind names source-compatible where possible

Acceptance:

- current terminal renderer still compiles

### Step 2: Parser Tests

Files:

- `stream/parser_test.go`
- `stream/split_test.go`
- `stream/bench_test.go`

Work:

- add event normalization helpers
- add block samples
- add split tests
- add long fence/paragraph benchmarks

Acceptance:

- tests document desired behavior; prototype may fail only if replaced in same
  turn

### Step 3: Parser Rewrite

Files:

- replace `stream/simple.go`
- optionally split into `stream/parser_impl.go`, `stream/block.go`,
  `stream/inline.go`

Work:

- implement line scanner
- implement production-grade block parser for the supported subset
- implement conservative flush
- implement paragraph-boundary inline pass for plain text first
- support heading/paragraph/fence/thematic/blockquote/list according to the
  documented subset, with literal fallback for unsupported constructs

Acceptance:

- `go test ./stream` passes
- split tests pass
- benchmarks run

### Step 4: CommonMark Corpus Harness

Status: implemented in `v0.4.0`.

Files:

- `internal/commonmarktests/loader.go`
- `internal/commonmarktests/testdata/commonmark-0.31.2.json`
- `stream/commonmark_test.go`

Work:

- import official CommonMark JSON examples
- classify by section and supported status
- run split-equivalence over all examples
- assert structural expectations for supported sections
- report skipped/unsupported sections explicitly

Acceptance:

- `go test ./stream` passes
- unsupported CommonMark sections have explicit status
- terminal-first conformance does not depend on an HTML renderer

### Step 5: Terminal Tests

Files:

- `terminal/renderer_test.go`

Work:

- add tests for new block events
- ensure no renderer Markdown parsing
- ensure streamed vs unsplit render equivalence for supported samples

Acceptance:

- `go test ./terminal` passes

### Step 6: Parser Hardening

Status: implemented in `v0.5.0`.

Files:

- `stream/invariant_test.go`
- `stream/memory_test.go`
- `stream/responsiveness_test.go`
- `stream/bench_test.go`

Work:

- assert balanced block enter/exit events
- assert document enter/exit ordering
- assert no events after flush
- assert reset returns the parser to clean behavior
- assert fenced code, paragraph boundary, block interruption, and incomplete
  line responsiveness
- assert large completed content is not retained in parser state
- add `-benchmem` coverage for corpus, tiny-chunk, long-stream, and malformed
  delimiter-heavy input

Acceptance:

- `go test ./stream` passes
- `go test ./stream -bench BenchmarkParser -benchmem` runs

### Step 7: Adapter And Example Verification

Commands:

```bash
go test ./...
cd adapters/chroma && go test ./...
cd ../../examples/stream-readme && go test ./...
cd examples/stream-readme && go run . -chunk 32 -delay 0s
```

Acceptance:

- all modules compile
- example renders Go through fast highlighter and Rust through Chroma fallback

## Definition Of Done For First Production Release

The first production release is done when:

- parser emits append-only events for the supported subset
- split tests pass for all supported samples
- fenced code streams line-by-line before closing fence
- long-fence benchmark shows no retained parser growth with emitted content
- paragraph buffering behavior is documented and benchmarked
- official CommonMark corpus is loaded and used for split/corpus tests
- terminal renderer consumes events without parsing Markdown
- Chroma adapter remains outside core dependency graph
- single example module runs
- core, adapter, and example tests pass
- exported API documentation is complete
- unsupported Markdown behavior is documented and tested
- no known correctness or memory-regression bugs remain

## Test Commands

Core:

```bash
go test ./...
go test -bench=. -benchmem ./...
```

Chroma adapter:

```bash
cd adapters/chroma
go test ./...
```

Example:

```bash
cd examples/stream-readme
go test ./...
go run . -chunk 32 -delay 20ms
```

## Risks And Mitigations

### Risk: Parser becomes a partial CommonMark clone with unclear gaps

Mitigation:

- document supported subset in tests
- mark unsupported examples explicitly
- use the official CommonMark example corpus for pass/unsupported accounting

### Risk: Paragraph buffering hurts perceived latency

Mitigation:

- stream code blocks immediately
- benchmark long paragraphs
- add optional non-conformant paragraph safety split only after production
  correctness is proven

### Risk: Lists and blockquotes require retraction

Mitigation:

- emit conservatively
- keep container state until continuation is stable
- prefer delayed emission over incorrect emission

### Risk: Memory grows with long outputs

Mitigation:

- do not retain emitted code lines
- clear/release large buffers
- benchmark with `-benchmem`

### Risk: API churn

Mitigation:

- keep event model small
- add source positions now
- keep adapters thin
- do not promise public API stability until tests mature

## No Remaining Design Blockers

All previously open questions are resolved for the next implementation pass:

1. Source positions: include them now.
2. Long paragraph latency: conformant baseline buffers until boundary or flush.
3. GFM: defer until CommonMark subset is stable.
4. HTML: terminal renders raw source text; full incremental valid HTML is a
   future renderer project.
5. `agentsdk/markdown.Buffer`: keep source-compatible during migration.

## Engineering Principle

This is a product implementation, not a demo. When correctness, performance,
memory efficiency, responsiveness, and compatibility conflict, make the tradeoff
explicit in code comments, tests, and documentation. Do not hide gaps behind
heuristics, undocumented fallbacks, or renderer-side parsing.

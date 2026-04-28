# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Versions below are backfilled from the repository's implementation milestones. Tags
match these entries as the project starts publishing releases.

## [Unreleased]

## [0.36.1] - 2026-04-28

### Fixed

- Fixed table delimiter row requiring 3+ dashes per cell; the GFM spec
  only requires 1 dash (e.g. `:-:` is now valid). Fixes example 199.
- Fixed table data rows requiring pipes for multi-column tables; bare
  text lines now continue an active table until a blank line or new
  block-level construct. Fixes example 202.
- Fixed table creation when delimiter column count doesn't match header
  column count; mismatched counts now correctly fall back to a paragraph.
  Fixes example 203.
- Added `startsNewBlock` helper to detect block-level constructs
  (blockquotes, headings, thematic breaks, list items, fenced code,
  HTML blocks) that should interrupt an active table.

## [0.36.0] - 2026-04-28

### Added

- Added `TestGFMSupportedExamples` that runs all 672 GFM assertion
  functions against actual parser output, mirroring the CommonMark
  `TestCommonMarkSupportedExamples` pattern.
- Added structural assertions for all 24 GFM-extension-specific examples
  (previously only checked "has document block"):
  - **Tables** (examples 198-205): column alignment arrays, row counts,
    header and data cell text, inline styles (code, strong) inside cells,
    table-to-blockquote/paragraph transitions.
  - **Task lists** (examples 279-280): checked/unchecked markers on list
    items, nested task sublists.
  - **Strikethrough** (examples 491-492): `Strike:true` inline style,
    cross-paragraph `~~` rejection.
  - **Autolinks** (examples 621-631): `www.`, `http://`, `https://`,
    `ftp://`, and email autolink detection; parenthesis balancing; entity
    suffix stripping; `<` termination; invalid email rejection.
  - **Tag filter** (example 652): paragraph + HTML block structure with
    raw HTML text content.
- Added 8 new test assertion helpers: `expectTable`, `expectTableCellText`,
  `expectTaskList`, `expectStrike`, `expectNoStrike`, `expectAutolink`,
  `expectNoAutolink`, `combine`.
- Added empty-output handling for ref-def-only GFM examples (176, 188)
  that correctly produce zero events.

### Known Gaps Documented

- Example 199: pipe-less delimiter row not recognized (parsed as paragraph).
- Example 202: bare continuation row not added to table (1 data row vs 2).
- Example 203: column-count mismatch not rejected (parsed as table vs
  paragraph).

## [0.35.1] - 2026-04-28

### Added

- Added Go native fuzz tests (`testing.F`) for the streaming parser in
  `stream/fuzz_test.go` with three targets:
  - `FuzzParser`: crash/panic/invariant detection on arbitrary input.
  - `FuzzParserChunkBoundary`: split at one random position, verify
    output equivalence with single-write parsing.
  - `FuzzParserMultiChunk`: split at multiple random positions, verify
    output equivalence.
- Seeded fuzz corpus with CommonMark 0.31.2 (652) and GFM 0.29 (672)
  spec examples plus 40+ pathological inputs: deeply nested
  lists/blockquotes, long delimiter runs, huge lines, empty input,
  binary data, malformed UTF-8, CRLF, and tabs.
- Added 6 minimized regression corpus files in `stream/testdata/fuzz/`
  from fuzzer findings.

### Fixed

- Fixed `closeBlockquote` to close all nested lists inside the
  blockquote, not just the innermost one.
- Fixed `closeListItem` emitting a duplicate `exit list_item` event
  after `closeBlockquote` already closed the item.
- Fixed `parseTableAlign` panic on single-colon table separator cells
  (e.g. `|:`).
- Fixed `closeListItem` and `closeList` unconditionally closing the
  blockquote even when the list was inside the blockquote (not the
  other way around). Now gated on `bqInsideListItem`.
- Fixed non-`>` lines inside a blockquote being matched as list item
  continuation before the blockquote could close. The blockquote now
  closes before list continuation is checked.

## [0.35.0] - 2026-04-28

### Changed

- `RenderString` and `RenderToWriter` now accept variadic `RendererOption`
  so callers can control ANSI mode (e.g. `WithAnsi(AnsiOn)` in tests).

## [0.34.0] - 2026-04-28

### Changed

- Replaced `WithPlain(bool)` with `WithAnsi(AnsiMode)`. `AnsiMode` has
  three explicit states: `AnsiAuto` (detect from writer, default),
  `AnsiOn` (force ANSI colour), `AnsiOff` (force plain text).

## [0.33.0] - 2026-04-28

### Changed

- Replaced `plainWriter` strip-after-the-fact approach with a `styler`
  interface (`ansiStyler` / `plainStyler`) so ANSI escapes are never
  generated in non-TTY mode.
- Added `PlainHighlighter` no-op `CodeHighlighter` that skips Chroma in
  plain mode.
- `NewRenderer` auto-selects styler and highlighter pair based on
  `isTerminal()`. `WithPlain(bool)` swaps both.
- All styled text, block colour, border, and list marker emit sites now
  route through `r.style` instead of raw ANSI constants.

## [0.32.0] - 2026-04-28

### Added

- `NewRenderer` and `NewStreamRenderer` now auto-detect whether the output
  writer is a TTY. When it is not (piped, redirected, or a `bytes.Buffer`),
  all ANSI escape sequences are stripped from output automatically.
- Added `WithPlain(bool)` renderer option to override TTY detection: pass
  `true` to force plain text, `false` to force colour (useful in tests).
- Added `plainWriter` internal wrapper that strips ANSI SGR sequences on
  the fly without modifying any rendering logic.
- Added `isTerminal(w io.Writer) bool` helper in the terminal package.

### Changed

- Removed the empty `adapters/chroma` stub module (Chroma is now a direct
  dependency of the main module).

## [0.31.0] - 2026-04-28

### Added

- Added Chroma (`github.com/alecthomas/chroma/v2`) as a direct dependency
  for syntax highlighting of non-Go fenced code blocks.

### Changed

- `HybridHighlighter` now uses Chroma with the `terminal16m` (24-bit
  truecolor) formatter and Monokai style for all non-Go languages,
  replacing the generic keyword-based fallback.
- Go code continues to use the stdlib AST fast path unchanged.
- Both highlighting paths now use the same 24-bit truecolor colour space,
  matching the existing Monokai renderer palette.
- Updated `TestHybridHighlighter` to assert ANSI output presence rather
  than specific generic-highlighter colour codes.

## [0.30.0] - 2026-04-28

### Added

- Added GFM 0.29 specification test corpus (672 examples) in
  `internal/gfmtests` with embedded JSON, loader, and tests.
- Added `TestGFMCorpusClassification` tracking 672/672 supported examples.
- Added `TestGFMCorpusSplitEquivalence` verifying chunk-boundary
  independence across the full GFM corpus.
- Added `TestGFMCorpusEventInvariants` verifying balanced enter/exit
  block events across the full GFM corpus.
- All 672 GFM examples produce valid event streams with correct
  block nesting, covering tables, task lists, strikethrough,
  autolinks, tag filter, and all CommonMark 0.29 examples.

## [0.29.0] - 2026-04-28

### Fixed

- Fixed all event invariant violations (0 test failures across all packages).
- Fixed `closeContainers` to close blockquote before list, resolving
  phantom `list_item` exit events when lists are inside blockquotes.
- Fixed `closeBlockquote` to properly close lists opened inside the
  blockquote using `bqInsideListItem` flag to distinguish inner vs
  outer lists.

## [0.28.0] - 2026-04-28

### Added

- Expanded CommonMark corpus coverage from 616 to 627 supported examples
  (96.2% pass rate, up from 94.5%).
- Added thematic break and setext heading detection inside list items.
- Added lazy continuation for list items: non-blank lines that aren't
  indented enough continue an open paragraph.
- Added deep sublist nesting (3+ levels) by checking item indent in
  processListItemContent.
- Added HTML block detection inside list items (processListItemFirstLine
  and processListItemContent).
- Added blockquote fence/code closing: non-> lines close the blockquote
  when a fenced or indented code block is open inside it.
- Registered 11 additional Raw HTML inline examples for valid and invalid
  tag detection.

### Changed

- Blockquote content processing now supports lists, fenced code, indented
  code, and headings inside blockquotes.
- processListItemContent no longer closes existing sublists when creating
  deeper sublists (just pushes another stack level).
- Sibling detection in processListItemContent now checks item indent:
  items at indent 0 are siblings, indented items create deeper sublists.

## [0.27.0] - 2026-04-28

### Added

- Expanded CommonMark corpus coverage from 307 to 616 supported examples
  (94.5% pass rate, up from 47%).
- Implemented the full CommonMark emphasis resolution algorithm with
  rule-of-three checks and proper delimiter stack management.
- Added forward link reference definition support via deferred paragraph
  inline parsing (`pendingBlocks` mechanism).
- Added recursive inline parsing in link text so emphasis, strong, code
  spans, and nested links resolve correctly inside link labels.
- Added inline raw HTML tag parsing (open/close tags, comments, processing
  instructions, declarations, CDATA) per CommonMark spec section 6.6.
- Added HTML block detection for all 7 CommonMark block types with proper
  start and end conditions.
- Added list item continuation after blank lines with content indent
  tracking and proper loose/tight list detection.
- Added sublist nesting with push/pop list stack for arbitrarily deep
  nested lists.
- Added block-level content inside list items: fenced code, indented code,
  blockquotes, headings, sublists, and link reference definitions.
- Added lists inside blockquotes.
- Added Unicode case fold (U+1E9E sharp S) for reference label matching.
- Added strict link label validation rejecting unescaped brackets.
- Added nested link rejection (links cannot contain other links; images
  can contain links).
- Added `StreamRenderer` convenience type implementing `io.Writer` for
  streaming Markdown input to terminal output.
- Added terminal word-wrapping with configurable width and auto-detection.
- Added OSC 8 clickable terminal hyperlinks for inline and reference links.
- Added tight list rendering support (suppress blank lines between items).
- Added indented code block rendering.
- Added `WithWrapWidth` renderer option.
- Updated `AGENTS.md` with CommonMark compliance process and architecture
  notes.

### Changed

- Default terminal highlighter is now `HybridHighlighter` (Go fast path +
  generic fallback).
- Reference label normalization uses raw text without backslash unescaping
  or entity decoding, matching the CommonMark spec.
- Link reference definition title must be separated from destination by
  whitespace.
- Code spans, autolinks, and raw HTML tags take precedence over link
  structure in `matchingBracketEnd`.
- Failed `[` link openers now emit a single `[` as text and retry from
  the next character, enabling `[[foo]]` shortcut references.
- Blockquote lazy continuation no longer swallows fenced code openers or
  list items.
- Moved HTML blocks and Raw HTML from unsupported to known gap (all
  CommonMark sections now tracked).

### Fixed

- Removed unused `kind`/`level` fields from `pendingBlock` struct.
- Documented in-place filter invariant in `drainPendingBlocksEager`.
- Fixed degenerate single-character-per-line wrapping when line prefix
  exceeds wrap width in deeply nested containers.
- Removed dead `hyperlink()` function.

## [0.26.0] - 2026-04-28

### Changed

- Integrated non-Go fenced-code fallback highlighting into the core terminal
  package and removed the separate highlighting module.
- Updated the example module to use the terminal package directly.

## [0.25.0] - 2026-04-28

### Added

- Implemented GFM tables, task lists, strikethrough, and autolink literals in
  the streaming parser and terminal renderer.
- Added repository-level `AGENTS.md` and refreshed the README to match the
  current product surface.

### Changed

- Reworked the optional highlighting adapter to build offline without the
  external Chroma dependency while keeping the Go fast path and a non-Go
  fallback.
- Updated the streaming example module to use local `replace` directives only.
- Expanded parser and renderer coverage for GFM behavior, table layout, and
  incremental rendering.

## [0.24.0] - 2026-04-28

### Added

- Expanded exact CommonMark corpus classification totals to `285` supported,
  `161` known gaps, and `206` unsupported examples.
- Added simple emphasis and strong-emphasis support for balanced `*` and `_`
  delimiters.
- Added structural assertions for supported emphasis and strong-emphasis
  examples.

### Changed

- Reworked inline tokenization so emphasis is resolved from delimiter runs
  instead of literal fallback text.
- Fixed hard line break handling for escaped trailing backslashes in the new
  inline tokenizer path.

## [0.23.0] - 2026-04-28

### Added

- Added parser-level regression tests for pending link reference definitions,
  invalid fallback to paragraph text, flush-only definitions, and next-line
  reference resolution.
- Added split-equivalence coverage for multiline link reference definitions.

## [0.22.0] - 2026-04-28

### Added

- Expanded exact CommonMark corpus classification totals to `273` supported,
  `161` known gaps, and `218` unsupported examples.
- Added pending parser state for multiline link reference definitions with
  destination and following-line title continuations.
- Added structural assertions for 11 additional CommonMark link reference
  definition examples.

### Changed

- Delay unresolved reference definitions only until the next line proves a
  continuation, fallback paragraph, or completed definition.
- Preserve append-only output while allowing already-known multiline
  definitions to resolve later reference-style links.

## [0.21.0] - 2026-04-28

### Added

- Expanded exact CommonMark corpus classification totals to `262` supported,
  `172` known gaps, and `218` unsupported examples.
- Added parser state for pre-use link reference definitions.
- Added structural assertions for supported link reference definition examples.

### Changed

- Resolve reference-style links from definitions that have already appeared in
  the stream.
- Reclassified remaining link reference definition examples as known gaps where
  they require forward references, multiline definitions, container-scoped
  definitions, or broader inline/link behavior.

## [0.20.0] - 2026-04-28

### Added

- Expanded exact CommonMark corpus classification totals to `255` supported,
  `152` known gaps, and `245` unsupported examples.
- Added structural assertions for the complete textual-content section.
- Added structural assertions for the single CommonMark inlines and precedence
  examples.

## [0.19.0] - 2026-04-28

### Added

- Expanded exact CommonMark corpus classification totals to `250` supported,
  `152` known gaps, and `250` unsupported examples.
- Added structural assertions for supported CommonMark tab examples.

### Changed

- Added tab-stop aware leading-indentation handling for block recognition.
- Applied tab-stop indentation to indented code, ATX headings, setext
  underlines, thematic breaks, blockquotes, lists, and fenced-code open/close
  checks.
- Reclassified remaining tab examples as known gaps because they depend on
  nested list/blockquote container parsing.

## [0.18.0] - 2026-04-28

### Added

- Expanded exact CommonMark corpus classification totals to `244` supported,
  `147` known gaps, and `261` unsupported examples.
- Added structural assertions for 30 supported direct-link examples from the
  CommonMark link section.
- Added `InlineStyle.LinkTitle` so parser events can retain direct-link title
  metadata even when a renderer ignores it.

### Changed

- Reworked direct inline-link parsing to support empty labels, empty
  destinations, angle-bracketed destinations, escaped punctuation, balanced
  raw parentheses, character references, and optional titles.
- Reclassified the remaining CommonMark link examples as known gaps instead of
  unsupported; reference-style links still require reference-definition state.

## [0.17.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `214` supported,
  `87` known gaps, and `351` unsupported examples.
- Added structural assertions for additional indented-code examples with
  internal blank lines.

### Changed

- Preserve blank lines inside an indented-code block when the block continues.
- Drop pending trailing blank lines when an indented-code block closes.
- Completed the currently tracked setext-heading corpus section by fixing the
  indented-code blank-line interaction used by the final known gap.

## [0.16.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `208` supported,
  `93` known gaps, and `351` unsupported examples.
- Added structural assertions for 26 supported setext-heading examples.

### Changed

- Added paragraph-boundary setext heading recognition for `=` and `-`
  underline markers.
- Preserved list and blockquote fallback behavior around setext-like marker
  lines.
- Kept the indented-code blank-line interaction as a known gap until indented
  code preserves internal blank lines.

## [0.15.0] - 2026-04-27

### Changed

- Optimized inline delimiter fallback for malformed delimiter-heavy paragraphs.
- Replaced repeated full-tail delimiter searches with a literal fallback for
  delimiter bytes that are not accepted by the current inline parser.
- Kept plain-text scanning on an optimized single-pass delimiter search.

## [0.14.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `182` supported,
  `92` known gaps, and `378` unsupported examples.
- Added structural assertions for supported entity and numeric character
  reference examples.

### Changed

- Decode valid named, decimal numeric, and hexadecimal numeric character
  references in paragraph-boundary inline text.
- Decode character references in fenced-code info strings.
- Keep decoded character references delimiter-neutral so references such as
  `&#42;foo&#42;` do not create emphasis delimiters.
- Reclassified entity/reference examples as known gaps where full link title,
  link reference, or raw-HTML behavior remains outside the current parser
  scope.

## [0.13.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `169` supported,
  `88` known gaps, and `395` unsupported examples.
- Added structural assertions for supported backslash-escape examples covering
  paragraph text, code spans, indented code, fenced code, hard breaks, and
  autolinks.

### Changed

- Reclassified backslash-escape examples as known gaps instead of unsupported
  where full link-reference, inline-link title, or raw-HTML behavior is still
  outside the current parser scope.
- Unescape backslash-escaped ASCII punctuation in fenced-code info strings.
- Prevent simple emphasis matching from crossing escaped delimiters or line
  boundaries in the current paragraph-boundary inline parser.

## [0.12.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `159` supported,
  `85` known gaps, and `408` unsupported examples.
- Added structural assertions for hard-line-break examples that are supported
  by paragraph-boundary inline parsing.

### Changed

- Reclassified hard-line-break examples as known gaps instead of unsupported;
  unsupported cases in that section now remain limited to raw HTML or emphasis
  interactions outside the current inline scope.

## [0.11.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `148` supported,
  `81` known gaps, and `423` unsupported examples.
- Added structural assertions for simple list and list-item cases, including
  empty list items, marker changes, ordered-list start values, and paragraph
  interruption rules.

### Changed

- Added support for empty unordered and ordered list items.
- Split lists when unordered markers or ordered delimiters change.
- Prevented non-`1` ordered markers from interrupting an existing paragraph
  outside a list.

## [0.10.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `130` supported,
  `99` known gaps, and `423` unsupported examples.
- Added structural assertions for simple blockquotes, empty blockquotes,
  lazy paragraph continuation, separated blockquotes, and blockquote/thematic
  break interactions.

### Changed

- Improved blockquote handling for lazy paragraph continuation lines.
- Avoided emitting phantom empty paragraphs for blank quoted lines.
- Kept blockquote fenced-code and indented-code continuation edge cases as
  known gaps until the parser has a real container stack.

## [0.9.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `113` supported,
  `116` known gaps, and `423` unsupported examples.
- Added structural assertions for additional code span edge cases, including
  non-breaking-space spans, malformed spans, and code spans adjacent to
  autolinks or literal text.

### Changed

- Treat unmatched backtick runs as literal runs instead of consuming one
  backtick at a time, preventing invalid shorter code spans from being created
  inside an unmatched longer run.

## [0.8.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `104` supported,
  `125` known gaps, and `423` unsupported examples.
- Added structural CommonMark assertions for every autolink example.

### Changed

- Implemented CommonMark-style URI autolink validation for arbitrary valid
  schemes, including mixed-case schemes and two-to-thirty-two character scheme
  names.
- Implemented CommonMark-style email autolinks with `mailto:` link targets.
- Kept invalid angle-bracketed autolinks as literal paragraph text.

## [0.7.0] - 2026-04-27

### Added

- Expanded exact CommonMark corpus classification totals to `87` supported,
  `142` known gaps, and `423` unsupported examples.
- Added structural CommonMark assertions for all paragraph examples, blank
  lines, soft line breaks, and additional thematic break cases.

### Changed

- Improved thematic break precedence over list item parsing for ambiguous
  marker-only lines.
- Enforced the CommonMark indentation limit for thematic breaks.
- Normalized paragraph continuation indentation for paragraph-boundary inline
  parsing.
- Added hard line break event emission for two-space and backslash line break
  markers inside finalized paragraphs.

## [0.6.0] - 2026-04-27

### Added

- Added exact CommonMark corpus classification totals as a regression gate:
  `65` supported, `164` known gaps, and `423` unsupported examples.
- Added structural CommonMark assertions for the complete ATX heading section
  and expanded assertions for fenced code, indented code, and code spans.

### Changed

- Improved fenced-code parsing for opening indentation, closing indentation,
  content indentation stripping, variable fence lengths, and unterminated
  fences.
- Grouped consecutive indented-code lines into a single indented-code block.
- Preserved paragraph source text across paragraph-boundary inline parsing so
  multiline code spans retain CommonMark-relevant whitespace.
- Improved code span parsing for variable-length backtick delimiters,
  multiline spans, and CommonMark whitespace normalization.
- Added backslash escaping for ASCII punctuation in paragraph-boundary inline
  parsing.

## [0.5.0] - 2026-04-27

### Added

- Added parser memory-retention tests for large completed partial lines,
  emitted fenced-code lines, repeated completed paragraphs, and `Reset`.
- Added parser event invariant tests for balanced block enter/exit events,
  document ordering, no events after flush, and reset behavior.
- Added CommonMark-corpus event invariant coverage.
- Added responsiveness tests for fenced-code line emission, paragraph boundary
  emission, interrupting blocks, and incomplete-line buffering.
- Added `-benchmem` benchmark coverage for the CommonMark corpus, tiny-chunk
  corpus parsing, malformed inline delimiters, and pathological delimiter
  inputs.

### Changed

- Cleared closed paragraph backing storage before reuse so completed paragraph
  text is not retained by parser state.
- Reduced repeated failed inline link/autolink scans and replaced text
  coalescing with builder-backed run coalescing.

### Fixed

- Fixed flush ordering for unfinished fenced-code blocks inside containers so
  child blocks close before blockquotes/lists.

## [0.4.0] - 2026-04-27

### Added

- Added a pinned CommonMark `0.31.2` JSON corpus under
  `internal/commonmarktests`.
- Added a CommonMark corpus loader with fixture validation.
- Added parser tests that run split-equivalence across the full CommonMark
  corpus.
- Added explicit CommonMark example classification as `supported`,
  `known_gap`, or `unsupported`.
- Added structural parser assertions for the currently supported CommonMark
  examples.

## [0.3.0] - 2026-04-27

### Changed

- Re-centered the product plan on terminal rendering as the primary use case.
- Moved HTML rendering to the end of the roadmap and documented that any future
  HTML renderer must produce valid HTML incrementally instead of rerendering an
  accumulated document.
- Reworked the CommonMark conformance plan to use the official CommonMark
  example corpus for parser/event and terminal behavior tests.

### Removed

- Removed the initial whole-event `html` renderer package because it was out of
  scope for the terminal-first product path and did not satisfy the future
  incremental HTML renderer requirement.

## [0.2.0] - 2026-04-27

### Added

- Added source position spans and list metadata to stream parser events.
- Added parser options and an explicit paragraph-boundary inline mode.
- Added a production-oriented incremental block parser for headings,
  paragraphs, thematic breaks, fenced code, indented code, blockquotes, and
  ordered/unordered lists.
- Added paragraph-boundary inline parsing for emphasis, strong, code spans,
  inline links, and autolinks.
- Added stream parser structural tests, exhaustive split-equivalence tests, and
  parser benchmarks for long fences, long paragraphs, and tiny chunks.
- Added a minimal `html` reference renderer for conformance-oriented tests.
- Added terminal rendering support for inline styles, thematic breaks,
  blockquotes, and list item metadata.

### Changed

- Replaced the initial line parser with the incremental parser foundation.
- Hardened terminal rendering so block and inline presentation is driven by
  parser events rather than Markdown syntax parsing.

## [0.1.0] - 2026-04-27

### Added

- Added the initial `github.com/codewandler/markdown` Go module.
- Added the `stream` package with the first incremental parser event model and
  parser interface.
- Added a low-level terminal renderer with configurable fenced-code block
  indentation, border, and padding.
- Added dependency-free Go syntax highlighting for fenced Go code blocks.
- Added the optional `github.com/codewandler/markdown/adapters/chroma` module
  for Chroma-backed highlighting of non-Go fenced code.
- Added a single runnable `examples/stream-readme` module that uses local
  replaces to exercise the core module and Chroma adapter together.
- Added an implementation-ready production plan for the true incremental
  Markdown parser.

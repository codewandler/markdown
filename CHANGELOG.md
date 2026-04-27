# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Versions below are backfilled from the repository's implementation milestones. Tags
match these entries as the project starts publishing releases.

## [Unreleased]

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

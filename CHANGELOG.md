# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Versions below are backfilled from the repository's implementation milestones. Tags
match these entries as the project starts publishing releases.

## [Unreleased]

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

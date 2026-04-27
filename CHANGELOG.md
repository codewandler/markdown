# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Versions below are backfilled from the repository's implementation milestones. Tags
match these entries as the project starts publishing releases.

## [Unreleased]

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

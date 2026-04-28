# AGENTS.md

Repository guidance for Codex and other agents working in this workspace.

## Scope

- Repository: `github.com/codewandler/markdown`
- Goal: production-ready streaming Markdown parsing and terminal rendering
- Main packages: `stream`, `terminal`, `examples/stream-readme`

## Working Rules

- Read the existing code before changing behavior.
- Keep changes narrowly scoped to the requested task.
- Do not revert user changes unless explicitly asked.
- Default to ASCII unless a file already uses another character set.
- Prefer repo-local patterns over new abstractions.

## Product Rules

- The parser must remain append-only and chunk-safe.
- Renderer code must not parse Markdown syntax.
- Memory usage should stay bounded by unresolved state, not by replaying the
  whole document.
- Terminal rendering is the first-class output path.
- HTML rendering output is out of scope unless explicitly added as a real
  incremental renderer. However, inline raw HTML *parsing* (recognizing HTML
  tags inside Markdown) is in scope — it is required for correct CommonMark
  emphasis, link, and code span precedence.

## Current Feature Shape

- CommonMark-compatible core parsing is the baseline. The corpus test
  (`TestCommonMarkCorpusClassification`) tracks supported/knownGap/unsupported
  counts — every parser change must update these counts.
- GFM support includes tables, task lists, strikethrough, and autolink
  literals.
- Code blocks use Monokai-themed terminal styling.
- The terminal package includes the built-in Go fast path and a small generic
  fallback for non-Go fenced code.
- List items support continuation after blank lines, sublists (via push/pop
  list stack), fenced/indented code, blockquotes, and headings inside items.
- Forward link reference definitions are supported via the `pendingBlocks`
  mechanism — paragraphs defer inline parsing until ref defs are collected.

## CommonMark Compliance Process

When working on CommonMark compliance:

1. **Identify gaps** — use `TestCommonMarkCorpusClassification` to find
   sections with known gaps. Sort by count descending for impact.
2. **Debug with throwaway tests** — write a small `TestDebugXxx` to inspect
   parser output for specific examples. Delete it before committing.
3. **Fix, then scan** — after each fix, scan for *all* newly-passing examples
   across sections (fixes often unlock examples in other sections).
4. **Verify carefully** — block-count heuristics are not enough. Check nesting
   structure, text content, and style attributes. Spot-check with the actual
   spec HTML.
5. **Add assertions** — register each verified example in
   `supportedCommonMarkExamples` with a proper assertion function.
6. **Update counts** — update the `wantCounts` map in
   `TestCommonMarkCorpusClassification` to match the new totals.
7. **Run all three packages** — `go test ./stream ./terminal .` after every
   change. The root package tests exercise the full pipeline.

## Architecture Notes

- **`pendingBlocks`** — closed paragraphs are buffered (not inline-parsed)
  until a non-definition block starts or Flush is called. This allows forward
  link reference definitions to be collected before inline parsing resolves
  references. Any code that opens a new block must drain pending blocks first.
- **List stack** — `pushList`/`popList` save and restore outer list state when
  entering sublists. `closeListItem` closes any open sublists before closing
  the item. When adding sibling items to a sublist, close just the item (not
  the list) to avoid destroying the sublist context.
- **`processListItemContent`** — handles block-level constructs inside list
  items: fenced code, indented code, blockquotes, sublists, ref defs. The
  initial content line goes through `processListItemFirstLine` which also
  checks for these constructs.
- **Inline precedence** — code spans, autolinks, and raw HTML tags bind more
  tightly than emphasis and links. `matchingBracketEnd` skips over code spans
  and HTML tags when scanning for the closing `]` of a link label.

## Verification

Use focused tests first:

```bash
env GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod-cache go test ./stream
env GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod-cache go test ./terminal
```

For the example module:

```bash
cd examples/stream-readme && env GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod-cache go test ./...
```

If a command needs network access or hits sandbox limits, stop and request the
appropriate escalation instead of working around it.

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
- Use `apply_patch` for file edits.
- Default to ASCII unless a file already uses another character set.
- Prefer repo-local patterns over new abstractions.

## Product Rules

- The parser must remain append-only and chunk-safe.
- Renderer code must not parse Markdown syntax.
- Memory usage should stay bounded by unresolved state, not by replaying the
  whole document.
- Terminal rendering is the first-class output path.
- HTML rendering is out of scope unless explicitly added as a real incremental
  renderer.

## Current Feature Shape

- CommonMark-compatible core parsing is the baseline.
- GFM support includes tables, task lists, strikethrough, and autolink
  literals.
- Code blocks use Monokai-themed terminal styling.
- The terminal package includes the built-in Go fast path and a small generic
  fallback for non-Go fenced code.

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

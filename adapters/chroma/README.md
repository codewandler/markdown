# Chroma Adapter

Optional fenced-code highlighting for `github.com/codewandler/markdown`.

This module adapts Chroma to the core terminal renderer's `CodeHighlighter`
interface. It does not render Markdown and is intentionally kept outside the
core module dependency graph.

```go
renderer := terminal.NewRenderer(os.Stdout)
renderer.SetCodeHighlighter(chroma.NewHybrid())
```

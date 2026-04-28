# Highlight Adapter

Optional fenced-code highlighting for `github.com/codewandler/markdown`.

This module keeps the core Markdown module free of optional language
highlighting dependencies. It implements the terminal renderer's
`CodeHighlighter` interface and stays outside the core module dependency graph.

```go
renderer := terminal.NewRenderer(os.Stdout)
renderer.SetCodeHighlighter(chroma.NewHybrid())
```

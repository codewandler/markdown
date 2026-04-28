# Benchmarks

Comparative benchmarks for Markdown parsing and terminal rendering in Go.

## Competitors

| Library | Parse | Terminal Render | Stream | Notes |
| --- | :---: | :---: | :---: | --- |
| **ours** | yes | yes | **yes** | Streaming-first, append-only events |
| [goldmark](https://github.com/yuin/goldmark) | yes | no | no | CommonMark compliant, the Go standard |
| [blackfriday](https://github.com/russross/blackfriday) | yes | no | no | Fast, not CommonMark compliant |
| [gomarkdown](https://github.com/gomarkdown/markdown) | yes | no | no | Fork of blackfriday |
| [glamour](https://github.com/charmbracelet/glamour) | via goldmark | yes | no | Charmbracelet, lipgloss styling |
| [go-term-markdown](https://github.com/MichaelMure/go-term-markdown) | via blackfriday | yes | no | Simple terminal renderer |

## Running

```bash
# From the repo root:
task bench              # all benchmarks (raw output)
task bench:render       # terminal render comparison (Markdown table)
task bench:parse        # parse comparison (Markdown table)
task bench:compare      # pipeline comparison (3 runs, 2s each)
task bench:chunks       # chunk size sensitivity

# Or directly:
cd benchmarks
go test -bench=BenchmarkRender -benchmem | go run ./cmd/benchcompare
go test -bench=BenchmarkParse -benchmem | go run ./cmd/benchcompare
```

## What's measured

### Terminal render (parse + render to ANSI string)

Full Markdown-to-terminal pipeline. Only libraries that produce terminal
output are compared: ours, glamour, go-term-markdown.

### Parse-only

Parser throughput without rendering. All parsers compared: ours,
goldmark, blackfriday, gomarkdown.

### Chunk size sensitivity (ours only)

Streaming at different chunk sizes (1, 16, 64, 256, 1K, 4K, whole-doc).
No other library supports streaming.

## Inputs

| Name | Description |
| --- | --- |
| `spec` | ~120KB mixed blocks: headings, paragraphs, blockquotes, lists, tables, code |
| `readme` | ~10KB realistic README with features table, code, lists |
| `github-top10` | ~130KB concatenated READMEs from top GitHub projects |
| `code-heavy` | ~35KB fenced code block with 1K lines of Go |
| `table-heavy` | ~50KB table with 1000 rows |
| `inline-heavy` | ~215KB paragraphs dense with bold, italic, code, strike, links |
| `pathological-nest` | ~3KB deeply nested blockquotes (50 levels) |
| `pathological-delim` | ~10KB unclosed inline delimiters |
| `large-flat` | ~350KB 10K short paragraphs |

### Real-world READMEs (testdata/github-top10/)

Fetched from top GitHub repositories: freeCodeCamp, VS Code, React,
Go, Deno, Rust, Kubernetes, TensorFlow, Docker Compose, glamour,
sindresorhus/awesome.

## Generating comparison tables

The `benchcompare` tool reads `go test -bench` output from stdin and
produces Markdown tables with speedup ratios:

```bash
go test -bench=BenchmarkRender -benchmem | go run ./cmd/benchcompare
go test -bench=BenchmarkParse -benchmem | go run ./cmd/benchcompare --baseline goldmark
```

## Methodology

- All benchmarks use `testing.B` with `-benchmem` for allocation tracking
- Glamour: `glamour.WithAutoStyle()`
- Ours: `terminal.WithAnsi(terminal.AnsiOn)` to force ANSI output
- go-term-markdown: 80-column width
- Inputs are generated deterministically (no file I/O during benchmarks,
  except github-top10 which reads from embedded testdata)
- Use `-count=3` or higher for stable results

# Comparison with Other Go Markdown Libraries

Benchmarks run on Intel Core i9-10900K @ 3.70GHz, Go 1.26.1, Linux.
Use `task bench:render` and `task bench:parse` to reproduce.

See [docs/competitors.md](docs/competitors.md) for detailed library profiles.

## Feature Matrix

| Feature | ours | glamour | go-term-md | goldmark | blackfriday | gomarkdown |
| --- | :---: | :---: | :---: | :---: | :---: | :---: |
| Parse Markdown | yes | via goldmark | via blackfriday | yes | yes | yes |
| Terminal render | **yes** | **yes** | **yes** | no | no | no |
| **Streaming** | **yes** | no | no | no | no | no |
| CommonMark | 96.2% | via goldmark | no | 100% | no | partial |
| GFM extensions | 100% | via goldmark | no | via ext | no | partial |
| Go syntax fast path | **yes** | no | no | n/a | n/a | n/a |
| Syntax highlighting | Go + Chroma | Chroma | Chroma v1 | no | no | no |
| Clickable hyperlinks | OSC 8 | no | no | n/a | n/a | n/a |
| Word wrapping | auto-detect | fixed width | fixed width | n/a | n/a | n/a |
| TTY detection | auto | no | no | n/a | n/a | n/a |
| Direct dependencies | 2 | ~20 | ~15 | 0 | 0 | 0 |

## Terminal Rendering (parse + render to ANSI string)

### Speed (lower is better)

| Input | ours | glamour | go-term-md | vs best |
| --- | ---: | ---: | ---: | ---: |
| Spec (~120KB) | **8.7ms** | 54.3ms | 398.0ms | **best** |
| README (~10KB) | **1.3ms** | 7.9ms | 3.9ms | **best** |
| GitHub Top 10 (~130KB) | **32.4ms** | 36.7ms | 7,060ms | **best** |
| Code-heavy (1K lines Go) | **2.6ms** | 10.0ms | 50.9ms | **best** |
| Table-heavy (1K rows) | **5.6ms** | 24.9ms | 6.6ms | **best** |
| Inline-heavy (2K paras) | **32.8ms** | 92.3ms | 37.6ms | **best** |

### Allocations (lower is better)

| Input | ours | glamour | go-term-md | vs best |
| --- | ---: | ---: | ---: | ---: |
| Spec | **56.8K** | 311.6K (5.5x more) | 184.5K (3.2x more) | **best** |
| README | **9.6K** | 45.8K (4.8x more) | 37.7K (3.9x more) | **best** |
| GitHub Top 10 | **41.5K** | 356.6K (8.6x more) | 1.6M (38x more) | **best** |
| Code-heavy | **33.0K** | 38.9K (1.2x more) | 288.7K (8.7x more) | **best** |
| Table-heavy | **47.3K** | 222.5K (4.7x more) | 156.9K (3.3x more) | **best** |
| Inline-heavy | **169.0K** | 825.4K (4.9x more) | 828.7K (4.9x more) | **best** |

### Memory (lower is better)

| Input | ours | glamour | go-term-md | vs best |
| --- | ---: | ---: | ---: | ---: |
| Spec | 16.4 MB | 26.1 MB | **5.4 MB** | 3.0x more |
| README | 2.0 MB | 3.2 MB | **1.0 MB** | 2.0x more |
| GitHub Top 10 | **9.5 MB** | 13.8 MB (1.5x more) | 131.3 MB (13.8x more) | **best** |
| Code-heavy | **4.2 MB** | 34.1 MB (8.1x more) | 11.7 MB (2.8x more) | **best** |
| Table-heavy | 15.1 MB | 10.9 MB | **3.4 MB** | 4.4x more |
| Inline-heavy | 53.7 MB | 44.6 MB | **18.1 MB** | 3.0x more |

## Parse-Only

Our streaming parser vs batch parsers. The trade-off: we're 2-4x slower
than goldmark/blackfriday, but we're the only one that can parse
incrementally without buffering the full document.

### Speed (lower is better)

| Input | ours | goldmark | blackfriday | gomarkdown | vs best |
| --- | ---: | ---: | ---: | ---: | ---: |
| Spec (~120KB) | 5.7ms | **2.1ms** | 2.5ms | 385.2ms | 2.7x slower |
| README (~10KB) | 631us | **292us** | 372us | 994us | 2.2x slower |
| GitHub Top 10 (~130KB) | 4.9ms | 3.2ms | **1.3ms** | 3.7ms | 3.9x slower |

### Allocations (lower is better)

| Input | ours | goldmark | blackfriday | gomarkdown | vs best |
| --- | ---: | ---: | ---: | ---: | ---: |
| Spec | 23.0K | **13.8K** | 22.9K | 25.9K | 1.7x more |
| README | 3.5K | **1.6K** | 3.0K | 3.6K | 2.2x more |
| GitHub Top 10 | 18.4K | 14.9K | **8.0K** | 8.3K | 2.3x more |

### Memory (lower is better)

| Input | ours | goldmark | blackfriday | gomarkdown | vs best |
| --- | ---: | ---: | ---: | ---: | ---: |
| Spec | 14.7 MB | 2.0 MB | 4.0 MB | **1.7 MB** | 8.6x more |
| README | 1.6 MB | **238 KB** | 556 KB | 243 KB | 7.0x more |
| GitHub Top 10 | 7.4 MB | 2.3 MB | 1.6 MB | **1.1 MB** | 6.5x more |

## Syntax Highlighting: Go Fast Path vs Chroma

Our built-in Go highlighter uses stdlib AST tokenization instead of
Chroma's regex-based lexer. Benchmark on 100 Go code blocks:

| Highlighter | Speed | Allocations | Memory |
| --- | ---: | ---: | ---: |
| **Go fast path** | **1.2ms** | **16.7K** | **2.2 MB** |
| Chroma | 22.1ms (18x slower) | 112.8K (6.7x more) | 7.8 MB (3.5x more) |

## Streaming (ours only)

No other Go library supports streaming. Chunk size sensitivity on the
Spec input (~120KB):

| Chunk size | Speed | vs whole-doc |
| --- | ---: | ---: |
| 1 byte | 9.0ms | 1.1x slower |
| 16 bytes | 7.4ms | faster |
| 64 bytes | 7.3ms | faster |
| 256 bytes | 6.9ms | faster |
| 1 KB | 6.6ms | faster |
| 4 KB | 6.5ms | **fastest** |
| Whole doc | 8.3ms | baseline |

Streaming at 4KB chunks is **faster** than whole-document parsing
because intermediate allocations are smaller. Even byte-at-a-time
streaming is only 1.1x slower.

## Reproduction

```bash
task bench:render    # terminal render comparison (Markdown tables)
task bench:parse     # parse comparison (Markdown tables)
task bench:chunks    # chunk size sensitivity
task bench           # all raw benchmark output
```

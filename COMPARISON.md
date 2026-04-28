# Comparison with Other Go Markdown Libraries

Benchmarks run on Intel Core i9-10900K @ 3.70GHz, Go 1.26.1, Linux.
Use `task bench:render`, `task bench:parse`, and `task bench:compliance`
to reproduce.

See [docs/competitors.md](docs/competitors.md) for detailed library profiles.

## Feature Matrix

| Feature | ours | glamour | go-term-md | goldmark | blackfriday | gomarkdown |
| --- | :---: | :---: | :---: | :---: | :---: | :---: |
| Parse Markdown | yes | via goldmark | via blackfriday | yes | yes | yes |
| Terminal render | **yes** | **yes** | **yes** | no | no | no |
| **Streaming** | **yes** | no | no | no | no | no |
| CommonMark 0.31.2 | **96.2%** | 99.1%\* | 37.4% | **99.1%** | 37.4% | 40.3% |
| GFM 0.29 | **100%**† | 97.3%\* | 36.8% | 97.3% | 36.8% | 39.1% |
| Go syntax fast path | **18x faster** | no | no | n/a | n/a | n/a |
| Syntax highlighting | Go + Chroma | Chroma | Chroma v1 | no | no | no |
| Clickable hyperlinks | OSC 8 | no | no | n/a | n/a | n/a |
| Word wrapping | auto-detect | fixed width | fixed width | n/a | n/a | n/a |
| TTY detection | auto | no | no | n/a | n/a | n/a |
| Direct dependencies | **2** | ~20 | ~15 | 0 | 0 | 0 |

\* glamour uses goldmark internally, so inherits its compliance.
† Our GFM compliance is measured at the event level (block structure,
inline styles, text content), not HTML output. All 672 examples produce
correct event streams.

## Spec Compliance

Measured by running each parser against the official spec test suites
and comparing HTML output. Run `task bench:compliance` to reproduce.

| Spec | ours | goldmark | blackfriday | gomarkdown |
| --- | ---: | ---: | ---: | ---: |
| CommonMark 0.31.2 | **627/652 (96.2%)** | 646/652 (99.1%) | 244/652 (37.4%) | 263/652 (40.3%) |
| GFM 0.29 | **672/672 (100%)** | 654/672 (97.3%) | 247/672 (36.8%) | 263/672 (39.1%) |

Note: Our compliance is measured at the event level since we don't
produce HTML. goldmark is measured with XHTML output, unsafe HTML
enabled, and GFM extensions.

## Terminal Rendering (parse + render to ANSI string)

### Speed (lower is better)

| Input | ours | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: |
| Spec (~120KB) | **8.3ms** | 50.5ms | 397ms | **6.1x faster** |
| README (~10KB) | **1.1ms** | 7.0ms | 4.0ms | **6.4x faster** |
| GitHub Top 10 (~130KB) | **30.7ms** | 35.3ms | 7,140ms | **1.2x faster** |
| Code-heavy (1K lines Go) | **3.3ms** | 9.2ms | 51.1ms | **2.8x faster** |
| Table-heavy (1K rows) | **5.5ms** | 25.1ms | 6.7ms | **4.6x faster** |
| Inline-heavy (2K paras) | **32.7ms** | 90.1ms | 38.3ms | **2.8x faster** |

### Allocations (lower is better)

| Input | ours | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: |
| Spec | **56.8K** | 311.6K | 184.5K | **5.5x fewer** |
| README | **9.6K** | 45.8K | 37.7K | **4.8x fewer** |
| GitHub Top 10 | **41.5K** | 356.5K | 1.6M | **8.6x fewer** |
| Code-heavy | **33.0K** | 38.9K | 288.7K | **1.2x fewer** |
| Table-heavy | **47.3K** | 222.5K | 156.9K | **4.7x fewer** |
| Inline-heavy | **169.0K** | 825.4K | 828.7K | **4.9x fewer** |

### Memory (lower is better)

| Input | ours | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: |
| Spec | 16.4 MB | 26.1 MB | **5.4 MB** | **1.6x less** |
| README | 2.0 MB | 3.2 MB | **1.0 MB** | **1.6x less** |
| GitHub Top 10 | **9.5 MB** | 13.8 MB | 131.4 MB | **1.5x less** |
| Code-heavy | **4.2 MB** | 34.1 MB | 11.7 MB | **8.1x less** |
| Table-heavy | 15.1 MB | **10.9 MB** | 3.4 MB | 1.4x more |
| Inline-heavy | 53.7 MB | **44.6 MB** | 18.1 MB | 1.2x more |

## Parse-Only

Our streaming parser vs batch parsers. The `ours-reuse` variant
reuses the parser across iterations to isolate parse cost from
allocation cost (~8% faster).

### Speed (lower is better)

| Input | ours | ours-reuse | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Spec (~120KB) | 5.7ms | 5.4ms | **1.7ms** | 2.2ms | 385ms | 3.2x slower |
| README (~10KB) | 635us | 639us | **273us** | 375us | 932us | 2.3x slower |
| GitHub Top 10 (~130KB) | 5.2ms | 6.4ms | 3.0ms | **1.7ms** | 3.6ms | 3.1x slower |

### Allocations (lower is better)

| Input | ours | ours-reuse | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Spec | 23.0K | 23.0K | **13.8K** | 22.9K | 25.9K | 1.7x more |
| README | 3.5K | 3.5K | **1.6K** | 3.0K | 3.6K | 2.2x more |
| GitHub Top 10 | 18.4K | 18.4K | 14.9K | **8.0K** | 8.3K | 2.3x more |

### Memory (lower is better)

| Input | ours | ours-reuse | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Spec | 14.7 MB | 14.6 MB | 2.0 MB | 4.0 MB | **1.7 MB** | 7.4x more |
| README | 1.6 MB | 1.6 MB | **238 KB** | 556 KB | 243 KB | 7.0x more |
| GitHub Top 10 | 7.4 MB | 7.3 MB | 2.3 MB | 1.6 MB | **1.1 MB** | 3.2x more |

**Why we use more memory**: Our parser allocates `Event` structs into a
flat slice (the streaming output). Batch parsers build compact AST trees
with pointer-linked nodes. This is the fundamental trade-off for
streaming: we produce a consumable event stream immediately, while batch
parsers require the full document before producing output.

## Syntax Highlighting: Go Fast Path vs Chroma

Our built-in Go highlighter uses stdlib AST tokenization instead of
Chroma's regex-based lexer. Benchmark on 100 Go code blocks:

| Highlighter | Speed | Allocations | Memory | vs Chroma |
| --- | ---: | ---: | ---: | ---: |
| **Go fast path** | **1.7ms** | **16.7K** | **2.2 MB** | -- |
| Chroma | 21.8ms | 112.8K | 7.8 MB | **13x slower, 6.7x more allocs** |

## Streaming (ours only)

No other Go library supports streaming. Chunk size sensitivity on the
Spec input (~120KB):

| Chunk size | Speed | Allocs | vs whole-doc |
| --- | ---: | ---: | ---: |
| 1 byte | 10.3ms | 69.0K | 1.4x slower |
| 16 bytes | 9.2ms | 65.5K | 1.2x slower |
| 64 bytes | 8.1ms | 61.0K | 1.1x slower |
| 256 bytes | 10.7ms | 58.3K | 1.4x slower |
| 1 KB | 9.6ms | 57.3K | 1.3x slower |
| 4 KB | **6.5ms** | 57.0K | **fastest** |
| Whole doc | 7.6ms | 56.8K | baseline |

Streaming at 4KB chunks is **faster** than whole-document parsing
because intermediate allocations are smaller. Even byte-at-a-time
streaming is only 1.4x slower.

## Reproduction

```bash
task bench:render      # terminal render comparison (Markdown tables)
task bench:parse       # parse comparison (Markdown tables)
task bench:compliance  # spec compliance against all parsers
task bench:chunks      # chunk size sensitivity
task bench             # all raw benchmark output
```

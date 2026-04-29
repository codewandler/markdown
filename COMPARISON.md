# Comparison with Other Go Markdown Libraries

Benchmarks run on Intel(R) Core(TM) i9-10900K CPU @ 3.70GHz, go1.26.1-X:nodwarf5, Linux. Git SHA: `3d48425`.

See [docs/competitors.md](docs/competitors.md) for detailed library profiles.

## Feature Matrix

| Feature | ours | goldmark | glamour | blackfriday | gomarkdown | go-term-markdown |
| --- | :---: | :---: | :---: | :---: | :---: | :---: |
| Parser | custom streaming | goldmark | goldmark | blackfriday | gomarkdown | blackfriday v1 |
| Terminal render | **✅** | ❌ | **✅** | ❌ | ❌ | **✅** |
| **Streaming** | **✅** | ❌ | ❌ | ❌ | ❌ | ❌ |
| CommonMark 0.31.2 | 99.4% | 99.2% | - | 37.4% | 40.3% | - |
| GFM 0.29 | 95.4% | 97.5% | - | 36.8% | 39.1% | - |
| Syntax highlighting | Go fast path + Chroma | — | Chroma | — | — | Chroma v1 |
| Clickable hyperlinks | OSC 8 | — | ❌ | — | — | ❌ |
| Word wrapping | auto-detect | — | fixed width | — | — | fixed width |
| TTY detection | auto | — | ❌ | — | — | ❌ |
| Direct dependencies | **2** | **0** | **28** | **0** | **0** | **20** |
| ⭐ Stars | — | 4.7K | 3.4K | 5.6K | 1.7K | 291 |
| Go source lines | 5.9K | 13.7K | 4.3K | 5.3K | 9.1K | 1.8K |
| Test coverage | 75.4% | 17.4% | 49.7% | 79.5% | — | — |

## Spec Compliance

Measured by running each parser against the official spec test suites
and comparing HTML output.

| Spec | ours | goldmark | blackfriday | gomarkdown |
| --- | ---: | ---: | ---: | ---: |
| CommonMark 0.31.2 | 648/652 (99.4%) | 647/652 (99.2%) | 244/652 (37.4%) | 263/652 (40.3%) |
| GFM 0.29 | 641/672 (95.4%) | 655/672 (97.5%) | 247/672 (36.8%) | 263/672 (39.1%) |

Note: All parsers are measured by comparing HTML output against the
spec expected HTML. Our HTML renderer is new and does not yet cover
all edge cases — our event-level (structural) compliance is 96.2%
CommonMark and 100% GFM. The HTML compliance will converge as the
renderer matures.

## Terminal Rendering (parse + render to ANSI string)

### Speed (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| README | 1.1ms | **1.0ms** | 6.2ms | 3.9ms | **5.7x faster** |
| TableHeavy | 5.8ms | **4.5ms** | 28.3ms | 6.8ms | **4.9x faster** |
| CodeHeavy | **2.4ms** | 2.7ms | 9.7ms | 52.3ms | **4.0x faster** |
| GitHubTop10 | 32.2ms | **31.7ms** | 38.4ms | 2.88s | **1.2x faster** |
| InlineHeavy | 30.5ms | **27.2ms** | 88.3ms | 38.6ms | **2.9x faster** |
| Spec | 8.0ms | **6.3ms** | 41.0ms | 410.5ms | **5.1x faster** |

### Allocations (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| README | **9.6K** | 9.7K | 49.4K | 37.8K | **5.1x fewer** |
| TableHeavy | **47.4K** | 47.5K | 262.6K | 156.9K | **5.5x fewer** |
| CodeHeavy | **33.1K** | 33.1K | 39.0K | 288.7K | **1.2x fewer** |
| GitHubTop10 | **40.0K** | 40.2K | 367.3K | 1.4M | **9.2x fewer** |
| InlineHeavy | **169.1K** | 169.5K | 811.4K | 828.7K | **4.8x fewer** |
| Spec | **56.9K** | 57.0K | 304.6K | 183.4K | **5.4x fewer** |

### Memory (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| README | 2.0 MB | 1.5 MB | 5.1 MB | **1.0 MB** | **2.6x less** |
| TableHeavy | 15.1 MB | 11.2 MB | 16.1 MB | **3.6 MB** | **1.1x less** |
| CodeHeavy | 4.0 MB | **3.2 MB** | 34.4 MB | 11.8 MB | **8.7x less** |
| GitHubTop10 | 9.5 MB | **7.0 MB** | 19.2 MB | 119.2 MB | **2.0x less** |
| InlineHeavy | 53.7 MB | 45.8 MB | 46.5 MB | **18.7 MB** | 1.2x more |
| Spec | 16.3 MB | 11.7 MB | 36.9 MB | **5.5 MB** | **2.3x less** |

## Parse-Only

### Speed (lower is better)

| Input | ours | ours-reuse | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 5.0ms | 4.4ms | 2.6ms | **1.1ms** | 3.9ms | 2.0x slower |
| README | 800.1us | 653.2us | **216.4us** | 313.7us | 949.8us | 3.7x slower |
| Spec | 6.3ms | 6.0ms | **1.7ms** | 2.2ms | 384.0ms | 3.7x slower |

### Allocations (lower is better)

| Input | ours | ours-reuse | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 18.6K | 18.4K | 13.1K | **8.0K** | 8.3K | 1.4x more |
| README | 3.5K | 3.5K | **1.4K** | 3.0K | 3.6K | 2.5x more |
| Spec | 23.1K | 23.0K | **11.4K** | 22.9K | 25.9K | 2.0x more |

### Memory (lower is better)

| Input | ours | ours-reuse | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 9.8 MB | 7.6 MB | 1.9 MB | 1.9 MB | **1.4 MB** | 5.2x more |
| README | 2.1 MB | 1.6 MB | **208.6 KB** | 580.7 KB | 267.0 KB | 10.2x more |
| Spec | 18.3 MB | 14.7 MB | **1.7 MB** | 4.1 MB | 1.8 MB | 11.0x more |

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
| **Go fast path** | **1.4ms** | **16.8K** | **2.4 MB** | -- |
| Chroma | 21.7ms | 113.7K | 7.7 MB | **16x slower, 6.8x more allocs** |

## Streaming (ours only)

No other Go library supports streaming. Chunk size sensitivity on the
Spec input (~120KB):

| Chunk size | Speed | Allocs | vs whole-doc |
| --- | ---: | ---: | ---: |
| 1 byte | 7.8ms | 57.1K | **fastest** |
| 16 bytes | 7.6ms | 57.1K | **fastest** |
| 64 bytes | 7.8ms | 57.1K | 1.0x slower |
| 256 bytes | 7.6ms | 57.1K | **fastest** |
| 1 KB | 8.0ms | 57.1K | 1.0x slower |
| 4 KB | 8.0ms | 57.1K | 1.0x slower |
| Whole doc | 7.8ms | 57.1K | baseline |

Streaming at 4KB chunks is **faster** than whole-document parsing
because intermediate allocations are smaller. Even byte-at-a-time
streaming is only ~1.4x slower.

## Reproduction

```bash
task competition:metadata    # Stage 1: discover metadata
task competition:compliance  # Stage 2: spec compliance
task competition:bench       # Stage 3: benchmarks
task competition:report      # Stage 4: generate this report
task competition:full        # all stages in sequence
```

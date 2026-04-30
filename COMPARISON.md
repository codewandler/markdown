# Comparison with Other Go Markdown Libraries

Benchmarks run on Intel(R) Core(TM) i9-10900K CPU @ 3.70GHz, go1.26.1-X:nodwarf5, Linux. Git SHA: `b01ccad`.

See [docs/competitors.md](docs/competitors.md) for detailed library profiles.

## Feature Matrix

| Feature | ours | goldmark | glamour | blackfriday | gomarkdown | go-term-markdown |
| --- | :---: | :---: | :---: | :---: | :---: | :---: |
| Parser | custom streaming | goldmark | goldmark | blackfriday | gomarkdown | blackfriday v1 |
| Terminal render | **✅** | ❌ | **✅** | ❌ | ❌ | **✅** |
| **Streaming** | **✅** | ❌ | ❌ | ❌ | ❌ | ❌ |
| CommonMark 0.31.2 | 100.0% | 99.2% | - | 37.4% | 40.3% | - |
| GFM 0.29 | 98.7% | 97.5% | - | 36.8% | 39.1% | - |
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
| CommonMark 0.31.2 | 652/652 (100.0%) | 647/652 (99.2%) | 244/652 (37.4%) | 263/652 (40.3%) |
| GFM 0.29 | 663/672 (98.7%) | 655/672 (97.5%) | 247/672 (36.8%) | 263/672 (39.1%) |

Note: All parsers are measured by comparing HTML output against the
spec expected HTML. GFM compliance uses per-example extension
dispatch matching the official spec_tests.py behavior. The 9
remaining GFM failures are emphasis Rule 13 cases where CommonMark
0.31.2 and GFM 0.29 disagree; we follow the newer CommonMark spec.

## Terminal Rendering (parse + render to ANSI string)

### Speed (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| CodeHeavy | 3.2ms | **3.1ms** | 9.1ms | 52.1ms | **2.8x faster** |
| GitHubTop10 | 32.9ms | **32.1ms** | 37.2ms | 2.77s | **1.1x faster** |
| InlineHeavy | 39.2ms | **30.8ms** | 88.2ms | 38.3ms | **2.2x faster** |
| README | 1.2ms | **1.1ms** | 6.2ms | 3.9ms | **5.3x faster** |
| Spec | 9.2ms | **7.2ms** | 40.9ms | 405.0ms | **4.4x faster** |
| TableHeavy | 6.8ms | **5.0ms** | 28.6ms | 6.7ms | **4.2x faster** |

### Allocations (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| CodeHeavy | **33.1K** | 33.1K | 39.0K | 288.7K | **1.2x fewer** |
| GitHubTop10 | **40.2K** | 40.5K | 367.3K | 1.4M | **9.1x fewer** |
| InlineHeavy | **171.0K** | 171.6K | 811.4K | 828.7K | **4.7x fewer** |
| README | **9.3K** | 9.3K | 49.4K | 37.8K | **5.3x fewer** |
| Spec | **56.1K** | 56.2K | 304.7K | 183.4K | **5.4x fewer** |
| TableHeavy | **47.4K** | 47.4K | 262.6K | 156.9K | **5.5x fewer** |

### Memory (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| CodeHeavy | 4.4 MB | **3.5 MB** | 34.4 MB | 11.8 MB | **7.8x less** |
| GitHubTop10 | 12.6 MB | **10.0 MB** | 19.1 MB | 119.1 MB | **1.5x less** |
| InlineHeavy | 84.1 MB | 71.2 MB | 46.5 MB | **18.7 MB** | 1.8x more |
| README | 2.6 MB | 2.5 MB | 5.1 MB | **1.0 MB** | **2.0x less** |
| Spec | 21.8 MB | 16.6 MB | 36.6 MB | **5.5 MB** | **1.7x less** |
| TableHeavy | 19.5 MB | 14.5 MB | 16.2 MB | **3.6 MB** | 1.2x more |

## Parse-Only

### Speed (lower is better)

| Input | ours | ours-4k | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 5.8ms | 6.0ms | 2.5ms | **1.2ms** | 3.9ms | 2.4x slower |
| README | 908.3us | 1.1ms | **240.3us** | 379.4us | 931.5us | 3.8x slower |
| Spec | 7.8ms | 8.6ms | **1.7ms** | 2.3ms | 392.7ms | 4.6x slower |

### Allocations (lower is better)

| Input | ours | ours-4k | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 19.7K | 19.8K | 13.1K | **8.0K** | 8.3K | 1.5x more |
| README | 3.2K | 3.2K | **1.4K** | 3.0K | 3.6K | 2.4x more |
| Spec | 22.3K | 22.4K | **11.4K** | 22.9K | 25.9K | 2.0x more |

### Memory (lower is better)

| Input | ours | ours-4k | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 14.6 MB | 14.9 MB | 1.9 MB | 1.9 MB | **1.4 MB** | 7.8x more |
| README | 2.8 MB | 3.4 MB | **208.6 KB** | 579.8 KB | 266.1 KB | 14.0x more |
| Spec | 25.8 MB | 27.6 MB | **1.7 MB** | 4.1 MB | 1.8 MB | 15.5x more |

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
| **Go fast path** | **1.3ms** | **16.7K** | **2.6 MB** | -- |
| Chroma | 20.3ms | 112.6K | 7.8 MB | **16x slower, 6.7x more allocs** |

## Streaming (ours only)

No other Go library supports streaming. Chunk size sensitivity on the
Spec input (~120KB):

| Chunk size | Speed | Allocs | vs whole-doc |
| --- | ---: | ---: | ---: |
| 1 byte | 10.0ms | 56.1K | 1.1x slower |
| 16 bytes | 9.3ms | 56.1K | 1.0x slower |
| 64 bytes | 9.4ms | 56.1K | 1.0x slower |
| 256 bytes | 9.5ms | 56.1K | 1.1x slower |
| 1 KB | 9.9ms | 56.1K | 1.1x slower |
| 4 KB | 9.8ms | 56.1K | 1.1x slower |
| Whole doc | 8.9ms | 56.1K | baseline |

Streaming adds minimal overhead: 4KB chunks are only ~1.1x slower
than whole-document parsing, and even byte-at-a-time streaming is
within ~1.1x — with identical allocation counts.

## Reproduction

```bash
task competition:metadata    # Stage 1: discover metadata
task competition:compliance  # Stage 2: spec compliance
task competition:bench       # Stage 3: benchmarks
task competition:report      # Stage 4: generate this report
task competition:full        # all stages in sequence
```

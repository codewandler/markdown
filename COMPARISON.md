# Comparison with Other Go Markdown Libraries

Benchmarks run on Intel(R) Core(TM) i9-10900K CPU @ 3.70GHz, go1.26.1, Linux. Git SHA: `6cdc1f0`.

See [docs/competitors.md](docs/competitors.md) for detailed library profiles.

## Feature Matrix

| Feature | ours | goldmark | glamour | blackfriday | gomarkdown | go-term-markdown |
| --- | :---: | :---: | :---: | :---: | :---: | :---: |
| Parser | custom streaming | goldmark | goldmark | blackfriday | gomarkdown | blackfriday v1 |
| Terminal render | **✅** | ❌ | **✅** | ❌ | ❌ | **✅** |
| **Streaming** | **✅** | ❌ | ❌ | ❌ | ❌ | ❌ |
| CommonMark 0.31.2 | 100.0% | 99.2% | - | 37.4% | 40.3% | - |
| GFM 0.29 | 97.1% | 94.8% | - | 34.6% | 36.8% | - |
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
| GFM spec.txt | 663/672 (98.7%) | 655/672 (97.5%) | 247/672 (36.8%) | 263/672 (39.1%) |
| GFM extensions.txt | 22/30 | 21/30 | 1/30 | 1/30 |
| GFM regression.txt | 22/26 | 14/26 | 4/26 | 4/26 |
| **GFM total** | **707/728 (97.1%)** | **690/728 (94.8%)** | **252/728 (34.6%)** | **268/728 (36.8%)** |

GFM compliance uses per-example extension dispatch matching the official
spec_tests.py behavior. See [docs/compliance.md](docs/compliance.md) for
details on remaining gaps.

## Terminal Rendering (parse + render to ANSI string)

### Speed (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| CodeHeavy | 3.1ms | **3.0ms** | 9.1ms | 51.2ms | **2.9x faster** |
| GitHubTop10 | **30.6ms** | 30.9ms | 36.6ms | 3.21s | **1.2x faster** |
| InlineHeavy | **30.8ms** | 31.1ms | 87.1ms | 38.4ms | **2.8x faster** |
| README | **807.9us** | 891.1us | 6.3ms | 3.8ms | **7.7x faster** |
| Spec | **4.6ms** | 5.0ms | 41.3ms | 405.7ms | **9.0x faster** |
| TableHeavy | 3.7ms | **3.2ms** | 28.4ms | 6.8ms | **7.7x faster** |

### Allocations (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| CodeHeavy | **33.0K** | 33.0K | 39.0K | 288.7K | **1.2x fewer** |
| GitHubTop10 | **28.9K** | 29.0K | 367.3K | 1.4M | **12.7x fewer** |
| InlineHeavy | **129.0K** | 129.2K | 811.4K | 828.7K | **6.3x fewer** |
| README | 7.0K | **7.0K** | 49.4K | 37.8K | **7.0x fewer** |
| Spec | **43.6K** | 43.7K | 304.6K | 183.4K | **7.0x fewer** |
| TableHeavy | **37.3K** | 37.3K | 262.6K | 156.9K | **7.0x fewer** |

### Memory (lower is better)

| Input | ours | ours-4k | glamour | go-term-md | vs glamour |
| --- | ---: | ---: | ---: | ---: | ---: |
| CodeHeavy | 3.2 MB | **3.0 MB** | 34.4 MB | 11.8 MB | **10.8x less** |
| GitHubTop10 | **3.9 MB** | 4.5 MB | 19.3 MB | 119.2 MB | **4.9x less** |
| InlineHeavy | 25.4 MB | **18.6 MB** | 46.5 MB | 18.7 MB | **1.8x less** |
| README | **762.0 KB** | 924.8 KB | 5.0 MB | 1.0 MB | **6.7x less** |
| Spec | 4.2 MB | **4.0 MB** | 36.2 MB | 5.5 MB | **8.6x less** |
| TableHeavy | 9.2 MB | 7.3 MB | 15.9 MB | **3.6 MB** | **1.7x less** |

## Parse-Only

### Speed (lower is better)

| Input | ours | ours-4k | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 4.5ms | 5.0ms | 2.8ms | **1.0ms** | 3.9ms | 1.6x slower |
| README | 435.6us | 527.7us | **263.9us** | 335.4us | 946.9us | 1.7x slower |
| Spec | 3.7ms | 4.1ms | **1.8ms** | 2.3ms | 391.0ms | 2.1x slower |

### Allocations (lower is better)

| Input | ours | ours-4k | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 8.3K | 8.4K | 13.1K | **8.0K** | 8.3K | **1.6x fewer** |
| README | **942** | 947 | 1.4K | 3.0K | 3.6K | **1.5x fewer** |
| Spec | **9.8K** | 9.9K | 11.4K | 22.9K | 25.9K | **1.2x fewer** |

### Memory (lower is better)

| Input | ours | ours-4k | goldmark | blackfriday | gomarkdown | vs goldmark |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| GitHubTop10 | 6.3 MB | 6.9 MB | 1.9 MB | 1.9 MB | **1.4 MB** | 3.3x more |
| README | 992.9 KB | 1.3 MB | **208.6 KB** | 579.9 KB | 266.9 KB | 4.8x more |
| Spec | 8.1 MB | 9.8 MB | **1.7 MB** | 4.1 MB | 1.8 MB | 4.9x more |

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
| **Go fast path** | **1.1ms** | **16.7K** | **1.5 MB** | -- |
| Chroma | 20.0ms | 112.6K | 6.7 MB | **19x slower, 6.7x more allocs** |

## Streaming (ours only)

No other Go library supports streaming. Chunk size sensitivity on the
Spec input (~120KB):

| Chunk size | Speed | Allocs | vs whole-doc |
| --- | ---: | ---: | ---: |
| 1 byte | 4.4ms | 43.6K | **fastest** |
| 16 bytes | 4.2ms | 43.6K | **fastest** |
| 64 bytes | 4.3ms | 43.6K | **fastest** |
| 256 bytes | 4.2ms | 43.6K | **fastest** |
| 1 KB | 4.4ms | 43.6K | **fastest** |
| 4 KB | 4.4ms | 43.6K | **fastest** |
| Whole doc | 4.5ms | 43.6K | baseline |

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

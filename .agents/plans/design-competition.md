# Design: `competition/` Package

Status: **draft v3**
Created: 2026-04-29

## Goal

A pipeline-driven competition framework where candidates are declared
minimally (repo URL + named variant factories) and everything else is
discovered, measured, and reported automatically. One pipeline run
produces a single `results.json` containing all data.

## Principles

1. **Minimal declaration** — a candidate is a repo URL + variant
   factories. Everything else is discovered.
2. **Variants** — same repo can have multiple named configurations
   (buffer sizes, highlighters, parser reuse). Each variant gets its
   own column in results.
3. **Pipeline stages** — each stage produces typed output that feeds
   the next. Stages can run independently or chained.
4. **Single output** — one `results.json` per run with everything:
   metadata, compliance, benchmarks, system info.
5. **Deterministic report** — `compgen` reads `results.json` and
   produces the same COMPARISON.md every time.

## Pipeline

```
Candidates (source: repo URL + variant factories)
    |
    v
Stage 1: Discover Metadata
    |  gh CLI + git clone + go commands
    |  -> MetadataResult per candidate (repo-level)
    v
Stage 2: Compliance Testing
    |  CommonMark + GFM spec suites via RenderHTML adapter
    |  -> ComplianceResult per variant (that has RenderHTML)
    v
Stage 3: Benchmarks
    |  parse, render, streaming via adapters
    |  -> BenchmarkResult per variant per input
    v
Stage 4: Generate Report
    |  merge results.json -> COMPARISON.md
    v
Output: results.json + COMPARISON.md
```

## Package Structure

```
competition/
├── go.mod                      # separate module
├── candidate.go                # Candidate, Variant, Adapters types
├── candidates.go               # declarations + adapter factories
├── metadata.go                 # Stage 1: discover metadata
├── compliance.go               # Stage 2: compliance testing
├── results.go                  # RunResult, all result types
├── benchmarks/
│   ├── bench_test.go           # Stage 3: benchmark harness
│   ├── inputs.go               # input generators
│   └── testdata/
│       └── github-top10/       # real-world READMEs
├── cmd/
│   └── compgen/
│       └── main.go             # Stage 4: generate COMPARISON.md
└── README.md
```

## Types

### Candidate (source — what we declare)

```go
// Candidate is the minimal declaration for a competitor.
// Everything beyond this is discovered or measured by the pipeline.
type Candidate struct {
    // Repo is the GitHub repository URL. Primary key for metadata
    // discovery. Multiple variants can share a repo.
    Repo string

    // Variants are named configurations of this candidate.
    // Each variant gets its own column in benchmark results.
    // A candidate must have at least one variant.
    Variants []Variant
}

// Variant is a named configuration with specific adapter functions.
// Multiple variants for the same repo test different configurations
// (buffer sizes, highlighters, parser reuse, etc.).
type Variant struct {
    // Name is the display name in results.
    // Examples: "ours", "ours-reuse", "ours-4k", "goldmark"
    Name string

    // Description explains what this variant tests.
    // Example: "4KB read buffer (streaming sweet spot)"
    Description string

    // Adapters exercise this variant. nil = doesn't support.
    Adapters Adapters
}

type Adapters struct {
    // ParseFunc reads Markdown from r and parses it.
    // Returns the number of output items (events, nodes, etc.).
    ParseFunc      func(r io.Reader) (int, error)

    // RenderTerminal reads Markdown from r and renders styled
    // terminal output to w.
    RenderTerminal func(r io.Reader, w io.Writer) error

    // RenderHTML reads Markdown from r and writes HTML to w.
    // Used for compliance testing against spec expected output.
    RenderHTML     func(r io.Reader, w io.Writer) error
}
```

### Variant examples for our library

```go
{
    Repo: "https://github.com/codewandler/markdown",
    Variants: []Variant{
        {
            Name:        "ours",
            Description: "default configuration",
            Adapters:    defaultAdapters(),
        },
        {
            Name:        "ours-reuse",
            Description: "parser reused across iterations",
            Adapters:    reusableAdapters(),
        },
        {
            Name:        "ours-4k",
            Description: "4KB streaming chunks",
            Adapters:    chunkedAdapters(4096),
        },
        {
            Name:        "ours-1byte",
            Description: "1-byte chunks (worst case)",
            Adapters:    chunkedAdapters(1),
        },
        {
            Name:        "ours-go-highlight",
            Description: "Go stdlib fast path only",
            Adapters:    withHighlighter(terminal.NewDefaultHighlighter()),
        },
        {
            Name:        "ours-chroma",
            Description: "Chroma for all languages",
            Adapters:    withHighlighter(terminal.NewHybridHighlighter()),
        },
    },
}
```

### Results (single output file)

```go
// RunResult is the complete output of one pipeline run.
// Serialized as results.json.
type RunResult struct {
    RunAt      time.Time          `json:"run_at"`
    System     SystemInfo         `json:"system"`
    Candidates []CandidateResult  `json:"candidates"`
}

type SystemInfo struct {
    OS        string `json:"os"`
    Arch      string `json:"arch"`
    CPU       string `json:"cpu"`
    GoVersion string `json:"go_version"`
    Hostname  string `json:"hostname"`
}

// CandidateResult holds all discovered and measured data for one repo.
type CandidateResult struct {
    Repo       string                          `json:"repo"`
    Metadata   MetadataResult                  `json:"metadata"`
    Variants   map[string]VariantResult         `json:"variants"`
}

// VariantResult holds measured data for one named variant.
type VariantResult struct {
    Description string                         `json:"description"`
    Compliance  *ComplianceResult              `json:"compliance,omitempty"`
    Benchmarks  map[string]BenchmarkResult     `json:"benchmarks"`
    // key = input name ("Spec", "README", etc.)
}
```

### Stage 1: Metadata Discovery

```go
type MetadataResult struct {
    // Identity (from gh CLI / GitHub API)
    Name        string    `json:"name"`
    Owner       string    `json:"owner"`
    Description string    `json:"description"`
    License     string    `json:"license"`
    Stars       int       `json:"stars"`
    OpenIssues  int       `json:"open_issues"`
    Forks       int       `json:"forks"`
    LastCommit  time.Time `json:"last_commit"`

    // Code metrics (from clone + inspection)
    GoFiles     int       `json:"go_files"`
    GoLines     int       `json:"go_lines"`
    TestFiles   int       `json:"test_files"`
    TestLines   int       `json:"test_lines"`

    // Dependencies (from go.mod)
    DirectDeps     int      `json:"direct_deps"`
    TransitiveDeps int      `json:"transitive_deps"`
    DepNames       []string `json:"dep_names"`

    // Test coverage (from go test -cover)
    TestCoverage   float64  `json:"test_coverage"` // percentage

    DiscoveredAt time.Time `json:"discovered_at"`
}
```

**How it works (all automated):**

```bash
# GitHub API via gh CLI
gh repo view $repo --json name,owner,description,licenseInfo,
    stargazerCount,issues,forkCount,pushedAt

# Clone + code metrics
git clone --depth=1 $repo /tmp/comp/$name
find . -name '*.go' ! -name '*_test.go' | xargs wc -l
find . -name '*_test.go' | xargs wc -l

# Dependencies
go list -m all | wc -l          # transitive
go list -m -f '{{.Path}}' all   # names

# Test coverage
go test -cover ./... 2>&1 | grep -oP 'coverage: [\d.]+%'
```

### Stage 2: Compliance Testing

```go
type ComplianceResult struct {
    CommonMark SpecResult `json:"commonmark"`
    GFM        SpecResult `json:"gfm"`
}

type SpecResult struct {
    Version    string             `json:"version"`
    Pass       int                `json:"pass"`
    Total      int                `json:"total"`
    Percentage float64            `json:"percentage"`
    Sections   map[string]Section `json:"sections,omitempty"`
}

type Section struct {
    Pass  int `json:"pass"`
    Total int `json:"total"`
}
```

Only variants with `RenderHTML != nil` are compliance-tested.
Our event-level compliance is reported separately (from our own
test suite, not through this pipeline).

### Stage 3: Benchmarks

```go
type BenchmarkResult struct {
    Category string  `json:"category"` // "parse", "render"
    NsOp     float64 `json:"ns_op"`
    BOp      int64   `json:"b_op"`
    AllocsOp int64   `json:"allocs_op"`
}
```

The harness iterates all variants, skipping those with nil adapters:

```go
func BenchmarkRender(b *testing.B) {
    for _, c := range competition.All {
        for _, v := range c.Variants {
            if v.Adapters.RenderTerminal == nil {
                continue
            }
            for _, input := range AllInputs() {
                name := v.Name + "/" + input.Name
                b.Run(name, func(b *testing.B) {
                    src := []byte(input.Generate())
                    b.SetBytes(int64(len(src)))
                    for b.Loop() {
                        r := bytes.NewReader(src)
                        var w bytes.Buffer
                        v.Adapters.RenderTerminal(r, &w)
                    }
                })
            }
        }
    }
}
```

## Candidate Declarations

```go
// candidates.go

var All = []Candidate{
    {
        Repo: "https://github.com/codewandler/markdown",
        Variants: []Variant{
            {Name: "ours", Description: "default", Adapters: defaultAdapters()},
            {Name: "ours-reuse", Description: "parser reuse", Adapters: reusableAdapters()},
            {Name: "ours-4k", Description: "4KB chunks", Adapters: chunkedAdapters(4096)},
        },
    },
    {
        Repo: "https://github.com/yuin/goldmark",
        Variants: []Variant{
            {Name: "goldmark", Adapters: Adapters{
                ParseFunc:  goldmarkParse,
                RenderHTML: goldmarkRenderHTML,
            }},
        },
    },
    {
        Repo: "https://github.com/charmbracelet/glamour",
        Variants: []Variant{
            {Name: "glamour", Adapters: Adapters{
                RenderTerminal: glamourRender,
            }},
        },
    },
    {
        Repo: "https://github.com/russross/blackfriday",
        Variants: []Variant{
            {Name: "blackfriday", Adapters: Adapters{
                ParseFunc:  blackfridayParse,
                RenderHTML: blackfridayRenderHTML,
            }},
        },
    },
    {
        Repo: "https://github.com/gomarkdown/markdown",
        Variants: []Variant{
            {Name: "gomarkdown", Adapters: Adapters{
                ParseFunc:  gomarkdownParse,
                RenderHTML: gomarkdownRenderHTML,
            }},
        },
    },
    {
        Repo: "https://github.com/MichaelMure/go-term-markdown",
        Variants: []Variant{
            {Name: "go-term-md", Adapters: Adapters{
                RenderTerminal: goTermMarkdownRender,
            }},
        },
    },
}
```

## CLI / Taskfile

```yaml
competition:metadata:
    desc: Discover metadata for all candidates
    dir: competition
    cmd: go run . metadata -out results.json

competition:compliance:
    desc: Run compliance tests
    dir: competition
    cmd: go run . compliance -out results.json

competition:bench:
    desc: Run benchmarks
    dir: competition
    cmd: go test -bench=. -benchmem ./benchmarks

competition:report:
    desc: Generate COMPARISON.md from results.json
    dir: competition
    cmd: go run ./cmd/compgen -in results.json -out ../COMPARISON.md

competition:full:
    desc: Full pipeline run
    cmds:
        - task: competition:metadata
        - task: competition:compliance
        - task: competition:bench
        - task: competition:report
```

## Example results.json

```json
{
    "run_at": "2026-04-29T01:50:00Z",
    "system": {
        "os": "linux",
        "arch": "amd64",
        "cpu": "Intel Core i9-10900K @ 3.70GHz",
        "go_version": "go1.26.1"
    },
    "candidates": [
        {
            "repo": "https://github.com/codewandler/markdown",
            "metadata": {
                "name": "markdown",
                "owner": "codewandler",
                "license": "MIT",
                "stars": 0,
                "go_files": 12,
                "go_lines": 5200,
                "test_files": 15,
                "test_lines": 8400,
                "direct_deps": 2,
                "transitive_deps": 2,
                "test_coverage": 82.3
            },
            "variants": {
                "ours": {
                    "description": "default",
                    "benchmarks": {
                        "Spec": {"category": "render", "ns_op": 8300000, "b_op": 16400000, "allocs_op": 56800},
                        "README": {"category": "render", "ns_op": 1100000, "b_op": 2000000, "allocs_op": 9600}
                    }
                },
                "ours-reuse": {
                    "description": "parser reuse",
                    "benchmarks": {
                        "Spec": {"category": "parse", "ns_op": 5400000, "b_op": 14600000, "allocs_op": 23000}
                    }
                }
            }
        },
        {
            "repo": "https://github.com/yuin/goldmark",
            "metadata": {
                "name": "goldmark",
                "owner": "yuin",
                "license": "MIT",
                "stars": 3800,
                "direct_deps": 0,
                "test_coverage": 91.5
            },
            "variants": {
                "goldmark": {
                    "compliance": {
                        "commonmark": {"version": "0.31.2", "pass": 646, "total": 652, "percentage": 99.1},
                        "gfm": {"version": "0.29", "pass": 654, "total": 672, "percentage": 97.3}
                    },
                    "benchmarks": {
                        "Spec": {"category": "parse", "ns_op": 1700000, "b_op": 2000000, "allocs_op": 13800}
                    }
                }
            }
        }
    ]
}
```

## Migration from `benchmarks/`

1. Create `competition/` with new structure
2. Move `inputs.go`, `testdata/github-top10/` into `competition/benchmarks/`
3. Define candidates with variant factories in `candidates.go`
4. Implement metadata discovery (Stage 1)
5. Rewrite compliance tests to use adapters (Stage 2)
6. Rewrite benchmarks to iterate variants (Stage 3)
7. Build `compgen` to read `results.json` (Stage 4)
8. Delete old `benchmarks/` directory
9. Update Taskfile, README, roadmap

## Open Questions

1. **Metadata caching** — Stage 1 is slow (clones repos). Cache
   `results.json` and only re-run metadata on demand?
2. **Our compliance** — we can't use RenderHTML. Report event-level
   compliance from our own test suite as a special case in results.json?
3. **Panic recovery** — wrap adapter calls in `recover()` for
   pathological inputs? Record "panic" as result.
4. **Historical tracking** — keep old results.json files to show
   trends over time?

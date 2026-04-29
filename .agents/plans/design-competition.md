# Design: `competition/` Package

Status: **draft v4 — questions resolved**
Created: 2026-04-29

## Goal

A pipeline-driven competition framework where candidates are declared
with repo URL, qualitative features, and named variant factories.
Quantitative data (metadata, compliance, benchmarks) is discovered
and measured automatically. Each pipeline run produces a timestamped
results snapshot.

## Principles

1. **Explicit declaration** — a candidate is a repo URL + qualitative
   features + variant factories. Features (streaming, TTY detection,
   highlighting, etc.) are declared because they require human
   knowledge. Everything quantitative is discovered.
2. **Variants** — same repo can have multiple named configurations
   (buffer sizes, highlighters, parser reuse). Each variant gets its
   own column in results.
3. **Pipeline stages** — each stage produces typed output that feeds
   the next. Stages can run independently or chained.
4. **Timestamped output** — each run writes
   `results/results-{timestamp}.json`. Report generation picks the
   most recent by default. Old snapshots accumulate in version control.
5. **Deterministic report** — `compgen` reads a results file and
   produces the same COMPARISON.md every time. `--latest` (default)
   picks the newest snapshot; `--results path` uses a specific one.

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
├── results/                    # timestamped result snapshots
│   └── results-2026-04-29T01:50.json
├── cmd/
│   └── compgen/
│       └── main.go             # Stage 4: generate COMPARISON.md
└── README.md
```

## Types

### Candidate (source — what we declare)

```go
// Candidate is the declaration for a competitor.
// Metadata (stars, deps, coverage) is discovered by the pipeline.
// Features are declared here because they are qualitative and
// cannot be measured automatically.
type Candidate struct {
    // Repo is the GitHub repository URL. Primary key for metadata
    // discovery. Multiple variants can share a repo.
    Repo string

    // Features describes qualitative capabilities that cannot be
    // discovered or measured by the pipeline. These populate the
    // feature matrix in the report.
    Features Features

    // Variants are named configurations of this candidate.
    // Each variant gets its own column in benchmark results.
    // A candidate must have at least one variant.
    Variants []Variant
}

// Features describes qualitative capabilities of a library.
// These are declared, not discovered, because they require human
// knowledge about the library's behavior.
type Features struct {
    // Parser identifies the parsing engine.
    // Examples: "custom", "goldmark", "blackfriday", "blackfriday v1"
    Parser string

    // TerminalRender indicates terminal output support.
    TerminalRender bool

    // Streaming indicates true append-only streaming support
    // (not re-rendering the full document on each chunk).
    Streaming bool

    // SyntaxHighlighting describes code block highlighting.
    // Examples: "Go + Chroma", "Chroma", "Chroma v1", ""
    SyntaxHighlighting string

    // ClickableLinks indicates OSC 8 terminal hyperlink support.
    ClickableLinks bool

    // WordWrap describes wrapping behavior.
    // Examples: "auto-detect", "fixed width", ""
    WordWrap string

    // TTYDetection indicates automatic terminal detection
    // (e.g. stripping ANSI when piped).
    TTYDetection bool

    // Notes are free-form per-candidate remarks for the report.
    // Example: "Uses blackfriday v1 internally"
    Notes []string
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
    Features: Features{
        Parser:             "custom streaming",
        TerminalRender:     true,
        Streaming:          true,
        SyntaxHighlighting: "Go fast path + Chroma",
        ClickableLinks:     true,
        WordWrap:           "auto-detect",
        TTYDetection:       true,
    },
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
    Features   Features                        `json:"features"`
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
        Features: Features{
            Parser:             "custom streaming",
            TerminalRender:     true,
            Streaming:          true,
            SyntaxHighlighting: "Go fast path + Chroma",
            ClickableLinks:     true,
            WordWrap:           "auto-detect",
            TTYDetection:       true,
        },
        Variants: []Variant{
            {Name: "ours", Description: "default", Adapters: defaultAdapters()},
            {Name: "ours-reuse", Description: "parser reuse", Adapters: reusableAdapters()},
            {Name: "ours-4k", Description: "4KB chunks", Adapters: chunkedAdapters(4096)},
        },
    },
    {
        Repo: "https://github.com/yuin/goldmark",
        Features: Features{
            Parser: "goldmark",
            Notes:  []string{"De facto standard Go Markdown parser"},
        },
        Variants: []Variant{
            {Name: "goldmark", Adapters: Adapters{
                ParseFunc:  goldmarkParse,
                RenderHTML: goldmarkRenderHTML,
            }},
        },
    },
    {
        Repo: "https://github.com/charmbracelet/glamour",
        Features: Features{
            Parser:             "goldmark",
            TerminalRender:     true,
            SyntaxHighlighting: "Chroma",
            WordWrap:           "fixed width",
            Notes:              []string{"Uses goldmark internally", "Multiple built-in themes"},
        },
        Variants: []Variant{
            {Name: "glamour", Adapters: Adapters{
                RenderTerminal: glamourRender,
            }},
        },
    },
    {
        Repo: "https://github.com/russross/blackfriday",
        Features: Features{
            Parser: "blackfriday",
            Notes:  []string{"Unmaintained since 2019", "Not CommonMark compliant"},
        },
        Variants: []Variant{
            {Name: "blackfriday", Adapters: Adapters{
                ParseFunc:  blackfridayParse,
                RenderHTML: blackfridayRenderHTML,
            }},
        },
    },
    {
        Repo: "https://github.com/gomarkdown/markdown",
        Features: Features{
            Parser: "gomarkdown",
            Notes:  []string{"Active fork of blackfriday"},
        },
        Variants: []Variant{
            {Name: "gomarkdown", Adapters: Adapters{
                ParseFunc:  gomarkdownParse,
                RenderHTML: gomarkdownRenderHTML,
            }},
        },
    },
    {
        Repo: "https://github.com/MichaelMure/go-term-markdown",
        Features: Features{
            Parser:             "blackfriday v1",
            TerminalRender:     true,
            SyntaxHighlighting: "Chroma v1",
            WordWrap:           "fixed width",
            Notes:              []string{"Inline terminal images via pixterm"},
        },
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
    cmd: go run . metadata

competition:compliance:
    desc: Run compliance tests
    dir: competition
    cmd: go run . compliance

competition:bench:
    desc: Run benchmarks
    dir: competition
    cmd: go test -bench=. -benchmem -count=5 -json ./benchmarks

competition:report:
    desc: Generate COMPARISON.md from latest results
    dir: competition
    cmd: go run ./cmd/compgen --latest -out ../COMPARISON.md

competition:full:
    desc: Full pipeline run
    cmds:
        - task: competition:metadata
        - task: competition:compliance
        - task: competition:bench
        - task: competition:report
```

Each stage reads the latest `results/results-*.json`, merges its
data, and writes a new timestamped snapshot.

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

## Resolved Questions

### 1. Metadata Caching

**Decision:** Cache with 24h TTL + `--refresh` flag.

Stage 1 loads existing `results.json`, skips candidates whose
`discovered_at` is within `MaxAge` (default 24h), only clones/queries
stale entries, then merges back.

```go
type MetadataOptions struct {
    MaxAge   time.Duration // default 24h
    ForceAll bool          // --refresh flag
}
```

### 2. Our Compliance

**Decision:** Use internal event-level method until HTML renderer ships.

We already document the methodology difference in COMPARISON.md. The
competition pipeline populates our compliance by running
`TestCommonMarkCorpusClassification` and parsing the counts. No special
struct field needed — the report generator knows our variant uses
event-level testing and adds the footnote.

Once the `html` package ships (see `PLAN-html-renderer.md`), our
variants gain a `RenderHTML` adapter and compliance is measured
identically to all other candidates. The footnote is then removed.

### 3. Panic Recovery

**Decision:** Yes, always wrap adapter calls.

Unmaintained parsers (blackfriday, gomarkdown) crash on pathological
inputs. Without recovery, one panic kills the entire run.

```go
func safeCall(fn func() error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic: %v", r)
        }
    }()
    return fn()
}
```

For benchmarks: panics are recorded as `"error": "panic: ..."` and
the variant is skipped for that input. For compliance: a panic counts
as a failure.

### 4. Historical Tracking

**Decision:** Timestamp-based result files + version control.

Each pipeline run writes `results-{YYYY-MM-DDTHH:MM}.json`. The report
generator (`compgen`) reads the most recent file by default, or accepts
`--results path` to use a specific one. Old files accumulate in the
repo (version-controlled), giving natural history without extra
machinery.

```
competition/
├── results/
│   ├── results-2026-04-29T01:50.json
│   ├── results-2026-04-30T14:22.json
│   └── ...
```

`compgen --latest` picks the newest. `compgen --results results/results-2026-04-29T01:50.json`
uses a specific snapshot.

### 5. Benchmark Reproducibility

**Decision:** Run with `-count=5`, store all runs, report median.

```go
type BenchmarkResult struct {
    Category     string    `json:"category"`
    Runs         []RunData `json:"runs"`
    MedianNsOp   float64   `json:"median_ns_op"`
    MedianBOp    int64     `json:"median_b_op"`
    MedianAllocs int64     `json:"median_allocs_op"`
}

type RunData struct {
    NsOp     float64 `json:"ns_op"`
    BOp      int64   `json:"b_op"`
    AllocsOp int64   `json:"allocs_op"`
}
```

`compgen` reports median values. Raw runs are preserved for anyone
who wants to compute confidence intervals or run `benchstat`.

### 6. Benchmark Output Parsing

**Decision:** Shell out with `go test -json`.

The pipeline runner executes:
```bash
go test -bench=. -benchmem -count=5 -json ./benchmarks
```

JSON output gives structured results without regex parsing. The
pipeline collects, groups by variant+input, computes medians, and
merges into the results file.

### 7. Module Boundary

**Decision:** Separate module with `replace` directive.

```
// competition/go.mod
replace github.com/codewandler/markdown => ../
```

Benchmarks always test local code. `compgen` includes the git SHA
in the report header so results are traceable.

### 8. `benchcompare` Fate

**Decision:** Subsumed by `compgen`.

`compgen` reads `results-*.json` and produces COMPARISON.md with all
tables (speed, allocations, memory, compliance, feature matrix).
The old `benchmarks/cmd/benchcompare` is deleted in the migration.

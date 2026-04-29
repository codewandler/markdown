package competition

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BenchmarkOptions configures the benchmark pipeline stage.
type BenchmarkOptions struct {
	// ResultsDir is the directory for timestamped result files.
	ResultsDir string

	// BenchDir is the directory containing the benchmark tests.
	// Default: "benchmarks"
	BenchDir string

	// BenchPattern is the -bench flag pattern. Default: "."
	BenchPattern string

	// Count is the number of benchmark iterations (-count). Default: 3.
	Count int

	// BenchTime is the -benchtime flag. Default: "1s".
	BenchTime string

	// Timeout is the -timeout flag. Default: "300s".
	Timeout string

	// Log receives progress messages.
	Log io.Writer
}

func (o *BenchmarkOptions) benchDefaults() {
	if o.ResultsDir == "" {
		o.ResultsDir = ResultsDir
	}
	if o.BenchDir == "" {
		o.BenchDir = "benchmarks"
	}
	if o.BenchPattern == "" {
		o.BenchPattern = "."
	}
	if o.Count == 0 {
		o.Count = 3
	}
	if o.BenchTime == "" {
		o.BenchTime = "1s"
	}
	if o.Timeout == "" {
		o.Timeout = "300s"
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
}

// RunBenchmarks executes Stage 3 of the pipeline: benchmarks.
//
// It shells out to `go test -bench -benchmem -json` in the benchmarks
// directory, parses the JSON output, computes medians, and merges
// results into a new timestamped snapshot.
func RunBenchmarks(opts BenchmarkOptions) (*RunResult, error) {
	opts.benchDefaults()

	prev, err := LoadLatestResults(opts.ResultsDir)
	if err != nil {
		return nil, fmt.Errorf("load previous results: %w", err)
	}

	result := &RunResult{
		RunAt:  time.Now(),
		GitSHA: GitSHA(),
		System: CollectSystemInfo(),
	}
	if prev != nil {
		result.Candidates = prev.Candidates
	}

	fmt.Fprintf(opts.Log, "  running go test -bench=%s -count=%d ...\n",
		opts.BenchPattern, opts.Count)

	// Shell out to go test.
	cmd := exec.Command("go", "test",
		"-bench="+opts.BenchPattern,
		"-benchmem",
		"-count="+strconv.Itoa(opts.Count),
		"-benchtime="+opts.BenchTime,
		"-timeout="+opts.Timeout,
		"-json",
		"./"+opts.BenchDir,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: %w", err)
	}
	cmd.Stderr = opts.Log.(io.Writer)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start go test: %w", err)
	}

	// Parse JSON output.
	raw := parseBenchJSON(stdout)

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("go test: %w", err)
	}

	// Group by variant+input, compute medians, merge.
	grouped := groupBenchRuns(raw)
	merged := 0
	for key, runs := range grouped {
		br := BenchmarkResult{
			Category: key.category,
			Runs:     runs,
		}
		computeMedians(&br)
		mergeBenchmark(result, key, br)
		merged++
	}

	fmt.Fprintf(opts.Log, "  parsed %d benchmark results\n", merged)

	path, err := SaveResults(opts.ResultsDir, result)
	if err != nil {
		return nil, fmt.Errorf("save results: %w", err)
	}
	fmt.Fprintf(opts.Log, "  saved %s\n", path)

	return result, nil
}

// --- JSON parsing -----------------------------------------------------------

// testEvent is the JSON structure emitted by `go test -json`.
type testEvent struct {
	Action  string `json:"Action"`
	Output  string `json:"Output"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
}

// benchLine matches Go benchmark output lines.
var benchLine = regexp.MustCompile(
	`^(Benchmark\S+)-\d+\s+\d+\s+([\d.]+)\s+ns/op` +
		`(?:\s+[\d.]+\s+MB/s)?` +
		`\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`,
)

type rawBench struct {
	fullName string
	nsOp     float64
	bOp      int64
	allocsOp int64
}

func parseBenchJSON(r io.Reader) []rawBench {
	var results []rawBench
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var ev testEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Action != "output" {
			continue
		}
		m := benchLine.FindStringSubmatch(strings.TrimSpace(ev.Output))
		if m == nil {
			continue
		}
		nsOp, _ := strconv.ParseFloat(m[2], 64)
		bOp, _ := strconv.ParseInt(m[3], 10, 64)
		allocsOp, _ := strconv.ParseInt(m[4], 10, 64)
		results = append(results, rawBench{
			fullName: m[1],
			nsOp:     nsOp,
			bOp:      bOp,
			allocsOp: allocsOp,
		})
	}
	return results
}

// --- Grouping + medians -----------------------------------------------------

type benchKey struct {
	category string // "parse", "render", "pathological", etc.
	variant  string // "ours", "goldmark", etc.
	input    string // "Spec", "README", etc.
}

func groupBenchRuns(raw []rawBench) map[benchKey][]RunData {
	groups := map[benchKey][]RunData{}
	for _, r := range raw {
		key := parseBenchName(r.fullName)
		groups[key] = append(groups[key], RunData{
			NsOp:     r.nsOp,
			BOp:      r.bOp,
			AllocsOp: r.allocsOp,
		})
	}
	return groups
}

// parseBenchName splits "BenchmarkParse/ours/README" into category, variant, input.
//
// Most benchmarks follow the 3-part pattern: Category/Variant/Input.
// Special benchmarks like ChunkSize and Highlight are 2-part:
// ChunkSize/4K and Highlight/go-fast-path. These are always "ours"
// benchmarks, so we set variant="ours" and use the second part as input.
func parseBenchName(name string) benchKey {
	name = strings.TrimPrefix(name, "Benchmark")
	parts := strings.SplitN(name, "/", 3)

	key := benchKey{}
	if len(parts) >= 1 {
		key.category = strings.ToLower(parts[0])
	}

	switch key.category {
	case "chunksize", "highlight":
		// 2-part: category is the section, variant is always "ours",
		// the sub-name is the input.
		key.variant = "ours"
		if len(parts) >= 2 {
			key.input = parts[1]
		}
	default:
		// 3-part: Category/Variant/Input
		if len(parts) >= 2 {
			key.variant = parts[1]
		}
		if len(parts) >= 3 {
			key.input = parts[2]
		}
	}
	return key
}

func computeMedians(br *BenchmarkResult) {
	if len(br.Runs) == 0 {
		return
	}

	ns := make([]float64, len(br.Runs))
	bs := make([]int64, len(br.Runs))
	as := make([]int64, len(br.Runs))
	for i, r := range br.Runs {
		ns[i] = r.NsOp
		bs[i] = r.BOp
		as[i] = r.AllocsOp
	}

	sort.Float64s(ns)
	sort.Slice(bs, func(i, j int) bool { return bs[i] < bs[j] })
	sort.Slice(as, func(i, j int) bool { return as[i] < as[j] })

	mid := len(ns) / 2
	br.MedianNsOp = ns[mid]
	br.MedianBOp = bs[mid]
	br.MedianAllocs = as[mid]
}

// --- Merge ------------------------------------------------------------------

// benchResultKey returns the key used to store a benchmark result.
// It includes the category to avoid collisions when the same variant
// has both parse and render benchmarks for the same input.
func benchResultKey(key benchKey) string {
	return key.category + "/" + key.input
}

func mergeBenchmark(r *RunResult, key benchKey, br BenchmarkResult) {
	storeKey := benchResultKey(key)

	// Find the candidate that owns this variant.
	// First check existing variant map entries.
	for i := range r.Candidates {
		c := &r.Candidates[i]
		for vName := range c.Variants {
			if vName == key.variant {
				vr := c.Variants[vName]
				if vr.Benchmarks == nil {
					vr.Benchmarks = make(map[string]BenchmarkResult)
				}
				vr.Benchmarks[storeKey] = br
				c.Variants[vName] = vr
				return
			}
		}
	}

	// Variant not in map yet — find the candidate by matching against
	// the declared All list, then create the variant entry.
	for i := range r.Candidates {
		c := &r.Candidates[i]
		if ownsVariant(c.Repo, key.variant) {
			if c.Variants == nil {
				c.Variants = make(map[string]VariantResult)
			}
			vr := c.Variants[key.variant]
			vr.Description = key.variant
			if vr.Benchmarks == nil {
				vr.Benchmarks = make(map[string]BenchmarkResult)
			}
			vr.Benchmarks[storeKey] = br
			c.Variants[key.variant] = vr
			return
		}
	}

	// Still not found — skip silently.
}

// ownsVariant checks whether a repo URL owns a variant name by
// looking it up in the declared All candidates.
func ownsVariant(repo, variantName string) bool {
	for _, c := range All {
		if c.Repo != repo {
			continue
		}
		for _, v := range c.Variants {
			if v.Name == variantName {
				return true
			}
		}
	}
	return false
}

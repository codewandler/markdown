package competition

import "time"

// RunResult is the complete output of one pipeline run.
// Serialized as results/results-{timestamp}.json.
type RunResult struct {
	RunAt      time.Time         `json:"run_at"`
	GitSHA     string            `json:"git_sha,omitempty"`
	System     SystemInfo        `json:"system"`
	Candidates []CandidateResult `json:"candidates"`
}

// SystemInfo describes the machine that produced the results.
type SystemInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPU       string `json:"cpu"`
	GoVersion string `json:"go_version"`
	Hostname  string `json:"hostname"`
}

// CandidateResult holds all discovered and measured data for one repo.
type CandidateResult struct {
	Repo     string                   `json:"repo"`
	Features Features                 `json:"features"`
	Metadata MetadataResult           `json:"metadata"`
	Variants map[string]VariantResult `json:"variants"`
}

// VariantResult holds measured data for one named variant.
type VariantResult struct {
	Description string                     `json:"description"`
	Compliance  *ComplianceResult          `json:"compliance,omitempty"`
	Benchmarks  map[string]BenchmarkResult `json:"benchmarks,omitempty"`
}

// MetadataResult holds data discovered from GitHub and the repo clone.
type MetadataResult struct {
	// Identity (from GitHub API / gh CLI)
	Name        string    `json:"name"`
	Owner       string    `json:"owner"`
	Description string    `json:"description"`
	License     string    `json:"license"`
	Stars       int       `json:"stars"`
	OpenIssues  int       `json:"open_issues"`
	Forks       int       `json:"forks"`
	LastCommit  time.Time `json:"last_commit"`

	// Code metrics (from clone + inspection)
	GoFiles   int `json:"go_files"`
	GoLines   int `json:"go_lines"`
	TestFiles int `json:"test_files"`
	TestLines int `json:"test_lines"`

	// Dependencies (from go.mod / go list)
	DirectDeps     int      `json:"direct_deps"`
	TransitiveDeps int      `json:"transitive_deps"`
	DepNames       []string `json:"dep_names,omitempty"`

	// Test coverage (from go test -cover)
	TestCoverage float64 `json:"test_coverage"`

	DiscoveredAt time.Time `json:"discovered_at"`
}

// ComplianceResult holds spec test results for a variant.
type ComplianceResult struct {
	CommonMark     SpecResult `json:"commonmark"`
	GFM            SpecResult `json:"gfm"`
	GFMExtensions  SpecResult `json:"gfm_extensions"`
	GFMRegression  SpecResult `json:"gfm_regression"`
}

// SpecResult holds pass/total counts for one spec suite.
type SpecResult struct {
	Version    string             `json:"version"`
	Pass       int                `json:"pass"`
	Total      int                `json:"total"`
	Percentage float64            `json:"percentage"`
	Sections   map[string]Section `json:"sections,omitempty"`
}

// Section holds pass/total counts for one spec section.
type Section struct {
	Pass  int `json:"pass"`
	Total int `json:"total"`
}

// BenchmarkResult holds benchmark measurements for one
// variant + input combination.
type BenchmarkResult struct {
	Category     string    `json:"category"` // "parse", "render", "stream"
	Runs         []RunData `json:"runs"`
	MedianNsOp   float64   `json:"median_ns_op"`
	MedianBOp    int64     `json:"median_b_op"`
	MedianAllocs int64     `json:"median_allocs_op"`
	Error        string    `json:"error,omitempty"` // e.g. "panic: ..."
}

// RunData holds raw measurements from a single benchmark iteration.
type RunData struct {
	NsOp     float64 `json:"ns_op"`
	BOp      int64   `json:"b_op"`
	AllocsOp int64   `json:"allocs_op"`
}

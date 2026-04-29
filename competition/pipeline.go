package competition

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// MetadataOptions configures the metadata discovery stage.
type MetadataOptions struct {
	// MaxAge is the staleness threshold. Candidates with metadata
	// newer than MaxAge are skipped. Default: 24h.
	MaxAge time.Duration

	// ForceAll ignores MaxAge and re-discovers all candidates.
	ForceAll bool

	// ResultsDir is the directory for timestamped result files.
	// Default: "results" (relative to competition/).
	ResultsDir string

	// TmpDir is the base directory for shallow clones.
	// Default: os.TempDir()/competition.
	TmpDir string

	// Log receives progress messages. Default: os.Stderr.
	Log io.Writer
}

func (o *MetadataOptions) defaults() {
	if o.MaxAge == 0 {
		o.MaxAge = DefaultMaxAge
	}
	if o.ResultsDir == "" {
		o.ResultsDir = ResultsDir
	}
	if o.TmpDir == "" {
		o.TmpDir = filepath.Join(os.TempDir(), "competition")
	}
	if o.Log == nil {
		o.Log = os.Stderr
	}
}

// RunMetadata executes Stage 1 of the pipeline: metadata discovery.
//
// It loads the latest results file, skips candidates whose metadata
// is still fresh, discovers metadata for stale candidates, merges
// the results, and saves a new timestamped snapshot.
func RunMetadata(candidates []Candidate, opts MetadataOptions) (*RunResult, error) {
	opts.defaults()

	// Load existing results (nil if none exist).
	prev, err := LoadLatestResults(opts.ResultsDir)
	if err != nil {
		return nil, fmt.Errorf("load previous results: %w", err)
	}

	// Start a new result, carrying forward previous data.
	result := &RunResult{
		RunAt:  time.Now(),
		GitSHA: GitSHA(),
		System: CollectSystemInfo(),
	}
	if prev != nil {
		result.Candidates = prev.Candidates
	}

	for _, c := range candidates {
		if !opts.ForceAll && IsFresh(prev, c.Repo, opts.MaxAge) {
			fmt.Fprintf(opts.Log, "  skip %s (fresh)\n", c.Repo)
			continue
		}

		fmt.Fprintf(opts.Log, "  discover %s ...\n", c.Repo)
		meta, err := DiscoverMetadata(c.Repo, opts.TmpDir)
		if err != nil {
			fmt.Fprintf(opts.Log, "  ERROR %s: %v\n", c.Repo, err)
			continue
		}
		MergeMetadata(result, c.Repo, c.Features, meta)
		fmt.Fprintf(opts.Log, "  done %s (%s, %d stars, %d Go files, %d lines)\n",
			c.Repo, meta.License, meta.Stars, meta.GoFiles, meta.GoLines)
	}

	// Save new snapshot.
	path, err := SaveResults(opts.ResultsDir, result)
	if err != nil {
		return nil, fmt.Errorf("save results: %w", err)
	}
	fmt.Fprintf(opts.Log, "  saved %s\n", path)

	return result, nil
}

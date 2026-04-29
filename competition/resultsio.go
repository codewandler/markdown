package competition

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// ResultsDir is the default directory for timestamped result files.
	ResultsDir = "results"

	// DefaultMaxAge is the default staleness threshold for metadata.
	DefaultMaxAge = 24 * time.Hour
)

// ResultsFilename returns the filename for a new results snapshot.
func ResultsFilename(t time.Time) string {
	return fmt.Sprintf("results-%s.json", t.Format("2006-01-02T15-04"))
}

// SaveResults writes a RunResult as indented JSON to the results directory.
// It creates the directory if needed.
func SaveResults(dir string, r *RunResult) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create results dir: %w", err)
	}
	name := ResultsFilename(r.RunAt)
	path := filepath.Join(dir, name)

	data, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		return "", fmt.Errorf("marshal results: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// LoadLatestResults reads the most recent results file from dir.
// Returns nil (not an error) if no results exist yet.
func LoadLatestResults(dir string) (*RunResult, error) {
	path, err := latestResultsPath(dir)
	if err != nil {
		return nil, nil // no results yet
	}
	return LoadResults(path)
}

// LoadResults reads a RunResult from a specific file path.
func LoadResults(path string) (*RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var r RunResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &r, nil
}

// latestResultsPath returns the path to the most recent results file.
func latestResultsPath(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "results-") && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no results files in %s", dir)
	}
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1]), nil
}

// MergeMetadata updates or inserts metadata for a repo in the result set.
// If the repo already exists, its metadata is replaced. If not, a new
// CandidateResult is appended.
func MergeMetadata(r *RunResult, repo string, features Features, meta MetadataResult) {
	for i := range r.Candidates {
		if r.Candidates[i].Repo == repo {
			r.Candidates[i].Metadata = meta
			r.Candidates[i].Features = features
			return
		}
	}
	r.Candidates = append(r.Candidates, CandidateResult{
		Repo:     repo,
		Features: features,
		Metadata: meta,
		Variants: make(map[string]VariantResult),
	})
}

// IsFresh reports whether the metadata for a repo was discovered
// within maxAge of now.
func IsFresh(r *RunResult, repo string, maxAge time.Duration) bool {
	if r == nil {
		return false
	}
	for _, c := range r.Candidates {
		if c.Repo == repo {
			return time.Since(c.Metadata.DiscoveredAt) < maxAge
		}
	}
	return false
}

// GitSHA returns the current git HEAD short SHA, or "" on error.
func GitSHA() string {
	out, err := gitOutput("rev-parse", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// gitOutput runs a git command and returns trimmed stdout.
func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

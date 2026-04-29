package competition

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DiscoverMetadata collects metadata for a candidate by querying
// the GitHub API (via gh CLI) and cloning the repo for code metrics.
// tmpDir is the base directory for shallow clones.
func DiscoverMetadata(repo string, tmpDir string) (MetadataResult, error) {
	var m MetadataResult
	m.DiscoveredAt = time.Now()

	// Extract owner/name from URL.
	ownerName, err := repoOwnerName(repo)
	if err != nil {
		return m, err
	}

	// Stage 1a: GitHub API via gh CLI.
	if err := discoverGitHub(&m, ownerName); err != nil {
		return m, fmt.Errorf("github metadata: %w", err)
	}

	// Stage 1b: Clone + code metrics.
	cloneDir := filepath.Join(tmpDir, m.Owner, m.Name)
	if err := shallowClone(repo, cloneDir); err != nil {
		return m, fmt.Errorf("clone: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	if err := discoverCodeMetrics(&m, cloneDir); err != nil {
		return m, fmt.Errorf("code metrics: %w", err)
	}

	if err := discoverDeps(&m, cloneDir); err != nil {
		// Non-fatal: some repos may not be Go modules.
		m.DirectDeps = -1
	}
	_ = err

	if err := discoverCoverage(&m, cloneDir); err != nil {
		// Non-fatal: tests may not run in a shallow clone.
		m.TestCoverage = -1
	}

	return m, nil
}

// repoOwnerName extracts "owner/name" from a GitHub URL.
func repoOwnerName(repo string) (string, error) {
	// Handle https://github.com/owner/name and github.com/owner/name
	s := repo
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")
	parts := strings.SplitN(s, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("cannot parse repo URL: %s", repo)
	}
	return parts[0] + "/" + parts[1], nil
}

// --- GitHub API -------------------------------------------------------------

// ghRepoJSON matches the gh CLI JSON output for repo metadata.
type ghRepoJSON struct {
	Name        string `json:"name"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description string `json:"description"`
	LicenseInfo *struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"licenseInfo"`
	StargazerCount int `json:"stargazerCount"`
	Issues         struct {
		TotalCount int `json:"totalCount"`
	} `json:"issues"`
	ForkCount int       `json:"forkCount"`
	PushedAt  time.Time `json:"pushedAt"`
}

func discoverGitHub(m *MetadataResult, ownerName string) error {
	out, err := exec.Command("gh", "repo", "view", ownerName,
		"--json", "name,owner,description,licenseInfo,stargazerCount,issues,forkCount,pushedAt",
	).Output()
	if err != nil {
		return fmt.Errorf("gh repo view %s: %w", ownerName, err)
	}

	var gh ghRepoJSON
	if err := json.Unmarshal(out, &gh); err != nil {
		return fmt.Errorf("parse gh output: %w", err)
	}

	m.Name = gh.Name
	m.Owner = gh.Owner.Login
	m.Description = gh.Description
	if gh.LicenseInfo != nil {
		m.License = gh.LicenseInfo.Name
	}
	m.Stars = gh.StargazerCount
	m.OpenIssues = gh.Issues.TotalCount
	m.Forks = gh.ForkCount
	m.LastCommit = gh.PushedAt
	return nil
}

// --- Clone ------------------------------------------------------------------

func shallowClone(repo, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	// Remove any previous clone.
	os.RemoveAll(dir)
	cmd := exec.Command("git", "clone", "--depth=1", "--quiet", repo, dir)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// --- Code metrics -----------------------------------------------------------

func discoverCodeMetrics(m *MetadataResult, dir string) error {
	goFiles, goLines, err := countLines(dir, "*.go", "*_test.go", false)
	if err != nil {
		return err
	}
	testFiles, testLines, err := countLines(dir, "*_test.go", "", true)
	if err != nil {
		return err
	}
	m.GoFiles = goFiles
	m.GoLines = goLines
	m.TestFiles = testFiles
	m.TestLines = testLines
	return nil
}

// countLines counts .go files and their lines.
// If matchTest is true, only *_test.go files are counted.
// If matchTest is false, *_test.go files matching excludePattern are excluded.
func countLines(dir, pattern, excludePattern string, matchTest bool) (files, lines int, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "vendor" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		name := filepath.Base(path)
		if matchTest {
			matched, _ := filepath.Match(pattern, name)
			if !matched {
				return nil
			}
		} else {
			matched, _ := filepath.Match(pattern, name)
			if !matched {
				return nil
			}
			if excludePattern != "" {
				excluded, _ := filepath.Match(excludePattern, name)
				if excluded {
					return nil
				}
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable
		}
		files++
		lines += countNewlines(data)
		return nil
	})
	return files, lines, err
}

func countNewlines(data []byte) int {
	n := 0
	for _, b := range data {
		if b == '\n' {
			n++
		}
	}
	// Count last line if it doesn't end with newline.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
}

// --- Dependencies -----------------------------------------------------------

func discoverDeps(m *MetadataResult, dir string) error {
	// Direct deps from go.mod.
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}", "all")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// First line is the module itself.
	if len(lines) > 0 {
		lines = lines[1:]
	}
	m.TransitiveDeps = len(lines)
	m.DepNames = lines

	// Count direct deps from go.mod require block.
	modData, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return err
	}
	m.DirectDeps = countDirectDeps(string(modData))
	return nil
}

func countDirectDeps(gomod string) int {
	count := 0
	inRequire := false
	for _, line := range strings.Split(gomod, "\n") {
		line = strings.TrimSpace(line)
		if line == "require (" {
			inRequire = true
			continue
		}
		if line == ")" {
			inRequire = false
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			count++
		}
		// Single-line require.
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			count++
		}
	}
	return count
}

// --- Test coverage ----------------------------------------------------------

func discoverCoverage(m *MetadataResult, dir string) error {
	cmd := exec.Command("go", "test", "-cover", "-short", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	m.TestCoverage = parseCoverage(string(out))
	return nil
}

// parseCoverage extracts the average coverage percentage from
// `go test -cover` output lines like:
//
//	ok  pkg  0.5s  coverage: 82.3% of statements
func parseCoverage(output string) float64 {
	var total float64
	var count int
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, "coverage: ")
		if idx < 0 {
			continue
		}
		s := line[idx+len("coverage: "):]
		if pct := strings.Index(s, "%"); pct > 0 {
			if v, err := strconv.ParseFloat(s[:pct], 64); err == nil {
				total += v
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

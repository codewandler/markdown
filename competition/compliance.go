package competition

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/codewandler/markdown/internal/commonmarktests"
	"github.com/codewandler/markdown/internal/gfmtests"
)

// ComplianceOptions configures the compliance testing stage.
type ComplianceOptions struct {
	// ResultsDir is the directory for timestamped result files.
	ResultsDir string

	// Log receives progress messages.
	Log io.Writer
}

func (o *ComplianceOptions) complianceDefaults() {
	if o.ResultsDir == "" {
		o.ResultsDir = ResultsDir
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
}

// RunCompliance executes Stage 2 of the pipeline: compliance testing.
//
// It loads the latest results, runs CommonMark and GFM spec suites
// against every variant that has a RenderHTML adapter, merges the
// results, and saves a new timestamped snapshot.
func RunCompliance(candidates []Candidate, opts ComplianceOptions) (*RunResult, error) {
	opts.complianceDefaults()

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

	// Load spec corpora.
	cmExamples, err := commonmarktests.Load()
	if err != nil {
		return nil, fmt.Errorf("load CommonMark corpus: %w", err)
	}
	gfmExamples, err := gfmtests.Load()
	if err != nil {
		return nil, fmt.Errorf("load GFM corpus: %w", err)
	}
	gfmExtExamples, err := gfmtests.LoadExtensions()
	if err != nil {
		return nil, fmt.Errorf("load GFM extensions corpus: %w", err)
	}
	gfmRegExamples, err := gfmtests.LoadRegression()
	if err != nil {
		return nil, fmt.Errorf("load GFM regression corpus: %w", err)
	}

	for _, c := range candidates {
		for _, v := range c.Variants {
			if v.Adapters.RenderHTML == nil {
				continue
			}

			fmt.Fprintf(opts.Log, "  compliance %s/%s ...\n", displayNameFromCandidate(c), v.Name)

			cm := runSpecSuite(v.Adapters.RenderHTML, cmExamples)
			cm.Version = "0.31.2"

			gfm := runGFMSuite(v.Adapters, gfmExamples)
			gfm.Version = "0.29"

			// Extensions and regression corpora assume all extensions active.
			allExtsRender := allExtsRenderer(v.Adapters)
			gfmExt := runGFMSuiteAllExts(allExtsRender, gfmExtExamples)
			gfmReg := runGFMSuiteAllExts(allExtsRender, gfmRegExamples)

			cr := &ComplianceResult{
				CommonMark:    cm,
				GFM:           gfm,
				GFMExtensions: gfmExt,
				GFMRegression: gfmReg,
			}

			mergeCompliance(result, c.Repo, v.Name, v.Description, cr)

			totalPass := gfm.Pass + gfmExt.Pass + gfmReg.Pass
			totalCount := gfm.Total + gfmExt.Total + gfmReg.Total
			fmt.Fprintf(opts.Log, "    CommonMark: %d/%d (%.1f%%)\n",
				cm.Pass, cm.Total, cm.Percentage)
			fmt.Fprintf(opts.Log, "    GFM total:  %d/%d (%.1f%%)\n",
				totalPass, totalCount, float64(totalPass)/float64(totalCount)*100)
		}
	}

	path, err := SaveResults(opts.ResultsDir, result)
	if err != nil {
		return nil, fmt.Errorf("save results: %w", err)
	}
	fmt.Fprintf(opts.Log, "  saved %s\n", path)

	return result, nil
}

// runSpecSuite runs a spec suite against a render function.
func runSpecSuite(renderHTML func(io.Reader, io.Writer) error, examples []commonmarktests.Example) SpecResult {
	sections := map[string]Section{}
	pass, total := 0, len(examples)

	for _, ex := range examples {
		var buf bytes.Buffer
		r := strings.NewReader(ex.Markdown)
		err := SafeCall(func() error {
			return renderHTML(r, &buf)
		})

		matched := false
		if err == nil {
			matched = normalizeHTML(buf.String()) == normalizeHTML(ex.HTML)
		}

		sec := sections[ex.Section]
		sec.Total++
		if matched {
			sec.Pass++
			pass++
		}
		sections[ex.Section] = sec
	}

	pct := 0.0
	if total > 0 {
		pct = float64(pass) / float64(total) * 100
	}
	return SpecResult{
		Pass:       pass,
		Total:      total,
		Percentage: pct,
		Sections:   sections,
	}
}

// runGFMSuite runs the GFM spec suite, enabling extensions per-example
// as specified by the GFM spec (each example declares which extensions
// it requires).
func runGFMSuite(adapters Adapters, examples []gfmtests.Example) SpecResult {
	sections := map[string]Section{}
	pass, total := 0, len(examples)

	for _, ex := range examples {
		var exts []string
		if ex.Extension != "" {
			exts = []string{ex.Extension}
		}

		// Pick the right renderer based on whether extensions are needed.
		var renderFn func(io.Reader, io.Writer) error
		if len(exts) > 0 && adapters.RenderGFMHTML != nil {
			renderFn = func(r io.Reader, w io.Writer) error {
				return adapters.RenderGFMHTML(r, w, exts)
			}
		} else {
			renderFn = adapters.RenderHTML
		}
		if renderFn == nil {
			continue
		}

		var buf bytes.Buffer
		r := strings.NewReader(ex.Markdown)
		err := SafeCall(func() error {
			return renderFn(r, &buf)
		})

		matched := false
		if err == nil {
			matched = normalizeHTML(buf.String()) == normalizeHTML(ex.HTML)
		}

		sec := sections[ex.Section]
		sec.Total++
		if matched {
			sec.Pass++
			pass++
		}
		sections[ex.Section] = sec
	}

	pct := 0.0
	if total > 0 {
		pct = float64(pass) / float64(total) * 100
	}
	return SpecResult{
		Pass:       pass,
		Total:      total,
		Percentage: pct,
		Sections:   sections,
	}
}

// allExtsRenderer returns a render function with all GFM extensions enabled.
// For candidates with RenderGFMHTML, it passes all extensions. Otherwise
// falls back to RenderHTML (competitors like goldmark have extensions always on).
func allExtsRenderer(a Adapters) func(io.Reader, io.Writer) error {
	allExts := []string{"autolink", "tagfilter", "table", "strikethrough"}
	if a.RenderGFMHTML != nil {
		return func(r io.Reader, w io.Writer) error {
			return a.RenderGFMHTML(r, w, allExts)
		}
	}
	return a.RenderHTML
}

// runGFMSuiteAllExts runs a GFM corpus with all extensions enabled
// (for extensions.txt and regression.txt which have no per-example tags).
func runGFMSuiteAllExts(renderHTML func(io.Reader, io.Writer) error, examples []gfmtests.Example) SpecResult {
	adapted := make([]commonmarktests.Example, 0, len(examples))
	for _, ex := range examples {
		if strings.Contains(ex.HTML, "<IGNORE>") {
			continue
		}
		adapted = append(adapted, commonmarktests.Example{
			Markdown: ex.Markdown,
			HTML:     ex.HTML,
			Example:  ex.Example,
			Section:  ex.Section,
		})
	}
	result := runSpecSuite(renderHTML, adapted)
	// Set total to original count (including IGNORE) for consistent reporting.
	result.Total = len(examples)
	result.Pass += len(examples) - len(adapted) // count IGNORE as pass
	if result.Total > 0 {
		result.Percentage = float64(result.Pass) / float64(result.Total) * 100
	}
	return result
}

// normalizeHTML trims whitespace and normalizes newlines.
func normalizeHTML(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

// mergeCompliance updates or inserts compliance data for a variant.
func mergeCompliance(r *RunResult, repo, variantName, description string, cr *ComplianceResult) {
	for i := range r.Candidates {
		if r.Candidates[i].Repo != repo {
			continue
		}
		if r.Candidates[i].Variants == nil {
			r.Candidates[i].Variants = make(map[string]VariantResult)
		}
		vr := r.Candidates[i].Variants[variantName]
		vr.Description = description
		vr.Compliance = cr
		r.Candidates[i].Variants[variantName] = vr
		return
	}
	// Candidate not found — create it.
	r.Candidates = append(r.Candidates, CandidateResult{
		Repo: repo,
		Variants: map[string]VariantResult{
			variantName: {
				Description: description,
				Compliance:  cr,
			},
		},
	})
}

// displayNameFromCandidate extracts a short name from a Candidate.
func displayNameFromCandidate(c Candidate) string {
	parts := strings.Split(strings.TrimSuffix(c.Repo, "/"), "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return c.Repo
}

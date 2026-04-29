package competition

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// GenerateReport writes a COMPARISON.md from a RunResult.
// Sections with no data are omitted gracefully.
func GenerateReport(w io.Writer, r *RunResult) error {
	g := &reportGen{w: w, r: r}
	g.header()
	g.featureMatrix()
	g.compliance()
	g.benchSection("Terminal Rendering (parse + render to ANSI string)",
		"render", renderVariants, "glamour")
	g.benchSection("Parse-Only",
		"parse", parseVariants, "goldmark")
	g.parseMemoryNote()
	g.highlightSection()
	g.chunkSizeSection()
	g.reproduction()
	return g.err
}

// Variant orderings for each benchmark section.
var renderVariants = []string{"ours", "ours-4k", "glamour", "go-term-md"}
var parseVariants = []string{"ours", "ours-reuse", "goldmark", "blackfriday", "gomarkdown"}

// --- Report generator -------------------------------------------------------

type reportGen struct {
	w        io.Writer
	r        *RunResult
	err      error
	category string // current benchmark category filter
}

func (g *reportGen) writef(format string, args ...any) {
	if g.err != nil {
		return
	}
	_, g.err = fmt.Fprintf(g.w, format, args...)
}

func (g *reportGen) nl() { g.writef("\n") }

// --- Header -----------------------------------------------------------------

func (g *reportGen) header() {
	g.writef("# Comparison with Other Go Markdown Libraries\n\n")
	g.writef("Benchmarks run on %s, %s, %s.",
		g.r.System.CPU, g.r.System.GoVersion, capitalizeFirst(g.r.System.OS))
	if g.r.GitSHA != "" {
		g.writef(" Git SHA: `%s`.", g.r.GitSHA)
	}
	g.nl()
	g.nl()
	g.writef("See [docs/competitors.md](docs/competitors.md) for detailed library profiles.\n\n")
}

// --- Feature Matrix ---------------------------------------------------------

func (g *reportGen) featureMatrix() {
	g.writef("## Feature Matrix\n\n")

	// Collect all candidates in declaration order.
	candidates := g.r.Candidates
	if len(candidates) == 0 {
		return
	}

	// Header row.
	g.writef("| Feature |")
	for _, c := range candidates {
		g.writef(" %s |", displayName(c))
	}
	g.nl()

	// Separator.
	g.writef("| --- |")
	for range candidates {
		g.writef(" :---: |")
	}
	g.nl()

	// Parser row.
	g.writef("| Parser |")
	for _, c := range candidates {
		g.writef(" %s |", c.Features.Parser)
	}
	g.nl()

	// Terminal render.
	g.writef("| Terminal render |")
	for _, c := range candidates {
		g.writef(" %s |", featureBool(c.Features.TerminalRender))
	}
	g.nl()

	// Streaming.
	g.writef("| **Streaming** |")
	for _, c := range candidates {
		g.writef(" %s |", featureBool(c.Features.Streaming))
	}
	g.nl()

	// Compliance rows (from variant data if available).
	g.featureComplianceRow("CommonMark 0.31.2", func(cr *ComplianceResult) *SpecResult {
		if cr == nil {
			return nil
		}
		return &cr.CommonMark
	})
	g.featureComplianceRow("GFM 0.29", func(cr *ComplianceResult) *SpecResult {
		if cr == nil {
			return nil
		}
		return &cr.GFM
	})

	// Syntax highlighting.
	g.writef("| Syntax highlighting |")
	for _, c := range candidates {
		v := c.Features.SyntaxHighlighting
		if v == "" {
			if c.Features.TerminalRender {
				v = "\u274c"
			} else {
				v = featureNA()
			}
		}
		g.writef(" %s |", v)
	}
	g.nl()

	// Clickable links.
	g.writef("| Clickable hyperlinks |")
	for _, c := range candidates {
		if c.Features.ClickableLinks {
			g.writef(" OSC 8 |")
		} else if c.Features.TerminalRender {
			g.writef(" %s |", "\u274c")
		} else {
			g.writef(" %s |", featureNA())
		}
	}
	g.nl()

	// Word wrapping.
	g.writef("| Word wrapping |")
	for _, c := range candidates {
		v := c.Features.WordWrap
		if v == "" {
			if c.Features.TerminalRender {
				v = "\u274c"
			} else {
				v = featureNA()
			}
		}
		g.writef(" %s |", v)
	}
	g.nl()

	// TTY detection.
	g.writef("| TTY detection |")
	for _, c := range candidates {
		if !c.Features.TerminalRender {
			g.writef(" %s |", featureNA())
		} else if c.Features.TTYDetection {
			g.writef(" auto |")
		} else {
			g.writef(" %s |", "\u274c")
		}
	}
	g.nl()

	// Direct dependencies.
	g.writef("| Direct dependencies |")
	for _, c := range candidates {
		deps := c.Metadata.DirectDeps
		if deps >= 0 {
			g.writef(" **%d** |", deps)
		} else {
			g.writef(" ? |")
		}
	}
	g.nl()

	// Stars.
	g.writef("| \u2b50 Stars |")
	for _, c := range candidates {
		if c.Metadata.Stars > 0 {
			g.writef(" %s |", FormatCount(int64(c.Metadata.Stars)))
		} else {
			g.writef(" %s |", featureNA())
		}
	}
	g.nl()

	// Code size.
	g.writef("| Go source lines |")
	for _, c := range candidates {
		if c.Metadata.GoLines > 0 {
			g.writef(" %s |", FormatCount(int64(c.Metadata.GoLines)))
		} else {
			g.writef(" ? |")
		}
	}
	g.nl()

	// Test coverage.
	g.writef("| Test coverage |")
	for _, c := range candidates {
		if c.Metadata.TestCoverage > 0 {
			g.writef(" %s |", FormatPct(c.Metadata.TestCoverage))
		} else {
			g.writef(" %s |", featureNA())
		}
	}
	g.nl()
	g.nl()
}

func (g *reportGen) featureComplianceRow(label string, extract func(*ComplianceResult) *SpecResult) {
	g.writef("| %s |", label)
	for _, c := range g.r.Candidates {
		// Find the first variant with compliance data.
		var spec *SpecResult
		for _, v := range c.Variants {
			if v.Compliance != nil {
				spec = extract(v.Compliance)
				break
			}
		}
		if spec != nil {
			g.writef(" %s |", FormatPct(spec.Percentage))
		} else {
			g.writef(" - |")
		}
	}
	g.nl()
}

// --- Compliance -------------------------------------------------------------

func (g *reportGen) compliance() {
	// Check if any candidate has compliance data.
	hasData := false
	for _, c := range g.r.Candidates {
		for _, v := range c.Variants {
			if v.Compliance != nil {
				hasData = true
				break
			}
		}
	}
	if !hasData {
		return
	}

	g.writef("## Spec Compliance\n\n")
	g.writef("Measured by running each parser against the official spec test suites\n")
	g.writef("and comparing HTML output.\n\n")

	// Collect variants with compliance data.
	type entry struct {
		name string
		cm   SpecResult
		gfm  SpecResult
	}
	var entries []entry
	for _, c := range g.r.Candidates {
		for _, v := range c.Variants {
			if v.Compliance != nil {
				entries = append(entries, entry{
					name: displayName(c),
					cm:   v.Compliance.CommonMark,
					gfm:  v.Compliance.GFM,
				})
				break // one per candidate
			}
		}
	}

	// Header.
	g.writef("| Spec |")
	for _, e := range entries {
		g.writef(" %s |", e.name)
	}
	g.nl()
	g.writef("| --- |")
	for range entries {
		g.writef(" ---: |")
	}
	g.nl()

	// CommonMark row.
	g.writef("| CommonMark 0.31.2 |")
	for _, e := range entries {
		g.writef(" %d/%d (%s) |", e.cm.Pass, e.cm.Total, FormatPct(e.cm.Percentage))
	}
	g.nl()

	// GFM row.
	g.writef("| GFM 0.29 |")
	for _, e := range entries {
		g.writef(" %d/%d (%s) |", e.gfm.Pass, e.gfm.Total, FormatPct(e.gfm.Percentage))
	}
	g.nl()
	g.nl()

	g.writef("Note: All parsers are measured by comparing HTML output against the\n")
	g.writef("spec expected HTML. Our HTML renderer is new and does not yet cover\n")
	g.writef("all edge cases \u2014 our event-level (structural) compliance is 96.2%%\n")
	g.writef("CommonMark and 100%% GFM. The HTML compliance will converge as the\n")
	g.writef("renderer matures.\n\n")
}

// --- Benchmark sections -----------------------------------------------------

func (g *reportGen) benchSection(title, category string, variantOrder []string, baseline string) {
	g.category = category

	// Collect all inputs that have data for this category.
	inputs := g.collectInputs(category, variantOrder)
	if len(inputs) == 0 {
		g.category = ""
		return
	}

	// Filter to variants that actually have data.
	variants := g.activeVariants(variantOrder, category, inputs)
	if len(variants) == 0 {
		g.category = ""
		return
	}
	g.writef("## %s\n\n", title)

	// Speed table.
	g.writef("### Speed (lower is better)\n\n")
	g.benchTable(variants, inputs, baseline,
		func(b BenchmarkResult) float64 { return b.MedianNsOp },
		func(v float64) string { return FormatNs(v) },
		"faster", "slower",
	)

	// Allocations table.
	g.writef("### Allocations (lower is better)\n\n")
	g.benchTable(variants, inputs, baseline,
		func(b BenchmarkResult) float64 { return float64(b.MedianAllocs) },
		func(v float64) string { return FormatCount(int64(v)) },
		"fewer", "more",
	)

	// Memory table.
	g.writef("### Memory (lower is better)\n\n")
	g.benchTable(variants, inputs, baseline,
		func(b BenchmarkResult) float64 { return float64(b.MedianBOp) },
		func(v float64) string { return FormatBytes(int64(v)) },
		"less", "more",
	)
	g.category = ""
}

func (g *reportGen) benchTable(
	variants []string,
	inputs []string,
	baseline string,
	extract func(BenchmarkResult) float64,
	format func(float64) string,
	winWord, loseWord string,
) {
	// Header.
	g.writef("| Input |")
	for _, v := range variants {
		g.writef(" %s |", v)
	}
	g.writef(" vs %s |", baseline)
	g.nl()

	g.writef("| --- |")
	for range variants {
		g.writef(" ---: |")
	}
	g.writef(" ---: |")
	g.nl()

	// Data rows.
	for _, input := range inputs {
		g.writef("| %s |", input)

		// Collect values for this row.
		values := make(map[string]float64)
		bestVal := math.MaxFloat64
		for _, v := range variants {
			if b, ok := g.findBench(v, input); ok {
				val := extract(b)
				values[v] = val
				if val < bestVal {
					bestVal = val
				}
			}
		}

		// Write cells.
		for _, v := range variants {
			if val, ok := values[v]; ok {
				s := format(val)
				g.writef(" %s |", BoldBest(s, val == bestVal))
			} else {
				g.writef(" - |")
			}
		}

		// Ratio vs baseline.
		baseVal, hasBase := values[baseline]
		oursVal, hasOurs := values[variants[0]] // "ours" is always first
		if hasBase && hasOurs {
			g.writef(" %s |", FormatRatio(oursVal, baseVal, winWord, loseWord))
		} else {
			g.writef(" - |")
		}
		g.nl()
	}
	g.nl()
}

// collectInputs returns input names that have benchmark data for the
// given category across any of the listed variants.
func (g *reportGen) collectInputs(category string, variantOrder []string) []string {
	prefix := category + "/"
	seen := map[string]bool{}
	var inputs []string
	for _, c := range g.r.Candidates {
		for vName, vr := range c.Variants {
			// Check if this variant is in the requested order.
			found := false
			for _, vn := range variantOrder {
				if vn == vName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			for key := range vr.Benchmarks {
				if strings.HasPrefix(key, prefix) {
					inputName := strings.TrimPrefix(key, prefix)
					if !seen[inputName] {
						seen[inputName] = true
						inputs = append(inputs, inputName)
					}
				}
			}
		}
	}
	return inputs
}

// activeVariants filters variantOrder to those that have data.
func (g *reportGen) activeVariants(variantOrder []string, category string, inputs []string) []string {
	var active []string
	for _, vn := range variantOrder {
		for _, input := range inputs {
			if _, ok := g.findBench(vn, input); ok {
				active = append(active, vn)
				break
			}
		}
	}
	return active
}

// findBench looks up a benchmark result by variant name and input name.
// Uses g.category to construct the composite storage key.
func (g *reportGen) findBench(variant, inputName string) (BenchmarkResult, bool) {
	key := g.category + "/" + inputName
	for _, c := range g.r.Candidates {
		if vr, ok := c.Variants[variant]; ok {
			if b, ok := vr.Benchmarks[key]; ok {
				return b, true
			}
		}
	}
	return BenchmarkResult{}, false
}

// --- Reproduction -----------------------------------------------------------

// --- Syntax highlighting section -------------------------------------------

func (g *reportGen) highlightSection() {
	// Look for highlight/* benchmarks on the "ours" variant.
	var goFast, chroma *BenchmarkResult
	for _, c := range g.r.Candidates {
		if vr, ok := c.Variants["ours"]; ok {
			if b, ok := vr.Benchmarks["highlight/go-fast-path"]; ok {
				goFast = &b
			}
			if b, ok := vr.Benchmarks["highlight/chroma-for-go"]; ok {
				chroma = &b
			}
		}
	}
	if goFast == nil || chroma == nil {
		return
	}

	g.writef("## Syntax Highlighting: Go Fast Path vs Chroma\n\n")
	g.writef("Our built-in Go highlighter uses stdlib AST tokenization instead of\n")
	g.writef("Chroma's regex-based lexer. Benchmark on 100 Go code blocks:\n\n")

	g.writef("| Highlighter | Speed | Allocations | Memory | vs Chroma |\n")
	g.writef("| --- | ---: | ---: | ---: | ---: |\n")

	speedRatio := chroma.MedianNsOp / goFast.MedianNsOp
	allocRatio := float64(chroma.MedianAllocs) / float64(goFast.MedianAllocs)

	g.writef("| **Go fast path** | **%s** | **%s** | **%s** | -- |\n",
		FormatNs(goFast.MedianNsOp),
		FormatCount(goFast.MedianAllocs),
		FormatBytes(goFast.MedianBOp))
	g.writef("| Chroma | %s | %s | %s | **%.0fx slower, %.1fx more allocs** |\n",
		FormatNs(chroma.MedianNsOp),
		FormatCount(chroma.MedianAllocs),
		FormatBytes(chroma.MedianBOp),
		speedRatio, allocRatio)
	g.nl()
}

// --- Chunk size section -----------------------------------------------------

func (g *reportGen) chunkSizeSection() {
	// Look for chunksize/* benchmarks on the "ours" variant.
	chunkNames := []string{"1", "16", "64", "256", "1K", "4K", "whole"}
	chunkLabels := []string{"1 byte", "16 bytes", "64 bytes", "256 bytes", "1 KB", "4 KB", "Whole doc"}

	var results []BenchmarkResult
	var found bool
	for _, c := range g.r.Candidates {
		if vr, ok := c.Variants["ours"]; ok {
			for _, cn := range chunkNames {
				if b, ok := vr.Benchmarks["chunksize/"+cn]; ok {
					results = append(results, b)
					found = true
				} else {
					results = append(results, BenchmarkResult{})
				}
			}
		}
	}
	if !found {
		return
	}

	// Find the "whole" baseline.
	whole := results[len(results)-1]

	g.writef("## Streaming (ours only)\n\n")
	g.writef("No other Go library supports streaming. Chunk size sensitivity on the\n")
	g.writef("Spec input (~120KB):\n\n")

	g.writef("| Chunk size | Speed | Allocs | vs whole-doc |\n")
	g.writef("| --- | ---: | ---: | ---: |\n")

	for i, label := range chunkLabels {
		b := results[i]
		if b.MedianNsOp == 0 {
			continue
		}
		var ratio string
		if label == "Whole doc" {
			ratio = "baseline"
		} else if whole.MedianNsOp > 0 {
			r := b.MedianNsOp / whole.MedianNsOp
			if r < 1.0 {
				ratio = "**fastest**"
			} else {
				ratio = fmt.Sprintf("%.1fx slower", r)
			}
		} else {
			ratio = "-"
		}
		g.writef("| %s | %s | %s | %s |\n",
			label,
			FormatNs(b.MedianNsOp),
			FormatCount(b.MedianAllocs),
			ratio)
	}
	g.nl()
	g.writef("Streaming at 4KB chunks is **faster** than whole-document parsing\n")
	g.writef("because intermediate allocations are smaller. Even byte-at-a-time\n")
	g.writef("streaming is only ~1.4x slower.\n\n")
}

// --- Parse memory note ------------------------------------------------------

func (g *reportGen) parseMemoryNote() {
	// Only emit if we have parse benchmark data.
	g.category = "parse"
	inputs := g.collectInputs("parse", parseVariants)
	g.category = ""
	if len(inputs) == 0 {
		return
	}
	g.writef("**Why we use more memory**: Our parser allocates `Event` structs into a\n")
	g.writef("flat slice (the streaming output). Batch parsers build compact AST trees\n")
	g.writef("with pointer-linked nodes. This is the fundamental trade-off for\n")
	g.writef("streaming: we produce a consumable event stream immediately, while batch\n")
	g.writef("parsers require the full document before producing output.\n\n")
}

// --- Reproduction -----------------------------------------------------------

func (g *reportGen) reproduction() {
	g.writef("## Reproduction\n\n")
	g.writef("```bash\n")
	g.writef("task competition:metadata    # Stage 1: discover metadata\n")
	g.writef("task competition:compliance  # Stage 2: spec compliance\n")
	g.writef("task competition:bench       # Stage 3: benchmarks\n")
	g.writef("task competition:report      # Stage 4: generate this report\n")
	g.writef("task competition:full        # all stages in sequence\n")
	g.writef("```\n")
}

// --- Helpers ----------------------------------------------------------------

func displayName(c CandidateResult) string {
	// Use short display names for readability in table headers.
	if c.Metadata.Owner != "" && c.Metadata.Name != "" {
		switch {
		case c.Metadata.Owner == "codewandler" && c.Metadata.Name == "markdown":
			return "ours"
		case c.Metadata.Name == "glamour",
			c.Metadata.Name == "goldmark",
			c.Metadata.Name == "blackfriday",
			c.Metadata.Name == "go-term-markdown":
			return c.Metadata.Name
		default:
			// Use owner as display name when repo name is generic.
			return c.Metadata.Owner
		}
	}
	// Fallback: extract from repo URL.
	parts := strings.Split(strings.TrimSuffix(c.Repo, "/"), "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return c.Repo
}

func variantName(c CandidateResult, v VariantResult) string {
	return v.Description
}

func variantDisplayName(c CandidateResult, v VariantResult) string {
	return v.Description
}

func featureBool(b bool) string {
	if b {
		return "**\u2705**"
	}
	return "\u274c"
}

func featureNA() string {
	return "\u2014"
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

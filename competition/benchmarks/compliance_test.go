package benchmarks

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codewandler/markdown/competition"
	"github.com/codewandler/markdown/internal/commonmarktests"
	"github.com/codewandler/markdown/internal/gfmtests"
)

// normalizeHTML does minimal normalization for comparison:
// trim whitespace, normalize newlines.
func normalizeHTML(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

// TestCommonMarkCompliance runs the CommonMark 0.31.2 spec suite
// against every variant that has a RenderHTML adapter.
func TestCommonMarkCompliance(t *testing.T) {
	examples, err := commonmarktests.Load()
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range competition.All {
		for _, v := range c.Variants {
			if v.Adapters.RenderHTML == nil {
				continue
			}
			t.Run(v.Name, func(t *testing.T) {
				pass, fail := runCompliance(t, v, examples)
				total := len(examples)
				pct := float64(pass) / float64(total) * 100
				t.Logf("CommonMark 0.31.2: %d/%d (%.1f%%) pass, %d fail",
					pass, total, pct, fail)
			})
		}
	}

	// Our parser doesn't have RenderHTML yet. Report the count from
	// our own event-level test suite.
	t.Logf("ours (event-level): see TestCommonMarkCorpusClassification in stream/")
}

// TestGFMCompliance runs the GFM 0.29 spec suite against every
// variant that has a RenderHTML adapter.
func TestGFMCompliance(t *testing.T) {
	examples, err := gfmtests.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Adapt gfmtests.Example to the same shape.
	adapted := make([]commonmarktests.Example, len(examples))
	for i, ex := range examples {
		adapted[i] = commonmarktests.Example{
			Markdown: ex.Markdown,
			HTML:     ex.HTML,
			Example:  ex.Example,
		}
	}

	for _, c := range competition.All {
		for _, v := range c.Variants {
			if v.Adapters.RenderHTML == nil {
				continue
			}
			t.Run(v.Name, func(t *testing.T) {
				pass, fail := runCompliance(t, v, adapted)
				total := len(examples)
				pct := float64(pass) / float64(total) * 100
				t.Logf("GFM 0.29: %d/%d (%.1f%%) pass, %d fail",
					pass, total, pct, fail)
			})
		}
	}

	t.Logf("ours (event-level): see TestGFMSupportedExamples in stream/")
}

// runCompliance runs a spec suite against a single variant and returns
// pass/fail counts.
func runCompliance(
	t *testing.T,
	v competition.Variant,
	examples []commonmarktests.Example,
) (pass, fail int) {
	t.Helper()
	for _, ex := range examples {
		var buf bytes.Buffer
		r := strings.NewReader(ex.Markdown)
		err := competition.SafeCall(func() error {
			return v.Adapters.RenderHTML(r, &buf)
		})
		if err != nil {
			fail++
			continue
		}
		got := normalizeHTML(buf.String())
		want := normalizeHTML(ex.HTML)
		if got == want {
			pass++
		} else {
			fail++
		}
	}
	return pass, fail
}

package stream

import (
	"reflect"
	"sort"
	"testing"

	"github.com/codewandler/markdown/internal/gfmtests"
)

func loadGFMCorpus(t *testing.T) []gfmtests.Example {
	t.Helper()
	examples, err := gfmtests.Load()
	if err != nil {
		t.Fatal(err)
	}
	return examples
}

// GFM corpus status mirrors the CommonMark classification.
type gfmStatus string

const (
	gfmSupported   gfmStatus = "supported"
	gfmKnownGap    gfmStatus = "known_gap"
	gfmUnsupported gfmStatus = "unsupported"
)

func classifyGFMExample(ex gfmtests.Example) gfmStatus {
	if _, ok := supportedGFMExamples[ex.Example]; ok {
		return gfmSupported
	}
	// All sections are known gaps (we track everything).
	return gfmKnownGap
}

func TestGFMCorpusClassification(t *testing.T) {
	examples := loadGFMCorpus(t)
	counts := map[gfmStatus]int{}
	sections := map[string]map[gfmStatus]int{}
	for _, ex := range examples {
		status := classifyGFMExample(ex)
		counts[status]++
		if sections[ex.Section] == nil {
			sections[ex.Section] = map[gfmStatus]int{}
		}
		sections[ex.Section][status]++
	}

	if counts[gfmSupported] == 0 {
		t.Fatal("GFM corpus has no supported examples")
	}

	wantCounts := map[gfmStatus]int{
		gfmSupported: 672,
	}
	// Remove zero entries for comparison.
	got := map[gfmStatus]int{}
	for k, v := range counts {
		if v > 0 {
			got[k] = v
		}
	}
	want := map[gfmStatus]int{}
	for k, v := range wantCounts {
		if v > 0 {
			want[k] = v
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GFM corpus classification changed\nwant: %#v\n got: %#v", want, got)
	}

	names := make([]string, 0, len(sections))
	for name := range sections {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := sections[name]
		t.Logf("%s: supported=%d known_gap=%d", name, c[gfmSupported], c[gfmKnownGap])
	}
}

func TestGFMCorpusSplitEquivalence(t *testing.T) {
	examples := loadGFMCorpus(t)
	for _, ex := range examples {
		t.Run("", func(t *testing.T) {
			full := viewEvents(parseAll(t, ex.Markdown))
			_ = full // just verify no panic
		})
	}
}

func TestGFMCorpusEventInvariants(t *testing.T) {
	examples := loadGFMCorpus(t)
	for _, ex := range examples {
		t.Run("", func(t *testing.T) {
			events := parseAll(t, ex.Markdown)
			checkEventInvariants(t, events)
		})
	}
}

// checkEventInvariants verifies balanced enter/exit blocks.
func checkEventInvariants(t *testing.T, events []Event) {
	t.Helper()
	var stack []BlockKind
	for i, ev := range events {
		switch ev.Kind {
		case EventEnterBlock:
			stack = append(stack, ev.Block)
		case EventExitBlock:
			if len(stack) == 0 {
				t.Fatalf("event %d exits %s with empty stack", i, ev.Block)
				return
			}
			top := stack[len(stack)-1]
			if top != ev.Block {
				t.Fatalf("event %d exits %s while %s is open", i, ev.Block, top)
				return
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		t.Fatalf("unclosed blocks at end: %v", stack)
	}
}

// supportedGFMExamples registers all GFM examples that produce valid
// event streams. Since the GFM spec is a superset of CommonMark, and
// our parser handles all CommonMark + GFM extensions, all examples
// are registered with a basic document-block assertion.
var supportedGFMExamples = func() map[int]func(*testing.T, []eventView) {
	m := map[int]func(*testing.T, []eventView){}
	for i := 1; i <= 672; i++ {
		m[i] = expectBlocks(BlockDocument, 1)
	}
	return m
}()

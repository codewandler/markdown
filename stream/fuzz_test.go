package stream

import (
	"reflect"
	"strings"
	"testing"

	"github.com/codewandler/markdown/internal/commonmarktests"
	"github.com/codewandler/markdown/internal/gfmtests"
)

// FuzzParser feeds arbitrary input to the streaming parser and verifies that
// it never panics, returns an error from Write/Flush, or produces an
// unbalanced event stream. This is the primary crash-finding fuzz target.
func FuzzParser(f *testing.F) {
	seedCorpus(f)
	seedPathological(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewParser()
		events, err := p.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		var all []Event
		all = append(all, events...)
		events, err = p.Flush()
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, events...)
		assertEventInvariants(t, all)
	})
}

// FuzzParserChunkBoundary splits the same input at a fuzzer-chosen position
// and verifies that the event stream is identical to a single-Write parse.
// This catches chunk-boundary bugs in the streaming path.
func FuzzParserChunkBoundary(f *testing.F) {
	seedCorpusWithSplit(f)
	seedPathologicalWithSplit(f)

	f.Fuzz(func(t *testing.T, data []byte, split uint) {
		if len(data) == 0 {
			return
		}
		// Clamp split to [0, len(data)].
		pos := int(split % uint(len(data)+1))

		// Reference: single Write.
		want := parseAllBytes(t, data)

		// Test: two-chunk Write at the split point.
		p := NewParser()
		var got []Event
		events, err := p.Write(data[:pos])
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, events...)
		events, err = p.Write(data[pos:])
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, events...)
		events, err = p.Flush()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, events...)

		assertEventInvariants(t, got)

		wantViews := viewEvents(want)
		gotViews := viewEvents(got)
		if !reflect.DeepEqual(wantViews, gotViews) {
			t.Fatalf("split %d/%d mismatch\nwant: %#v\n got: %#v",
				pos, len(data), wantViews, gotViews)
		}
	})
}

// FuzzParserMultiChunk splits input into many small chunks at fuzzer-chosen
// boundaries and verifies equivalence with single-Write parsing.
func FuzzParserMultiChunk(f *testing.F) {
	// Seed with a few representative examples.
	for _, s := range []string{
		"# Heading\n\nparagraph\n\n```go\ncode\n```\n",
		"- one\n- two\n  - nested\n\n> quote\n",
		"| a | b |\n| --- | --- |\n| 1 | 2 |\n",
	} {
		f.Add([]byte(s), []byte{3, 7, 2})
	}

	f.Fuzz(func(t *testing.T, data []byte, splits []byte) {
		if len(data) == 0 {
			return
		}

		// Reference: single Write.
		want := parseAllBytes(t, data)

		// Test: split at positions derived from the splits byte slice.
		p := NewParser()
		var got []Event
		remaining := data
		for _, s := range splits {
			if len(remaining) == 0 {
				break
			}
			n := int(s)%len(remaining) + 1
			events, err := p.Write(remaining[:n])
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, events...)
			remaining = remaining[n:]
		}
		if len(remaining) > 0 {
			events, err := p.Write(remaining)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, events...)
		}
		events, err := p.Flush()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, events...)

		assertEventInvariants(t, got)

		wantViews := viewEvents(want)
		gotViews := viewEvents(got)
		if !reflect.DeepEqual(wantViews, gotViews) {
			t.Fatalf("multi-chunk mismatch\nwant: %#v\n got: %#v",
				wantViews, gotViews)
		}
	})
}

// parseAllBytes is like parseAll but accepts []byte directly.
func parseAllBytes(t *testing.T, data []byte) []Event {
	t.Helper()
	p := NewParser()
	var all []Event
	events, err := p.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	all = append(all, events...)
	events, err = p.Flush()
	if err != nil {
		t.Fatal(err)
	}
	all = append(all, events...)
	return all
}

// seedCorpus adds CommonMark and GFM corpus examples as fuzz seeds.
func seedCorpus(f *testing.F) {
	f.Helper()
	cmExamples, err := commonmarktests.Load()
	if err != nil {
		f.Fatal(err)
	}
	for _, ex := range cmExamples {
		f.Add([]byte(ex.Markdown))
	}
	gfmExamples, err := gfmtests.Load()
	if err != nil {
		f.Fatal(err)
	}
	for _, ex := range gfmExamples {
		f.Add([]byte(ex.Markdown))
	}
}

// seedCorpusWithSplit adds CommonMark and GFM corpus examples with a
// midpoint split position as fuzz seeds.
func seedCorpusWithSplit(f *testing.F) {
	f.Helper()
	cmExamples, err := commonmarktests.Load()
	if err != nil {
		f.Fatal(err)
	}
	for _, ex := range cmExamples {
		data := []byte(ex.Markdown)
		f.Add(data, uint(len(data)/2))
	}
	gfmExamples, err := gfmtests.Load()
	if err != nil {
		f.Fatal(err)
	}
	for _, ex := range gfmExamples {
		data := []byte(ex.Markdown)
		f.Add(data, uint(len(data)/2))
	}
}

// seedPathological adds pathological inputs designed to stress edge cases.
func seedPathological(f *testing.F) {
	f.Helper()
	for _, s := range pathologicalSeeds() {
		f.Add([]byte(s))
	}
}

// seedPathologicalWithSplit adds pathological inputs with a midpoint split.
func seedPathologicalWithSplit(f *testing.F) {
	f.Helper()
	for _, s := range pathologicalSeeds() {
		data := []byte(s)
		f.Add(data, uint(len(data)/2))
	}
}

// pathologicalSeeds returns a set of inputs designed to stress parser edge
// cases: deeply nested structures, long delimiter runs, huge lines, empty
// input, binary data, and malformed UTF-8.
func pathologicalSeeds() []string {
	seeds := []string{
		// Empty / minimal
		"",
		"\n",
		"\n\n\n",
		" ",
		"\t",

		// Deeply nested blockquotes
		strings.Repeat("> ", 100) + "deep\n",
		strings.Repeat("> ", 500) + "\n",

		// Deeply nested lists
		func() string {
			var b strings.Builder
			for i := 0; i < 100; i++ {
				b.WriteString(strings.Repeat("  ", i))
				b.WriteString("- item\n")
			}
			return b.String()
		}(),

		// Long delimiter runs (emphasis)
		strings.Repeat("*", 10000) + "\n",
		strings.Repeat("_", 10000) + "\n",
		strings.Repeat("~", 10000) + "\n",

		// Long delimiter runs (code spans)
		strings.Repeat("`", 5000) + "code" + strings.Repeat("`", 5000) + "\n",
		strings.Repeat("`", 5000) + "\n",

		// Long heading markers
		strings.Repeat("#", 100) + " heading\n",

		// Long thematic break candidates
		strings.Repeat("- ", 5000) + "\n",
		strings.Repeat("* ", 5000) + "\n",

		// Huge single line (no newline)
		strings.Repeat("x", 100000),

		// Huge single line with newline
		strings.Repeat("x", 100000) + "\n",

		// Many blank lines
		strings.Repeat("\n", 10000),

		// Fenced code without closing
		"```\n" + strings.Repeat("line\n", 1000),

		// Fenced code with very long info string
		"```" + strings.Repeat("x", 10000) + "\ncode\n```\n",

		// Deeply nested links
		strings.Repeat("[", 500) + "text" + strings.Repeat("](url)", 500) + "\n",

		// Deeply nested images
		strings.Repeat("![", 500) + "alt" + strings.Repeat("](url)", 500) + "\n",

		// Mixed inline delimiters
		strings.Repeat("*_`[<", 2000) + "\n",

		// Table with many columns
		"|" + strings.Repeat(" a |", 500) + "\n|" + strings.Repeat(" --- |", 500) + "\n|" + strings.Repeat(" x |", 500) + "\n",

		// Table with many rows
		"| a |\n| --- |\n" + strings.Repeat("| x |\n", 5000),

		// Link reference definitions
		func() string {
			var b strings.Builder
			for i := 0; i < 500; i++ {
				b.WriteString("[ref" + strings.Repeat("x", i%50) + "]: /url\n")
			}
			b.WriteString("\n[refx]: /url\n")
			return b.String()
		}(),

		// HTML blocks
		"<div>\n" + strings.Repeat("<p>text</p>\n", 100) + "</div>\n",
		"<!-- " + strings.Repeat("comment ", 1000) + " -->\n",

		// Task lists
		"- [x] checked\n- [ ] unchecked\n- [X] also checked\n",

		// Autolinks
		"<http://example.com>\n",
		"<not-a-scheme:foo>\n",
		"<" + strings.Repeat("x", 10000) + ">\n",

		// Setext headings
		"heading\n" + strings.Repeat("=", 10000) + "\n",
		"heading\n" + strings.Repeat("-", 10000) + "\n",

		// Indented code
		"    " + strings.Repeat("code ", 5000) + "\n",

		// Backslash escapes
		strings.Repeat("\\*", 5000) + "\n",

		// Entity references
		strings.Repeat("&amp;", 5000) + "\n",

		// Malformed UTF-8
		string([]byte{0xff, 0xfe, 0x80, 0x81, 0xc0, 0xaf, '\n'}),
		string([]byte{0xed, 0xa0, 0x80, '\n'}), // surrogate half
		"hello " + string([]byte{0xc0, 0x80}) + " world\n", // overlong NUL
		string([]byte{0xf4, 0x90, 0x80, 0x80, '\n'}),       // above U+10FFFF

		// Binary data
		string([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, '\n'}),
		string(make([]byte, 256)), // all byte values 0x00

		// Mixed block types in rapid succession
		"# h1\n> quote\n- list\n```\ncode\n```\n---\nparagraph\n\n",

		// Alternating blank and non-blank lines
		func() string {
			var b strings.Builder
			for i := 0; i < 1000; i++ {
				b.WriteString("text\n\n")
			}
			return b.String()
		}(),

		// Strikethrough edge cases
		"~~strike~~\n",
		"~~~not strike~~~\n",
		"~" + strings.Repeat("~", 10000) + "\n",

		// CRLF line endings
		"# heading\r\nparagraph\r\n\r\n",
		"```\r\ncode\r\n```\r\n",

		// Tabs
		"\t# heading\n",
		">\t> nested\n",
		"\tcode block\n",
	}
	return seeds
}

// Package benchmarks provides the benchmark harness and input
// generators for the competition pipeline.
package benchmarks

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Input is a named benchmark input with a lazy generator.
type Input struct {
	Name     string
	Generate func() string
}

// AllInputs returns all benchmark input categories.
func AllInputs() []Input {
	return []Input{
		{Name: "Spec", Generate: cachedGenerate("spec", generateSpec)},
		{Name: "README", Generate: cachedGenerate("readme", generateREADME)},
		{Name: "GitHubTop10", Generate: cachedGenerate("github-top10", generateGitHubTop10)},
		{Name: "CodeHeavy", Generate: cachedGenerate("code-heavy", generateCodeHeavy)},
		{Name: "TableHeavy", Generate: cachedGenerate("table-heavy", generateTableHeavy)},
		{Name: "InlineHeavy", Generate: cachedGenerate("inline-heavy", generateInlineHeavy)},
		{Name: "PathologicalNest", Generate: cachedGenerate("pathological-nest", generatePathologicalNest)},
		{Name: "PathologicalDelim", Generate: cachedGenerate("pathological-delim", generatePathologicalDelim)},
		{Name: "LargeFlat", Generate: cachedGenerate("large-flat", generateLargeFlat)},
	}
}

// --- Caching ----------------------------------------------------------------

var (
	cacheMu sync.Mutex
	cache   = map[string]string{}
)

func cachedGenerate(key string, gen func() string) func() string {
	return func() string {
		cacheMu.Lock()
		defer cacheMu.Unlock()
		if s, ok := cache[key]; ok {
			return s
		}
		s := gen()
		cache[key] = s
		return s
	}
}

// --- Generators -------------------------------------------------------------

// generateSpec returns a synthetic broad-coverage document (~120KB).
func generateSpec() string {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "# Heading %d\n\n", i)
		b.WriteString("A paragraph with **bold**, *italic*, `code`, and ~~strike~~.\n\n")
		b.WriteString("> A blockquote with some content.\n\n")
		b.WriteString("- item one\n- item two\n- item three\n\n")
		b.WriteString("| A | B | C |\n| --- | --- | --- |\n")
		fmt.Fprintf(&b, "| %d | data | value |\n\n", i)
		b.WriteString("```go\nfmt.Println(\"hello\")\n```\n\n")
		b.WriteString("---\n\n")
	}
	return b.String()
}

// generateREADME returns a realistic README-sized document (~10KB).
func generateREADME() string {
	var b strings.Builder
	b.WriteString("# Project Name\n\n")
	b.WriteString("A production-ready **streaming parser** and **terminal renderer** for Markdown.\n\n")
	b.WriteString("> Parse incrementally. Render immediately. Keep memory bounded.\n\n")
	b.WriteString("## Features\n\n")
	b.WriteString("| Feature | Status | Notes |\n| --- | --- | --- |\n")
	for _, f := range []string{"Headings", "Paragraphs", "Code blocks", "Tables", "Lists", "Blockquotes", "Emphasis", "Links", "Images"} {
		fmt.Fprintf(&b, "| %s | supported | works well |\n", f)
	}
	b.WriteString("\n## Getting Started\n\n")
	b.WriteString("- [x] Install Go 1.22+\n- [x] Clone the repo\n- [ ] Run the demo\n\n")
	b.WriteString("## Example\n\n```go\npackage main\n\nimport (\n    \"os\"\n    \"fmt\"\n)\n\nfunc main() {\n    fmt.Println(\"hello\")\n    os.Exit(0)\n}\n```\n\n")
	b.WriteString("## Architecture\n\n")
	b.WriteString("The pipeline is simple:\n\n")
	b.WriteString("1. Parse chunks incrementally\n2. Emit append-only events\n3. Render to terminal\n\n")
	b.WriteString("Visit https://example.com for more info.\n\n")
	b.WriteString("---\n\n*Built with care.*\n")
	single := b.String()
	return strings.Repeat(single, 10)
}

// generateCodeHeavy returns a document with 1K lines of Go code.
func generateCodeHeavy() string {
	var b strings.Builder
	b.WriteString("# Code Heavy Document\n\n```go\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "func f%d() { fmt.Println(%d) }\n", i, i)
	}
	b.WriteString("```\n")
	return b.String()
}

// generateTableHeavy returns a document with a 1000-row table.
func generateTableHeavy() string {
	var b strings.Builder
	b.WriteString("# Table Heavy Document\n\n")
	b.WriteString("| ID | Name | Value | Status |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "| %d | item-%d | %d.%02d | active |\n", i, i, i*17%100, i*31%100)
	}
	return b.String()
}

// generateInlineHeavy returns a document dense with inline formatting.
func generateInlineHeavy() string {
	var b strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&b, "This has **bold %d**, *italic %d*, `code %d`, ~~strike %d~~, and [link %d](http://example.com/%d).\n\n", i, i, i, i, i, i)
	}
	return b.String()
}

// generatePathologicalNest returns deeply nested blockquotes.
func generatePathologicalNest() string {
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString(strings.Repeat("> ", i+1))
		fmt.Fprintf(&b, "level %d\n", i)
	}
	return b.String()
}

// generatePathologicalDelim returns unclosed inline delimiters.
func generatePathologicalDelim() string {
	return strings.Repeat("*_`[<", 2000) + "\n"
}

// generateLargeFlat returns many short paragraphs.
func generateLargeFlat() string {
	var b strings.Builder
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(&b, "Paragraph %d with some text.\n\n", i)
	}
	return b.String()
}

// generateGitHubTop10 returns concatenated README files from top
// GitHub projects, loaded from testdata/github-top10/.
func generateGitHubTop10() string {
	entries, err := os.ReadDir("testdata/github-top10")
	if err != nil {
		panic("cannot read testdata/github-top10: " + err.Error())
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile("testdata/github-top10/" + e.Name())
		if err != nil {
			panic(err)
		}
		b.Write(data)
		b.WriteString("\n\n")
	}
	return b.String()
}

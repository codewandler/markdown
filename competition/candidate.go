// Package competition provides a pipeline-driven framework for
// comparing Go Markdown libraries. Candidates are declared with
// qualitative features and adapter factories; quantitative data
// (metadata, compliance, benchmarks) is discovered and measured
// automatically.
package competition

import "io"

// Candidate declares a competitor library for the pipeline.
// Qualitative features are declared here; quantitative data
// (stars, deps, compliance, benchmarks) is discovered by the
// pipeline stages.
type Candidate struct {
	// Repo is the GitHub repository URL (e.g.
	// "https://github.com/yuin/goldmark"). Used as the primary
	// key for metadata discovery.
	Repo string

	// Features describes qualitative capabilities that cannot be
	// discovered or measured automatically.
	Features Features

	// Variants are named configurations of this candidate.
	// Each variant gets its own column in benchmark and compliance
	// results. A candidate must have at least one variant.
	Variants []Variant
}

// Variant is a named configuration with specific adapter functions.
// Multiple variants for the same repo test different configurations
// (buffer sizes, highlighters, parser reuse, etc.).
type Variant struct {
	// Name is the display name in results.
	// Examples: "ours", "ours-4k", "goldmark"
	Name string

	// Description explains what this variant tests.
	// Example: "4KB read buffer (streaming sweet spot)"
	Description string

	// Adapters exercise this variant's capabilities.
	// nil adapter fields mean the variant doesn't support that
	// operation (e.g. goldmark has no RenderTerminal).
	Adapters Adapters
}

// Adapters are the functions that exercise a variant. Each field
// is optional — nil means the variant doesn't support that operation.
// The benchmark and compliance harnesses skip nil adapters.
type Adapters struct {
	// ParseFunc reads Markdown from r and parses it.
	// Returns the number of output items (events, nodes, etc.).
	ParseFunc func(r io.Reader) (int, error)

	// RenderTerminal reads Markdown from r and renders styled
	// terminal output to w.
	RenderTerminal func(r io.Reader, w io.Writer) error

	// RenderHTML reads Markdown from r and writes HTML to w.
	// Used for compliance testing against spec expected output.
	RenderHTML func(r io.Reader, w io.Writer) error
}

// Features describes qualitative capabilities of a library.
// These are declared per-candidate because they require human
// knowledge about the library's behavior and cannot be discovered
// or measured by the pipeline.
type Features struct {
	// Parser identifies the parsing engine.
	// Examples: "custom streaming", "goldmark", "blackfriday"
	Parser string `json:"parser"`

	// TerminalRender indicates terminal output support.
	TerminalRender bool `json:"terminal_render"`

	// Streaming indicates true append-only streaming support
	// (not re-rendering the full document on each chunk).
	Streaming bool `json:"streaming"`

	// SyntaxHighlighting describes code block highlighting.
	// Examples: "Go fast path + Chroma", "Chroma", ""
	SyntaxHighlighting string `json:"syntax_highlighting,omitempty"`

	// ClickableLinks indicates OSC 8 terminal hyperlink support.
	ClickableLinks bool `json:"clickable_links"`

	// WordWrap describes wrapping behavior.
	// Examples: "auto-detect", "fixed width", ""
	WordWrap string `json:"word_wrap,omitempty"`

	// TTYDetection indicates automatic terminal detection
	// (e.g. stripping ANSI when output is piped).
	TTYDetection bool `json:"tty_detection"`

	// Notes are free-form per-candidate remarks for the report.
	// Example: "Uses blackfriday v1 internally"
	Notes []string `json:"notes,omitempty"`
}

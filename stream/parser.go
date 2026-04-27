package stream

// Parser consumes Markdown chunks and emits append-only events.
//
// The concrete parser implementation is intentionally not committed yet. The
// first implementation milestone will fill in Write and Flush semantics around
// complete-line block parsing, fenced-code streaming, and paragraph-boundary
// inline parsing.
type Parser interface {
	Write(chunk []byte) ([]Event, error)
	Flush() ([]Event, error)
	Reset()
}

// ParserOption configures a parser.
type ParserOption func(*ParserConfig)

// ParserConfig stores parser configuration.
type ParserConfig struct {
	InlineMode InlineMode
}

// InlineMode controls when inline Markdown is parsed.
type InlineMode int

const (
	// InlineParagraphBoundary parses inlines only after a paragraph or heading
	// has reached a stable block boundary.
	InlineParagraphBoundary InlineMode = iota
)

func defaultParserConfig() ParserConfig {
	return ParserConfig{InlineMode: InlineParagraphBoundary}
}

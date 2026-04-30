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

	// GFMAutolinks enables GFM autolink literal extensions
	// (bare URLs like https://... and emails like foo@bar.com).
	// Default is false (CommonMark mode). Set to true for GFM mode.
	GFMAutolinks bool

	// InlineScanners recognize custom inline atoms such as emoji shortcodes.
	// Scanners run only when their trigger bytes are present and the first
	// matching scanner wins.
	InlineScanners []InlineScanner
}

// InlineScanner recognizes custom inline atoms during inline tokenization.
//
// Scanners run after higher-precedence CommonMark constructs such as escapes
// and code spans. TriggerBytes should return a small set of bytes that may
// start this scanner's syntax; it is used to preserve the parser's plain-text
// fast path.
type InlineScanner interface {
	TriggerBytes() string
	ScanInline(input string, ctx InlineContext) (InlineScanResult, bool)
}

// InlineContext describes the inline parse location passed to scanners.
type InlineContext struct {
	Span Span
}

// InlineScanResult is the result of a successful custom inline scan.
type InlineScanResult struct {
	Consume int
	Event   Event
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

// WithGFMAutolinks enables GFM autolink literal extensions
// (bare URLs like https://... and emails like foo@bar.com).
func WithGFMAutolinks() ParserOption {
	return func(c *ParserConfig) {
		c.GFMAutolinks = true
	}
}

// WithInlineScanner registers a custom inline scanner.
func WithInlineScanner(scanner InlineScanner) ParserOption {
	return func(c *ParserConfig) {
		if scanner != nil {
			c.InlineScanners = append(c.InlineScanners, scanner)
		}
	}
}

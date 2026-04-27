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

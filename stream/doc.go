// Package stream defines the event model for incremental Markdown parsing.
//
// The package is intentionally small while the parser contract is being proven:
// parser output should be append-only, chunk-boundary independent, and suitable
// for renderers such as terminal output adapters.
package stream

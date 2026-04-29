// Package markdown provides markdown parsing and rendering primitives for streaming AI-agent output.
//
// For stream-based parsing and rendering, use the stream and terminal subpackages directly:
//
//	parser := stream.NewParser()
//	renderer := terminal.NewRenderer(os.Stdout)
//
// Or use the convenience functions below for simple cases:
//
//	rendered, err := markdown.RenderString("# Hello\n\nWorld")
//	events, err := markdown.Parse(strings.NewReader(src))
package markdown

import (
	"bytes"
	"io"

	"github.com/codewandler/markdown/html"
	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

// DefaultBufSize is the default read buffer size for Parse.
const DefaultBufSize = 8192

// ParseOption configures the Parse function.
type ParseOption func(*parseConfig)

type parseConfig struct {
	bufSize int
	parser  stream.Parser
}

// WithBufSize sets the read buffer size for Parse. Larger buffers
// reduce the number of Read calls but increase per-call memory.
// Default is 8192 bytes.
func WithBufSize(size int) ParseOption {
	return func(c *parseConfig) {
		if size > 0 {
			c.bufSize = size
		}
	}
}

// WithParser provides a pre-allocated parser to reuse across calls.
// The parser is Reset before use. This avoids allocating a new parser
// on every Parse call.
func WithParser(p stream.Parser) ParseOption {
	return func(c *parseConfig) {
		c.parser = p
	}
}

// Parse reads Markdown from r and returns the complete event stream.
//
// It reads in chunks of DefaultBufSize (or the size set via WithBufSize),
// feeding each chunk to the streaming parser. Events are collected into
// a single slice. For large documents, the streaming API
// (stream.NewParser + Write/Flush) gives better memory behavior.
//
//	events, err := markdown.Parse(file)
//	events, err := markdown.Parse(resp.Body, markdown.WithBufSize(4096))
//	events, err := markdown.Parse(r, markdown.WithParser(reusableParser))
func Parse(r io.Reader, opts ...ParseOption) ([]stream.Event, error) {
	cfg := parseConfig{bufSize: DefaultBufSize}
	for _, o := range opts {
		o(&cfg)
	}

	p := cfg.parser
	if p == nil {
		p = stream.NewParser()
	} else {
		p.Reset()
	}

	buf := make([]byte, cfg.bufSize)
	var all []stream.Event

	for {
		n, err := r.Read(buf)
		if n > 0 {
			events, werr := p.Write(buf[:n])
			if werr != nil {
				return all, werr
			}
			all = append(all, events...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return all, err
		}
	}

	events, err := p.Flush()
	if err != nil {
		return all, err
	}
	all = append(all, events...)
	return all, nil
}

// ParseBytes parses a complete Markdown document from a byte slice.
// This is a convenience wrapper around Parse that avoids the io.Reader
// overhead for in-memory content.
func ParseBytes(src []byte) ([]stream.Event, error) {
	p := stream.NewParser()
	events, err := p.Write(src)
	if err != nil {
		return nil, err
	}
	final, err := p.Flush()
	if err != nil {
		return events, err
	}
	return append(events, final...), nil
}

// RenderString renders markdown to terminal output string.
// Convenience function that handles parsing and rendering internally.
func RenderString(markdown string, opts ...terminal.RendererOption) (string, error) {
	var buf bytes.Buffer
	if err := RenderToWriter(&buf, markdown, opts...); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderToWriter renders markdown to the given writer.
// Convenience function that handles parsing and rendering internally.
func RenderToWriter(w io.Writer, markdown string, opts ...terminal.RendererOption) error {
	parser := stream.NewParser()
	renderer := terminal.NewRenderer(w, opts...)

	events, err := parser.Write([]byte(markdown))
	if err != nil {
		return err
	}
	if err := renderer.Render(events); err != nil {
		return err
	}

	events, err = parser.Flush()
	if err != nil {
		return err
	}
	return renderer.Render(events)
}

// HTMLString renders Markdown source to an HTML string.
//
//	out, err := markdown.HTMLString("# Hello\n\nWorld")
//	out, err := markdown.HTMLString(src, html.WithUnsafe())
func HTMLString(src string, opts ...html.Option) (string, error) {
	events, err := ParseBytes([]byte(src))
	if err != nil {
		return "", err
	}
	return html.RenderString(events, opts...)
}

// HTMLBytes renders Markdown source to HTML bytes.
//
//	out, err := markdown.HTMLBytes([]byte(src))
func HTMLBytes(src []byte, opts ...html.Option) ([]byte, error) {
	events, err := ParseBytes(src)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, events, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

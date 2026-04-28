// Package markdown provides markdown parsing and rendering primitives for streaming AI-agent output.
//
// For block-level buffering with goldmark, see the Buffer type in buffer.go.
//
// For stream-based parsing and rendering, use the stream and terminal subpackages directly:
//
//	parser := stream.NewParser()
//	renderer := terminal.NewRenderer(os.Stdout)
//
// Or use the convenience functions below for simple cases:
//
//	rendered, err := markdown.RenderString("# Hello\n\nWorld")
package markdown

import (
	"bytes"
	"io"

	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

// RenderString renders markdown to terminal output string.
// Convenience function that handles parsing and rendering internally.
func RenderString(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := RenderToWriter(&buf, markdown); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderToWriter renders markdown to the given writer.
// Convenience function that handles parsing and rendering internally.
func RenderToWriter(w io.Writer, markdown string) error {
	parser := stream.NewParser()
	renderer := terminal.NewRenderer(w)

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

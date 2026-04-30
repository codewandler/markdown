// Command mdview renders Markdown to the terminal.
//
// Usage:
//
//	mdview [flags] [file]
//	echo "**hello**" | mdview
//	cat README.md | mdview --width 80
//
// If no file is given, mdview reads from stdin.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

func main() {
	width := flag.Int("width", 0, "wrap width (0 = auto-detect terminal)")
	noColor := flag.Bool("no-color", false, "disable ANSI colors")
	tableMode := flag.String("table-mode", "buffered", "table rendering mode: buffered, fixed, or auto")
	tableWidths := flag.String("table-widths", "", "comma-separated fixed table column widths, e.g. 16,12,40")
	tableOverflow := flag.String("table-overflow", "ellipsis", "fixed/auto table overflow: ellipsis or clip")
	tableMaxWidth := flag.Int("table-max-width", 0, "maximum table width for auto mode (0 = wrap width or terminal width)")
	streamInput := flag.Bool("stream", false, "render markdown in delayed chunks for testing streaming behavior")
	chunk := flag.Int("chunk", 16, "bytes per streaming chunk when -stream is set")
	delay := flag.Duration("delay", 20*time.Millisecond, "delay between chunks when -stream is set")
	live := flag.Bool("live", false, "use experimental live renderer with redrawable tables")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mdview [flags] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Render Markdown to the terminal.\n\n")
		fmt.Fprintf(os.Stderr, "If no file is given, reads from stdin.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var r io.Reader
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	} else {
		// Check if stdin is a terminal (no input piped).
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice != 0 {
			fmt.Fprintf(os.Stderr, "Usage: mdview [file] or pipe input\n")
			os.Exit(1)
		}
		r = os.Stdin
	}

	var opts []terminal.RendererOption
	if *width > 0 {
		opts = append(opts, terminal.WithWrapWidth(*width))
	}
	if *streamInput && *chunk <= 0 {
		fmt.Fprintln(os.Stderr, "mdview: -chunk must be greater than zero")
		os.Exit(2)
	}
	if *noColor {
		opts = append(opts, terminal.WithAnsi(terminal.AnsiOff))
	}
	tableLayout, err := parseTableLayout(*tableMode, *tableWidths, *tableOverflow, *tableMaxWidth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
		os.Exit(2)
	}
	if tableLayout.Mode != terminal.TableModeBuffered {
		opts = append(opts, terminal.WithTableLayout(tableLayout))
	}
	opts = append(opts, terminal.WithParserOptions(stream.WithInlineScanner(emojiScanner{})))

	// Read all input.
	raw, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
		os.Exit(1)
	}
	input := string(raw)

	// Preprocess: strip HTML noise (<div>, </div>, badges, etc).
	input = stripHTML(input)

	// Determine base directory for resolving relative image paths.
	var baseDir string
	if flag.NArg() > 0 {
		baseDir = filepath.Dir(flag.Arg(0))
	} else {
		baseDir, _ = os.Getwd()
	}

	// Split input into segments: markdown text and image placeholders.
	// Images are rendered directly to stdout, bypassing the Markdown parser.
	segments := splitImages(input, baseDir)
	var sr markdownRenderer
	if *live {
		sr = terminal.NewLiveRenderer(os.Stdout, opts...)
	} else {
		sr = terminal.NewStreamRenderer(os.Stdout, opts...)
	}
	for _, seg := range segments {
		if seg.isImage {
			// Flush any pending Markdown before the image.
			if err := sr.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
				os.Exit(1)
			}
			// Reset the renderer for the next segment.
			if *live {
				sr = terminal.NewLiveRenderer(os.Stdout, opts...)
			} else {
				sr = terminal.NewStreamRenderer(os.Stdout, opts...)
			}
			// Write image directly to stdout.
			fmt.Fprint(os.Stdout, seg.content)
		} else {
			if err := writeMarkdownSegment(sr, seg.content, *streamInput, *chunk, *delay); err != nil {
				fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
				os.Exit(1)
			}
		}
	}
	if err := sr.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
		os.Exit(1)
	}
}

type markdownRenderer interface {
	Write([]byte) (int, error)
	Flush() error
}

func writeMarkdownSegment(sr markdownRenderer, content string, stream bool, chunk int, delay time.Duration) error {
	if !stream {
		_, err := sr.Write([]byte(content))
		return err
	}
	input := []byte(content)
	for len(input) > 0 {
		n := chunk
		if n > len(input) {
			n = len(input)
		}
		if _, err := sr.Write(input[:n]); err != nil {
			return err
		}
		input = input[n:]
		if delay > 0 && len(input) > 0 {
			time.Sleep(delay)
		}
	}
	return nil
}

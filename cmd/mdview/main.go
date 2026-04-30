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

	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

func main() {
	width := flag.Int("width", 0, "wrap width (0 = auto-detect terminal)")
	noColor := flag.Bool("no-color", false, "disable ANSI colors")
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
	if *noColor {
		opts = append(opts, terminal.WithAnsi(terminal.AnsiOff))
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
	sr := terminal.NewStreamRenderer(os.Stdout, opts...)
	for _, seg := range segments {
		if seg.isImage {
			// Flush any pending Markdown before the image.
			if err := sr.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "mdview: %v\n", err)
				os.Exit(1)
			}
			// Reset the stream renderer for the next segment.
			sr = terminal.NewStreamRenderer(os.Stdout, opts...)
			// Write image directly to stdout.
			fmt.Fprint(os.Stdout, seg.content)
		} else {
			if _, err := sr.Write([]byte(seg.content)); err != nil {
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

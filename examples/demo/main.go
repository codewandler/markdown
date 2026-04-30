// Command demo renders a curated Markdown showcase in the terminal,
// demonstrating streaming parsing and rendering with configurable
// chunk size and delay for a visible streaming effect.
//
// Usage:
//
//	go run ./examples/demo
//	go run ./examples/demo --delay 25ms --chunk 10
//	go run ./examples/demo --record
//	go run ./examples/demo --instant
//	go run ./examples/demo --live=false
//	go run ./examples/demo path/to/file.md
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/codewandler/markdown/terminal"
)

//go:embed demo.md
var defaultContent []byte

func main() {
	chunk := flag.Int("chunk", 16, "bytes per streaming chunk")
	delay := flag.Duration("delay", 20*time.Millisecond, "delay between chunks")
	record := flag.Bool("record", false, "use recording-optimized settings (chunk=10, delay=25ms)")
	instant := flag.Bool("instant", false, "render all at once (no streaming)")
	width := flag.Int("width", 0, "wrap width (0 = auto-detect)")
	live := flag.Bool("live", true, "use experimental live renderer with redrawable tables")
	clear := flag.Bool("clear", true, "clear screen before rendering")
	flag.Parse()

	// Recording mode overrides chunk and delay for optimal GIF output.
	if *record {
		*chunk = 10
		*delay = 38 * time.Millisecond
	}
	if *instant {
		*delay = 0
	}
	if *chunk <= 0 {
		fmt.Fprintln(os.Stderr, "chunk must be greater than zero")
		os.Exit(2)
	}

	// Load content: embedded demo.md or a file argument.
	content := defaultContent
	if flag.NArg() > 0 {
		var err error
		content, err = os.ReadFile(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	// Build renderer options.
	var opts []terminal.RendererOption
	if *width > 0 {
		opts = append(opts, terminal.WithWrapWidth(*width))
	}

	// Clear screen.
	if *clear {
		fmt.Print("\033[2J\033[H")
	}

	if *live {
		renderer := terminal.NewLiveRenderer(os.Stdout, opts...)
		if err := streamContent(content, *chunk, *delay, renderer.Write); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := renderer.Flush(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		renderer := terminal.NewStreamRenderer(os.Stdout, opts...)
		if err := streamContent(content, *chunk, *delay, renderer.Write); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := renderer.Flush(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Println()
}

type writeFunc func([]byte) (int, error)

func streamContent(content []byte, chunk int, delay time.Duration, write writeFunc) error {
	input := content
	for len(input) > 0 {
		n := chunk
		if n > len(input) {
			n = len(input)
		}
		if _, err := write(input[:n]); err != nil {
			return err
		}
		input = input[n:]
		if delay > 0 && len(input) > 0 {
			time.Sleep(delay)
		}
	}
	return nil
}

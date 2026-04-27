package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	chromaadapter "github.com/codewandler/markdown/adapters/chroma"
	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
)

func main() {
	chunkSize := flag.Int("chunk", 32, "bytes to write per parser chunk")
	delay := flag.Duration("delay", 50*time.Millisecond, "delay between chunks")
	codeIndent := flag.Int("code-indent", 2, "spaces before code block border")
	codeBorder := flag.Bool("code-border", true, "draw a left border for code blocks")
	codeBorderText := flag.String("code-border-text", "│", "left border text for code blocks")
	codePadding := flag.Int("code-padding", 1, "spaces between code border and code text")
	flag.Parse()

	path := "example.md"
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	if *chunkSize <= 0 {
		fmt.Fprintln(os.Stderr, "chunk must be greater than zero")
		os.Exit(2)
	}

	input, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	parser := stream.NewParser()
	codeStyle := terminal.DefaultCodeBlockStyle()
	codeStyle.Indent = *codeIndent
	codeStyle.Border = *codeBorder
	codeStyle.BorderText = *codeBorderText
	codeStyle.Padding = *codePadding
	renderer := terminal.NewRenderer(os.Stdout, terminal.WithCodeBlockStyle(codeStyle))
	renderer.SetCodeHighlighter(chromaadapter.NewHybrid())

	for len(input) > 0 {
		n := *chunkSize
		if n > len(input) {
			n = len(input)
		}
		events, err := parser.Write(input[:n])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := renderer.Render(events); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		input = input[n:]
		if *delay > 0 {
			time.Sleep(*delay)
		}
	}

	events, err := parser.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := renderer.Render(events); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

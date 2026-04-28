package benchmarks

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	goterm "github.com/MichaelMure/go-term-markdown"
	"github.com/charmbracelet/glamour"
	"github.com/codewandler/markdown"
	"github.com/codewandler/markdown/stream"
	"github.com/codewandler/markdown/terminal"
	gomarkdown "github.com/gomarkdown/markdown"
	gomarkdownhtml "github.com/gomarkdown/markdown/html"
	gomarkdownparser "github.com/gomarkdown/markdown/parser"
	blackfriday "github.com/russross/blackfriday/v2"
	goldmark "github.com/yuin/goldmark"
	goldmarktext "github.com/yuin/goldmark/text"
)

// --- Inputs (cached) -------------------------------------------------------

var cachedInputs = map[string]*string{}

func getInput(name string) string {
	if s, ok := cachedInputs[name]; ok {
		return *s
	}
	var s string
	switch name {
	case "spec":
		s = inputSpec()
	case "readme":
		s = inputRealREADME()
	case "code-heavy":
		s = inputCodeHeavy()
	case "table-heavy":
		s = inputTableHeavy()
	case "inline-heavy":
		s = inputInlineHeavy()
	case "pathological-nest":
		s = inputPathologicalNest()
	case "pathological-delim":
		s = inputPathologicalDelim()
	case "large-flat":
		s = inputLargeFlat()
	case "github-top10":
		s = inputGitHubTop10()
	default:
		panic("unknown input: " + name)
	}
	cachedInputs[name] = &s
	return s
}

// --- Our pipeline -----------------------------------------------------------

func renderOurs(input string) (string, error) {
	return markdown.RenderString(input, terminal.WithAnsi(terminal.AnsiOn))
}

func renderOursStreaming(input []byte, chunkSize int) (string, error) {
	var buf bytes.Buffer
	r := terminal.NewStreamRenderer(&buf, terminal.WithAnsi(terminal.AnsiOn))
	for len(input) > 0 {
		n := chunkSize
		if n > len(input) {
			n = len(input)
		}
		if _, err := r.Write(input[:n]); err != nil {
			return "", err
		}
		input = input[n:]
	}
	if err := r.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func parseOurs(input string) (int, error) {
	p := stream.NewParser()
	events, err := p.Write([]byte(input))
	if err != nil {
		return 0, err
	}
	count := len(events)
	events, err = p.Flush()
	if err != nil {
		return 0, err
	}
	return count + len(events), nil
}

// --- Glamour ----------------------------------------------------------------

var glamourRenderer *glamour.TermRenderer

func getGlamourRenderer() *glamour.TermRenderer {
	if glamourRenderer == nil {
		r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
		if err != nil {
			panic(err)
		}
		glamourRenderer = r
	}
	return glamourRenderer
}

func renderGlamour(input string) (string, error) {
	return getGlamourRenderer().Render(input)
}

// --- go-term-markdown -------------------------------------------------------

func renderGoTermMarkdown(input string) string {
	return string(goterm.Render(input, 80, 0))
}

// --- Goldmark (parse only) --------------------------------------------------

var goldmarkMD = goldmark.New()

func parseGoldmark(input []byte) {
	goldmarkMD.Parser().Parse(goldmarktext.NewReader(input))
}

// --- Blackfriday (parse + HTML render) --------------------------------------

func renderBlackfriday(input []byte) []byte {
	return blackfriday.Run(input)
}

// --- Gomarkdown (parse + HTML render) ---------------------------------------

func renderGomarkdown(input []byte) []byte {
	p := gomarkdownparser.NewWithExtensions(gomarkdownparser.CommonExtensions)
	doc := p.Parse(input)
	renderer := gomarkdownhtml.NewRenderer(gomarkdownhtml.RendererOptions{})
	return gomarkdown.Render(doc, renderer)
}

// === Parse-only: all parsers ================================================

func benchParseAll(b *testing.B, name string) {
	input := getInput(name)
	src := []byte(input)
	b.SetBytes(int64(len(input)))

	b.Run("ours", func(b *testing.B) {
		for b.Loop() {
			parseOurs(input)
		}
	})
	b.Run("goldmark", func(b *testing.B) {
		for b.Loop() {
			parseGoldmark(src)
		}
	})
	b.Run("blackfriday", func(b *testing.B) {
		for b.Loop() {
			blackfriday.Run(src)
		}
	})
	b.Run("gomarkdown", func(b *testing.B) {
		for b.Loop() {
			renderGomarkdown(src)
		}
	})
}

func BenchmarkParse_Spec(b *testing.B)       { benchParseAll(b, "spec") }
func BenchmarkParse_README(b *testing.B)      { benchParseAll(b, "readme") }
func BenchmarkParse_GitHubTop10(b *testing.B) { benchParseAll(b, "github-top10") }

// === Terminal render: us vs glamour vs go-term-markdown ======================

func benchRenderTerminal(b *testing.B, name string) {
	input := getInput(name)
	b.SetBytes(int64(len(input)))

	b.Run("ours", func(b *testing.B) {
		for b.Loop() {
			renderOurs(input)
		}
	})
	b.Run("glamour", func(b *testing.B) {
		for b.Loop() {
			renderGlamour(input)
		}
	})
	b.Run("go-term-md", func(b *testing.B) {
		for b.Loop() {
			renderGoTermMarkdown(input)
		}
	})
}

func BenchmarkRender_Spec(b *testing.B)       { benchRenderTerminal(b, "spec") }
func BenchmarkRender_README(b *testing.B)      { benchRenderTerminal(b, "readme") }
func BenchmarkRender_GitHubTop10(b *testing.B) { benchRenderTerminal(b, "github-top10") }
func BenchmarkRender_CodeHeavy(b *testing.B)   { benchRenderTerminal(b, "code-heavy") }
func BenchmarkRender_TableHeavy(b *testing.B)  { benchRenderTerminal(b, "table-heavy") }
func BenchmarkRender_InlineHeavy(b *testing.B) { benchRenderTerminal(b, "inline-heavy") }

// === Pathological inputs: us vs glamour =====================================

func BenchmarkRender_PathologicalNest(b *testing.B)  { benchRenderTerminal(b, "pathological-nest") }
func BenchmarkRender_PathologicalDelim(b *testing.B) { benchRenderTerminal(b, "pathological-delim") }
func BenchmarkRender_LargeFlat(b *testing.B)         { benchRenderTerminal(b, "large-flat") }

// === Chunk size sensitivity (ours only — no other library streams) ==========

func BenchmarkChunkSize_Spec(b *testing.B) {
	input := []byte(getInput("spec"))
	b.SetBytes(int64(len(input)))
	for _, size := range []int{1, 16, 64, 256, 1024, 4096, len(input)} {
		var name string
		switch {
		case size == len(input):
			name = "whole"
		case size >= 1024:
			name = fmt.Sprintf("%dK", size/1024)
		default:
			name = fmt.Sprintf("%d", size)
		}
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				renderOursStreaming(input, size)
			}
		})
	}
}

// === Syntax highlighting: Go fast path vs Chroma ===========================
//
// We benchmark the full render pipeline with Go code using two different
// highlighter configurations:
// - DefaultHighlighter: our stdlib AST-based Go fast path
// - HybridHighlighter with lang="rust": forces Chroma path on Go code
//   (Chroma's Go lexer produces equivalent output)

func BenchmarkHighlight_GoCode(b *testing.B) {
	goCode := "```go\n" +
		"package main\n\n" +
		"import (\n\t\"fmt\"\n\t\"os\"\n)\n\n" +
		"func main() {\n" +
		"\tfmt.Println(\"hello\")\n" +
		"\tos.Exit(0)\n" +
		"}\n" +
		"```\n"
	// Repeat to get meaningful timing
	input := ""
	for i := 0; i < 100; i++ {
		input += goCode
	}
	b.SetBytes(int64(len(input)))

	b.Run("go-fast-path", func(b *testing.B) {
		for b.Loop() {
			var buf bytes.Buffer
			r := terminal.NewStreamRenderer(&buf,
				terminal.WithAnsi(terminal.AnsiOn),
				terminal.WithCodeHighlighter(terminal.NewDefaultHighlighter()),
			)
			r.Write([]byte(input))
			r.Flush()
		}
	})

	b.Run("chroma-for-go", func(b *testing.B) {
		// Use HybridHighlighter but feed it as "rust" so it goes through
		// Chroma instead of the Go fast path. We relabel the code blocks.
		rustInput := strings.ReplaceAll(input, "```go", "```rust")
		b.SetBytes(int64(len(rustInput)))
		for b.Loop() {
			var buf bytes.Buffer
			r := terminal.NewStreamRenderer(&buf,
				terminal.WithAnsi(terminal.AnsiOn),
				terminal.WithCodeHighlighter(terminal.NewHybridHighlighter()),
			)
			r.Write([]byte(rustInput))
			r.Flush()
		}
	})
}

// === Feature parity matrix ==================================================
//
// Parse:   ours, goldmark, blackfriday, gomarkdown, glamour (via goldmark)
// Render:  ours (terminal), glamour (terminal), go-term-markdown (terminal),
//          blackfriday (HTML), gomarkdown (HTML)
// Stream:  ours only

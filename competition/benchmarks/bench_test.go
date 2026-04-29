package benchmarks

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/codewandler/markdown/competition"
)

// === Parse benchmarks =======================================================
//
// Exercises ParseFunc for every variant that supports it, across all inputs.

func BenchmarkParse(b *testing.B) {
	parseInputs := []string{"Spec", "README", "GitHubTop10"}
	benchAdapters(b, parseInputs, func(v competition.Variant) bool {
		return v.Adapters.ParseFunc != nil
	}, func(b *testing.B, v competition.Variant, src []byte) {
		r := bytes.NewReader(src)
		r.Reset(src)
		competition.SafeCall(func() error {
			_, err := v.Adapters.ParseFunc(r)
			return err
		})
	})
}

// === Terminal render benchmarks =============================================
//
// Exercises RenderTerminal for every variant that supports it.

func BenchmarkRender(b *testing.B) {
	renderInputs := []string{
		"Spec", "README", "GitHubTop10",
		"CodeHeavy", "TableHeavy", "InlineHeavy",
	}
	benchAdapters(b, renderInputs, func(v competition.Variant) bool {
		return v.Adapters.RenderTerminal != nil
	}, func(b *testing.B, v competition.Variant, src []byte) {
		r := bytes.NewReader(src)
		var w bytes.Buffer
		r.Reset(src)
		w.Reset()
		competition.SafeCall(func() error {
			return v.Adapters.RenderTerminal(r, &w)
		})
	})
}

// === HTML render benchmarks ==================================================
//
// Exercises RenderHTML for every variant that supports it.
// Compares: ours vs goldmark vs blackfriday vs gomarkdown.

func BenchmarkRenderHTML(b *testing.B) {
	htmlInputs := []string{
		"Spec", "README", "GitHubTop10",
		"CodeHeavy", "TableHeavy", "InlineHeavy",
	}
	benchAdapters(b, htmlInputs, func(v competition.Variant) bool {
		return v.Adapters.RenderHTML != nil
	}, func(b *testing.B, v competition.Variant, src []byte) {
		r := bytes.NewReader(src)
		var w bytes.Buffer
		r.Reset(src)
		w.Reset()
		competition.SafeCall(func() error {
			return v.Adapters.RenderHTML(r, &w)
		})
	})
}

// === Pathological input benchmarks ==========================================
//
// Terminal render on adversarial inputs. Tests robustness and worst-case
// performance.

func BenchmarkPathological(b *testing.B) {
	pathInputs := []string{
		"PathologicalNest", "PathologicalDelim", "LargeFlat",
	}
	benchAdapters(b, pathInputs, func(v competition.Variant) bool {
		return v.Adapters.RenderTerminal != nil
	}, func(b *testing.B, v competition.Variant, src []byte) {
		r := bytes.NewReader(src)
		var w bytes.Buffer
		r.Reset(src)
		w.Reset()
		competition.SafeCall(func() error {
			return v.Adapters.RenderTerminal(r, &w)
		})
	})
}

// === Chunk size sensitivity (ours only) =====================================
//
// Tests streaming at various chunk sizes. Only variants whose name
// starts with "ours" are included.

func BenchmarkChunkSize(b *testing.B) {
	input := getInput("Spec")
	src := []byte(input)
	b.SetBytes(int64(len(src)))

	// Find the chunked variant.
	for _, c := range competition.All {
		for _, v := range c.Variants {
			if v.Adapters.RenderTerminal == nil {
				continue
			}
			if !strings.HasPrefix(v.Name, "ours") {
				continue
			}
			// Only run chunk sensitivity on the default variant;
			// the chunked variants are tested via BenchmarkRender.
			if v.Name != "ours" {
				continue
			}
			for _, size := range []int{1, 16, 64, 256, 1024, 4096, len(src)} {
				name := chunkSizeName(size, len(src))
				b.Run(name, func(b *testing.B) {
					for b.Loop() {
						r := &chunkedReader{data: src, chunk: size}
						var w bytes.Buffer
						v.Adapters.RenderTerminal(r, &w)
					}
				})
			}
		}
	}
}

// === Syntax highlighting ====================================================
//
// Compares Go fast path vs Chroma via the "ours" variant with
// different highlighter configurations. This benchmark is specific
// to our library and doesn't iterate all candidates.

func BenchmarkHighlight(b *testing.B) {
	goCode := "```go\n" +
		"package main\n\n" +
		"import (\n\t\"fmt\"\n\t\"os\"\n)\n\n" +
		"func main() {\n" +
		"\tfmt.Println(\"hello\")\n" +
		"\tos.Exit(0)\n" +
		"}\n" +
		"```\n"
	input := strings.Repeat(goCode, 100)
	src := []byte(input)
	b.SetBytes(int64(len(src)))

	// Find our default variant for the Go fast path test.
	for _, c := range competition.All {
		for _, v := range c.Variants {
			if v.Name != "ours" || v.Adapters.RenderTerminal == nil {
				continue
			}
			b.Run("go-fast-path", func(b *testing.B) {
				for b.Loop() {
					r := bytes.NewReader(src)
					var w bytes.Buffer
					v.Adapters.RenderTerminal(r, &w)
				}
			})
		}
	}

	// Chroma path: relabel as "rust" to force Chroma.
	rustInput := []byte(strings.ReplaceAll(input, "```go", "```rust"))
	b.SetBytes(int64(len(rustInput)))
	for _, c := range competition.All {
		for _, v := range c.Variants {
			if v.Name != "ours" || v.Adapters.RenderTerminal == nil {
				continue
			}
			b.Run("chroma-for-go", func(b *testing.B) {
				for b.Loop() {
					r := bytes.NewReader(rustInput)
					var w bytes.Buffer
					v.Adapters.RenderTerminal(r, &w)
				}
			})
		}
	}
}

// === Helpers ================================================================

// benchAdapters is the generic harness that iterates all candidates,
// variants, and inputs. It skips variants where the filter returns false.
func benchAdapters(
	b *testing.B,
	inputNames []string,
	filter func(competition.Variant) bool,
	run func(b *testing.B, v competition.Variant, src []byte),
) {
	b.Helper()
	inputs := indexInputs(inputNames)

	for _, c := range competition.All {
		for _, v := range c.Variants {
			if !filter(v) {
				continue
			}
			for _, inp := range inputs {
				name := v.Name + "/" + inp.name
				src := []byte(inp.data)
				b.Run(name, func(b *testing.B) {
					b.SetBytes(int64(len(src)))
					for b.Loop() {
						run(b, v, src)
					}
				})
			}
		}
	}
}

type indexedInput struct {
	name string
	data string
}

// indexInputs resolves input names to their generated data.
func indexInputs(names []string) []indexedInput {
	all := AllInputs()
	byName := make(map[string]Input, len(all))
	for _, inp := range all {
		byName[inp.Name] = inp
	}
	result := make([]indexedInput, 0, len(names))
	for _, name := range names {
		inp, ok := byName[name]
		if !ok {
			panic("unknown input: " + name)
		}
		result = append(result, indexedInput{name: name, data: inp.Generate()})
	}
	return result
}

// getInput returns the generated data for a single input by name.
func getInput(name string) string {
	for _, inp := range AllInputs() {
		if inp.Name == name {
			return inp.Generate()
		}
	}
	panic("unknown input: " + name)
}

func chunkSizeName(size, total int) string {
	switch {
	case size == total:
		return "whole"
	case size >= 1024:
		return fmt.Sprintf("%dK", size/1024)
	default:
		return fmt.Sprintf("%d", size)
	}
}

// chunkedReader wraps a byte slice and delivers it in fixed-size chunks,
// simulating streaming input.
type chunkedReader struct {
	data  []byte
	chunk int
	pos   int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	end := r.pos + r.chunk
	if end > len(r.data) {
		end = len(r.data)
	}
	n := copy(p, r.data[r.pos:end])
	r.pos += n
	return n, nil
}


package stream

import (
	"strings"
	"testing"

	"github.com/codewandler/markdown/internal/commonmarktests"
)

var benchmarkEventCount int

func BenchmarkParserLongFence(b *testing.B) {
	input := "```go\n" + strings.Repeat("fmt.Println(\"hello\")\n", 10000) + "```\n"
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		benchmarkEventCount = parseBenchmarkInput(b, input)
	}
}

func BenchmarkParserLongParagraph(b *testing.B) {
	input := strings.Repeat("alpha beta gamma ", 20000) + "\n"
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		benchmarkEventCount = parseBenchmarkInput(b, input)
	}
}

func BenchmarkParserTinyChunks(b *testing.B) {
	input := []byte("```go\n" + strings.Repeat("fmt.Println(\"hello\")\n", 1000) + "```\n")
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		p := NewParser()
		count := 0
		for _, c := range input {
			events, err := p.Write([]byte{c})
			if err != nil {
				b.Fatal(err)
			}
			count += len(events)
		}
		events, err := p.Flush()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkEventCount = count + len(events)
	}
}

func BenchmarkParserCommonMarkCorpus(b *testing.B) {
	input := commonMarkBenchmarkInput(b)
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		benchmarkEventCount = parseBenchmarkInput(b, input)
	}
}

func BenchmarkParserCommonMarkCorpusTinyChunks(b *testing.B) {
	input := []byte(commonMarkBenchmarkInput(b))
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		p := NewParser()
		count := 0
		for _, c := range input {
			events, err := p.Write([]byte{c})
			if err != nil {
				b.Fatal(err)
			}
			count += len(events)
		}
		events, err := p.Flush()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkEventCount = count + len(events)
	}
}

func BenchmarkParserMalformedInlineDelimiters(b *testing.B) {
	input := strings.Repeat("**unterminated * _ __ ` [link]( <http://example.com\n", 5000)
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		benchmarkEventCount = parseBenchmarkInput(b, input)
	}
}

func BenchmarkParserPathologicalInlineDelimiters(b *testing.B) {
	input := strings.Repeat("*_`[<", 20000) + "\n"
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		benchmarkEventCount = parseBenchmarkInput(b, input)
	}
}

func parseBenchmarkInput(b *testing.B, input string) int {
	b.Helper()
	p := NewParser()
	events, err := p.Write([]byte(input))
	if err != nil {
		b.Fatal(err)
	}
	count := len(events)
	events, err = p.Flush()
	if err != nil {
		b.Fatal(err)
	}
	return count + len(events)
}

func commonMarkBenchmarkInput(b *testing.B) string {
	b.Helper()
	examples, err := commonmarktests.Load()
	if err != nil {
		b.Fatal(err)
	}
	var input strings.Builder
	for _, ex := range examples {
		input.WriteString(ex.Markdown)
		input.WriteString("\n")
	}
	return input.String()
}

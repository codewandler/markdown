package stream

import (
	"strings"
	"testing"
)

func BenchmarkParserLongFence(b *testing.B) {
	input := "```go\n" + strings.Repeat("fmt.Println(\"hello\")\n", 10000) + "```\n"
	for b.Loop() {
		p := NewParser()
		if _, err := p.Write([]byte(input)); err != nil {
			b.Fatal(err)
		}
		if _, err := p.Flush(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserLongParagraph(b *testing.B) {
	input := strings.Repeat("alpha beta gamma ", 20000) + "\n"
	for b.Loop() {
		p := NewParser()
		if _, err := p.Write([]byte(input)); err != nil {
			b.Fatal(err)
		}
		if _, err := p.Flush(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserTinyChunks(b *testing.B) {
	input := []byte("```go\n" + strings.Repeat("fmt.Println(\"hello\")\n", 1000) + "```\n")
	for b.Loop() {
		p := NewParser()
		for _, c := range input {
			if _, err := p.Write([]byte{c}); err != nil {
				b.Fatal(err)
			}
		}
		if _, err := p.Flush(); err != nil {
			b.Fatal(err)
		}
	}
}

package html

import (
	"testing"

	"github.com/codewandler/markdown/stream"
)

func TestRenderSupportedSubset(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "heading",
			in:   "# Hello *world*\n",
			want: "<h1>Hello <em>world</em></h1>\n",
		},
		{
			name: "paragraph strong code link",
			in:   "A **strong** `code` [link](https://example.com)\n",
			want: "<p>A <strong>strong</strong> <code>code</code> <a href=\"https://example.com\">link</a></p>\n",
		},
		{
			name: "fenced code",
			in:   "```go\npackage main\n```\n",
			want: "<pre><code class=\"language-go\">package main\n</code></pre>\n",
		},
		{
			name: "thematic break",
			in:   "---\n",
			want: "<hr />",
		},
		{
			name: "list",
			in:   "- one\n- two\n\n",
			want: "<ul>\n<li><p>one</p>\n</li>\n<li><p>two</p>\n</li>\n</ul>\n",
		},
		{
			name: "blockquote",
			in:   "> quote\n\n",
			want: "<blockquote>\n<p>quote</p>\n</blockquote>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := stream.NewParser()
			events, err := p.Write([]byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			flush, err := p.Flush()
			if err != nil {
				t.Fatal(err)
			}
			events = append(events, flush...)
			if got := Render(events); got != tt.want {
				t.Fatalf("html mismatch\nwant: %q\n got: %q", tt.want, got)
			}
		})
	}
}

package html

import "testing"

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "hello"},
		{"a & b", "a &amp; b"},
		{"<div>", "&lt;div&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"<a href=\"&\">", "&lt;a href=&quot;&amp;&quot;&gt;"},
		{"no special chars", "no special chars"},
		{"&amp;", "&amp;amp;"},
		{"a < b > c & d \"e\"", "a &lt; b &gt; c &amp; d &quot;e&quot;"},
	}
	for _, tt := range tests {
		got := escapeHTML(tt.in)
		if got != tt.want {
			t.Errorf("escapeHTML(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"https://example.com", "https://example.com"},
		{"https://example.com/path?q=hello&id=1", "https://example.com/path?q=hello&id=1"},
		{"/url with spaces", "/url%20with%20spaces"},
		{"https://example.com/ä", "https://example.com/%C3%A4"},
		// Preserve already-encoded sequences.
		{"https://example.com/%20path", "https://example.com/%20path"},
		{"/url%3Fq", "/url%3Fq"},
		// Bare percent that is not a valid %XX.
		{"100%done", "100%25done"},
		// Mixed.
		{"/path with spaces/%20already", "/path%20with%20spaces/%20already"},
	}
	for _, tt := range tests {
		got := escapeURL(tt.in)
		if got != tt.want {
			t.Errorf("escapeURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

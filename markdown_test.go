package markdown_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codewandler/markdown"
	"github.com/codewandler/markdown/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderString_HeadingAndParagraph(t *testing.T) {
	out, err := markdown.RenderString("# Hello\n\nWorld\n", terminal.WithAnsi(terminal.AnsiOn))
	require.NoError(t, err)
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "World")
	assert.NotContains(t, out, "# Hello")
	assert.Contains(t, out, "\x1b[") // ANSI present
}

func TestRenderString_FencedCode(t *testing.T) {
	out, err := markdown.RenderString("```go\npackage main\n```\n", terminal.WithAnsi(terminal.AnsiOn))
	require.NoError(t, err)
	assert.Contains(t, out, "package")
	assert.Contains(t, out, "main")
	assert.NotContains(t, out, "```")
	assert.Contains(t, out, "\x1b[")
}

func TestRenderString_EmptyInput(t *testing.T) {
	out, err := markdown.RenderString("")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRenderToWriter_WritesToProvidedWriter(t *testing.T) {
	var buf bytes.Buffer
	err := markdown.RenderToWriter(&buf, "**bold** and `code`\n", terminal.WithAnsi(terminal.AnsiOn))
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "bold")
	assert.Contains(t, out, "code")
	assert.NotContains(t, out, "**bold**")
	assert.NotContains(t, out, "`code`")
	assert.Contains(t, out, "\x1b[")
}

func TestRenderToWriter_MatchesRenderString(t *testing.T) {
	src := "## Section\n\n- item one\n- item two\n\n> blockquote\n"
	fromString, err := markdown.RenderString(src)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, markdown.RenderToWriter(&buf, src))

	assert.Equal(t, fromString, buf.String())
}

func TestRenderString_InlineStyles(t *testing.T) {
	out, err := markdown.RenderString("*italic* **bold** ~~strike~~ `code`\n", terminal.WithAnsi(terminal.AnsiOn))
	require.NoError(t, err)
	assert.Contains(t, out, "\x1b[3m") // italic
	assert.Contains(t, out, "\x1b[1m") // bold
	assert.Contains(t, out, "\x1b[9m") // strikethrough
	// inline code uses monokaiYellow
	assert.Contains(t, out, "\x1b[38;2;230;219;116m")
	// raw markers must not appear
	assert.NotContains(t, out, "**")
	assert.NotContains(t, out, "~~")
}

func TestRenderString_Table(t *testing.T) {
	src := "| A | B |\n| --- | --- |\n| 1 | 2 |\n"
	out, err := markdown.RenderString(src)
	require.NoError(t, err)
	plain := strings.NewReplacer("\x1b[", "", "m", "").Replace(out)
	_ = plain
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "B")
	assert.Contains(t, out, "│") // box-drawing border
	assert.NotContains(t, out, "---")
}

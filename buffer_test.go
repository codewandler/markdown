package markdown

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuffer_ParagraphNeedsBoundary(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	n, err := b.WriteString("Hello")
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Empty(t, got)
	require.Equal(t, "Hello", b.Pending())

	_, err = b.WriteString(" world\n\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockParagraph, got[0].Kind)
	require.Equal(t, "Hello world\n", got[0].Markdown)
	require.Equal(t, "", b.Pending())
}

func TestBuffer_HeadingEmitsOnCompletedLine(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("## Title\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockHeading, got[0].Kind)
	require.Equal(t, "## Title\n", got[0].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_OpenFenceHeldUntilClosed(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("Before\n\n```go\nfmt.Println(\"hi\")\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockParagraph, got[0].Kind)
	require.Equal(t, "Before\n", got[0].Markdown)
	require.Equal(t, "```go\nfmt.Println(\"hi\")\n", b.Pending())

	_, err = b.WriteString("```\n")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, BlockCodeFence, got[1].Kind)
	require.Equal(t, "```go\nfmt.Println(\"hi\")\n```\n", got[1].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_TildeFenceHeldUntilClosed(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("~~~txt\nabc\n")
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, "~~~txt\nabc\n", b.Pending())

	_, err = b.WriteString("~~~~\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockCodeFence, got[0].Kind)
	require.Equal(t, "~~~txt\nabc\n~~~~\n", got[0].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_FenceContentDoesNotCloseFence(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("```txt\nnot a close ``` inside\n")
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, "```txt\nnot a close ``` inside\n", b.Pending())

	_, err = b.WriteString("```\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockCodeFence, got[0].Kind)
}

func TestBuffer_IndentedFenceUpToThreeSpaces(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("   ```go\nfmt.Println(1)\n")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = b.WriteString("   ```\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockCodeFence, got[0].Kind)
	require.Equal(t, "```go\nfmt.Println(1)\n   ```\n", got[0].Markdown)
}

func TestBuffer_TabIndentedFenceParsesAsIndentedCodeBlock(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("	```go\nfmt.Println(1)\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockCode, got[0].Kind)
	require.Equal(t, "```go\n", got[0].Markdown)
	require.Equal(t, "fmt.Println(1)\n", b.Pending())

	require.NoError(t, b.Flush())
	require.Len(t, got, 2)
	require.Equal(t, BlockParagraph, got[1].Kind)
	require.Equal(t, "fmt.Println(1)\n", got[1].Markdown)
}

func TestBuffer_UnclosedFenceFlushesAtEnd(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("```txt\nabc\n")
	require.NoError(t, err)
	require.Empty(t, got)

	require.NoError(t, b.Flush())
	require.Len(t, got, 1)
	require.Equal(t, BlockCodeFence, got[0].Kind)
	require.Equal(t, "```txt\nabc\n", got[0].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_SetextHeadingEmitsOnCompletedBoundaryLine(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("Title\n---\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockHeading, got[0].Kind)
	require.Equal(t, "Title\n---\n", got[0].Markdown)
	require.Equal(t, "", b.Pending())
}

func TestBuffer_ATXHeadingWithoutTrailingNewlineStaysPending(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("## Title")
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, "## Title", b.Pending())

	require.NoError(t, b.Flush())
	require.Len(t, got, 1)
	require.Equal(t, BlockHeading, got[0].Kind)
	require.Equal(t, "## Title", got[0].Markdown)
}

func TestBuffer_ListNeedsClosingBoundary(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("- one\n- two\n")
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, "- one\n- two\n", b.Pending())

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockList, got[0].Kind)
	require.Equal(t, "- one\n- two\n", got[0].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_NestedListNeedsClosingBoundary(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("- parent\n  - child\n")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockList, got[0].Kind)
	require.Equal(t, "- parent\n  - child\n", got[0].Markdown)
}

func TestBuffer_ListContinuationParagraphStaysInSameBlock(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("- item\n\n  continuation\n")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockList, got[0].Kind)
	require.Equal(t, "- item\n\n  continuation\n", got[0].Markdown)
}

func TestBuffer_ParagraphThenListAcrossChunks(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("Para\n")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = b.WriteString("- item\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockParagraph, got[0].Kind)
	require.Equal(t, "Para\n", got[0].Markdown)
	require.Equal(t, "- item\n", b.Pending())

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, BlockList, got[1].Kind)
	require.Equal(t, "- item\n", got[1].Markdown)
}

func TestBuffer_BlockquoteNeedsClosingBoundary(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("> a\n")
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, "> a\n", b.Pending())

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockBlockquote, got[0].Kind)
	require.Equal(t, "> a\n", got[0].Markdown)
}

func TestBuffer_NestedBlockquoteNeedsClosingBoundary(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("> outer\n> > inner\n")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockBlockquote, got[0].Kind)
	require.Equal(t, "> outer\n> > inner\n", got[0].Markdown)
}

func TestBuffer_BlockquoteLazyContinuationStaysBuffered(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("> quote\ncontinuation\n")
	require.NoError(t, err)
	require.Empty(t, got)
	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockBlockquote, got[0].Kind)
	require.Equal(t, "> quote\ncontinuation\n", got[0].Markdown)
}

func TestBuffer_HTMLBlockNeedsClosingBoundary(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("<div>\nhello\n</div>\n")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = b.WriteString("\n")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, BlockHTML, got[0].Kind)
	require.Equal(t, "<div>\nhello\n</div>\n", got[0].Markdown)
}

func TestBuffer_IncompleteHTMLFlushesAtEnd(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("<div>\nhello\n")
	require.NoError(t, err)
	require.Empty(t, got)

	require.NoError(t, b.Flush())
	require.Len(t, got, 1)
	require.Equal(t, BlockHTML, got[0].Kind)
	require.Equal(t, "<div>\nhello\n", got[0].Markdown)
}

func TestBuffer_FlushEmitsFinalTail(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("tail without blank line")
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, "tail without blank line", b.Pending())

	require.NoError(t, b.Flush())
	require.Len(t, got, 1)
	require.Equal(t, BlockParagraph, got[0].Kind)
	require.Equal(t, "tail without blank line", got[0].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_FlushPreservesWhitespaceOnlyTail(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	_, err := b.WriteString("\n\t")
	require.NoError(t, err)
	require.Empty(t, got)

	require.NoError(t, b.Flush())
	require.Len(t, got, 1)
	require.Equal(t, BlockOther, got[0].Kind)
	require.Equal(t, "\n\t", got[0].Markdown)
	require.Empty(t, b.Pending())
}

func TestBuffer_MultipleBlocksOneWrite(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	src := "# Title\nParagraph one.\n\n- a\n- b\n\n"
	_, err := b.WriteString(src)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, []BlockKind{BlockHeading, BlockParagraph, BlockList}, []BlockKind{got[0].Kind, got[1].Kind, got[2].Kind})
	require.Equal(t, "# Title\n", got[0].Markdown)
	require.Equal(t, "Paragraph one.\n", got[1].Markdown)
	require.Equal(t, "- a\n- b\n", got[2].Markdown)
}

func TestBuffer_MultipleBlocksRemainConcatenableWhenSeparatorsAreProvidedByConsumer(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	src := "Paragraph one.\n\n- a\n- b\n\n"
	_, err := b.WriteString(src)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "Paragraph one.\n", got[0].Markdown)
	require.Equal(t, "- a\n- b\n", got[1].Markdown)
	require.Equal(t, src, got[0].Markdown+"\n"+got[1].Markdown+"\n")
}

func TestBuffer_ByteByByteFence(t *testing.T) {
	var emitted strings.Builder
	b := NewBuffer(func(blocks []Block) {
		for _, block := range blocks {
			emitted.WriteString(block.Markdown)
		}
	})

	src := "Text\n\n```txt\nabc\n```\n"
	for i := 0; i < len(src); i++ {
		_, err := b.Write([]byte{src[i]})
		require.NoError(t, err)
	}
	require.Equal(t, "Text\n```txt\nabc\n```\n", emitted.String())
	require.Empty(t, b.Pending())
}

func TestBuffer_ResetClearsPending(t *testing.T) {
	b := NewBuffer(nil)
	_, err := b.WriteString("partial")
	require.NoError(t, err)
	require.Equal(t, "partial", b.Pending())
	b.Reset()
	require.Empty(t, b.Pending())
}

func TestBuffer_WriteReturnsLenOnInternalError(t *testing.T) {
	b := NewBuffer(nil)
	b.md = nil // intentionally invalid internal state to force an error path

	n, err := b.WriteString("hello")
	require.Error(t, err)
	require.Equal(t, len("hello"), n)
}

func TestBuffer_ForceSplitsVeryLongPendingBlocks(t *testing.T) {
	var got []Block
	b := NewBuffer(func(blocks []Block) { got = append(got, blocks...) })

	// Write a long paragraph without any blank line.
	long := strings.Repeat("word ", 900) // ~4500 bytes
	_, err := b.WriteString(long)
	require.NoError(t, err)
	require.NotEmpty(t, got, "long block should have been force-emitted")
	require.Equal(t, BlockParagraph, got[len(got)-1].Kind)
}

func TestBuffer_CallbackDeliveryIsSerialized(t *testing.T) {
	var active int32
	var overlap int32
	var calls int32

	b := NewBuffer(func(blocks []Block) {
		if atomic.AddInt32(&active, 1) > 1 {
			atomic.StoreInt32(&overlap, 1)
		}
		defer atomic.AddInt32(&active, -1)
		atomic.AddInt32(&calls, 1)
		time.Sleep(20 * time.Millisecond)
	})

	var wg sync.WaitGroup
	for _, chunk := range []string{"# A\n", "# B\n", "# C\n"} {
		wg.Add(1)
		go func(chunk string) {
			defer wg.Done()
			_, err := b.WriteString(chunk)
			require.NoError(t, err)
		}(chunk)
	}
	wg.Wait()

	require.Zero(t, atomic.LoadInt32(&overlap), "callback invocations must not overlap")
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

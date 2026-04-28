package markdown

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
)

// BlockKind describes a coarse top-level markdown block category.
type BlockKind string

const (
	BlockParagraph  BlockKind = "paragraph"
	BlockHeading    BlockKind = "heading"
	BlockList       BlockKind = "list"
	BlockCodeFence  BlockKind = "code_fence"
	BlockCode       BlockKind = "code"
	BlockBlockquote BlockKind = "blockquote"
	BlockHTML       BlockKind = "html"
	BlockThematic   BlockKind = "thematic_break"
	BlockOther      BlockKind = "other"
)

// maxPendingBytes is the soft upper bound for buffered content before forcing
// a paragraph split. This prevents O(n²) re-parsing on very long single blocks.
const maxPendingBytes = 4096

// Block is a stable top-level markdown block extracted from the stream.
type Block struct {
	Markdown string
	Kind     BlockKind
}

// BlockHandler receives one or more newly stabilized markdown blocks.
type BlockHandler func([]Block)

// BufferOption configures a markdown Buffer.
type BufferOption func(*Buffer)

// WithMarkdown injects a custom goldmark instance.
//
// Note: streaming stability heuristics are tuned for standard goldmark
// CommonMark block behavior. If a custom instance enables extensions with
// different block parsing semantics, buffering behavior may become more or less
// conservative than expected.
func WithMarkdown(md goldmark.Markdown) BufferOption {
	return func(b *Buffer) {
		if md != nil {
			b.md = md
		}
	}
}

// Buffer accepts partial markdown writes and emits stable top-level blocks.
//
// Use [NewBuffer] to construct a Buffer. The zero value is not ready for use.
//
// Emission contract:
//   - Output is append-only: once a block has been emitted it will never be retracted.
//   - Emitted items are whole top-level markdown blocks, never inline fragments.
//   - Emitted block markdown is normalized for standalone rendering: inter-block
//     trailing blank lines are trimmed while intrinsic block content is preserved.
//   - [Write] is conservative and may keep a trailing incomplete block buffered.
//   - [Flush] is end-of-stream behavior and emits the remaining buffered tail
//     best-effort, even if it would normally be held back during streaming.
//   - Callback delivery is serialized: handler invocations never overlap, even
//     when multiple goroutines call [Write] concurrently.
//
// Internally, goldmark is used to parse top-level blocks. The stability policy
// is intentionally conservative:
//   - trailing incomplete lines are kept buffered
//   - fenced code blocks are held until their closing fence arrives
//   - most container blocks require a separating blank line before emission
//   - Flush emits whatever remains at end-of-stream
//
// Buffer is safe for concurrent use.
type Buffer struct {
	mu      sync.Mutex
	cbMu    sync.Mutex
	handler BlockHandler
	md      goldmark.Markdown
	pending string
}

// NewBuffer creates a new markdown buffer.
func NewBuffer(handler BlockHandler, opts ...BufferOption) *Buffer {
	if handler == nil {
		handler = func([]Block) {}
	}
	b := &Buffer{handler: handler, md: goldmark.New()}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

// Write appends markdown data and emits any newly stable blocks.
//
// If an internal processing error occurs after the bytes have been accepted into
// the buffer, Write returns (len(p), err).
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.pending += string(p)
	blocks, err := b.processLocked(false)
	b.mu.Unlock()
	if err != nil {
		return len(p), err
	}
	if len(blocks) > 0 {
		b.deliver(blocks)
	}
	return len(p), nil
}

// WriteString appends markdown text and emits any newly stable blocks.
func (b *Buffer) WriteString(s string) (int, error) {
	return b.Write([]byte(s))
}

// Flush emits the remaining buffered content and clears the pending buffer.
//
// Flush is best-effort finalization for end-of-stream handling.
func (b *Buffer) Flush() error {
	b.mu.Lock()
	blocks, err := b.processLocked(true)
	b.mu.Unlock()
	if err != nil {
		return err
	}
	if len(blocks) > 0 {
		b.deliver(blocks)
	}
	return nil
}

// Reset clears the buffered tail without emitting it.
func (b *Buffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = ""
}

// Pending returns the currently buffered, not-yet-emitted markdown tail.
func (b *Buffer) Pending() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pending
}

func (b *Buffer) deliver(blocks []Block) {
	b.cbMu.Lock()
	defer b.cbMu.Unlock()
	b.handler(blocks)
}

func (b *Buffer) processLocked(force bool) ([]Block, error) {
	blocks, remainder, err := b.extractStableBlocks(b.pending, force)
	if err != nil {
		return nil, err
	}
	b.pending = remainder
	return blocks, nil
}

func (b *Buffer) extractStableBlocks(src string, force bool) ([]Block, string, error) {
	if src == "" {
		return nil, "", nil
	}
	if b.md == nil || b.md.Parser() == nil {
		return nil, src, fmt.Errorf("markdown: buffer is not initialized; use NewBuffer")
	}

	source := []byte(src)
	doc := b.md.Parser().Parse(gmtext.NewReader(source))
	nodes := topLevelBlocks(doc)
	if len(nodes) == 0 {
		if force {
			return []Block{{Markdown: src, Kind: BlockOther}}, "", nil
		}
		return nil, src, nil
	}

	starts := make([]int, len(nodes))
	for i, n := range nodes {
		start, ok := nodeStart(n)
		if !ok {
			return nil, src, fmt.Errorf("markdown: could not determine block start for %s", n.Kind())
		}
		starts[i] = start
	}

	stableCount := stableBlockCount(src, nodes, starts, force)
	if stableCount <= 0 {
		return nil, src, nil
	}

	stableEnd := len(src)
	if stableCount < len(nodes) {
		stableEnd = starts[stableCount]
	}

	blocks := make([]Block, 0, stableCount)
	for i := 0; i < stableCount; i++ {
		end := stableEnd
		if i+1 < len(nodes) && starts[i+1] < end {
			end = starts[i+1]
		}
		if end < starts[i] {
			continue
		}
		blocks = append(blocks, Block{
			Markdown: trimTrailingBlankLines(src[starts[i]:end]),
			Kind:     blockKind(nodes[i]),
		})
	}
	return blocks, src[stableEnd:], nil
}

func topLevelBlocks(doc ast.Node) []ast.Node {
	var nodes []ast.Node
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		nodes = append(nodes, n)
	}
	return nodes
}

func stableBlockCount(src string, nodes []ast.Node, starts []int, force bool) int {
	if force {
		return len(nodes)
	}
	if idx := firstUnclosedFenceStart(src); idx >= 0 {
		for i, start := range starts {
			if start >= idx {
				return i
			}
		}
		return len(nodes)
	}
	if endsWithBlankLine(src) {
		return len(nodes)
	}
	if len(nodes) == 0 {
		return 0
	}
	// Force-split very long pending buffers to avoid O(n²) re-parsing.
	if len(src) > maxPendingBytes {
		return len(nodes)
	}
	if !strings.HasSuffix(src, "\n") {
		return max(0, len(nodes)-1)
	}
	if lastBlockStableWithoutBlankLine(nodes[len(nodes)-1]) {
		return len(nodes)
	}
	return len(nodes) - 1
}

func lastBlockStableWithoutBlankLine(n ast.Node) bool {
	switch n.(type) {
	case *ast.Heading, *ast.ThematicBreak, *ast.FencedCodeBlock, *ast.CodeBlock:
		return true
	default:
		return false
	}
}

func endsWithBlankLine(src string) bool {
	if !strings.HasSuffix(src, "\n") {
		return false
	}
	trimmed := strings.TrimSuffix(src, "\n")
	if trimmed == "" {
		return true
	}
	lastNL := strings.LastIndexByte(trimmed, '\n')
	if lastNL < 0 {
		return false
	}
	return strings.TrimSpace(trimmed[lastNL+1:]) == ""
}

func trimTrailingBlankLines(s string) string {
	if s == "" || !strings.HasSuffix(s, "\n") {
		return s
	}
	lines := splitLinesKeepNewline(s)
	keep := len(lines)
	for keep > 0 && strings.TrimSpace(strings.TrimSuffix(lines[keep-1], "\n")) == "" {
		keep--
	}
	if keep == len(lines) || keep == 0 {
		return s
	}
	return strings.Join(lines[:keep], "")
}

// firstUnclosedFenceStart returns the byte offset of the first still-open fence
// in src, or -1 if all fences are closed.
//
// This supplements the parser with a conservative streaming check: CommonMark
// allows an unclosed fence to absorb the rest of the document until EOF, so we
// must not emit any block that starts inside such a fence.
func firstUnclosedFenceStart(src string) int {
	var openMarker byte
	var openLen int
	var openStart int
	cursor := 0
	for _, line := range splitLinesKeepNewline(src) {
		trimmedLeft := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmedLeft)
		if indent <= 3 && len(trimmedLeft) >= 3 {
			marker := trimmedLeft[0]
			if marker == '`' || marker == '~' {
				run := countLeadingByte(trimmedLeft, marker)
				if run >= 3 {
					if openMarker == 0 {
						openMarker = marker
						openLen = run
						openStart = cursor
					} else if marker == openMarker && run >= openLen {
						openMarker = 0
						openLen = 0
						openStart = 0
					}
				}
			}
		}
		cursor += len(line)
	}
	if openMarker != 0 {
		return openStart
	}
	return -1
}

func splitLinesKeepNewline(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func countLeadingByte(s string, b byte) int {
	count := 0
	for count < len(s) && s[count] == b {
		count++
	}
	return count
}

func nodeStart(n ast.Node) (int, bool) {
	best := -1
	if pos := n.Pos(); pos >= 0 {
		best = pos
	}
	if n.Type() == ast.TypeBlock || n.Type() == ast.TypeDocument {
		if lines := n.Lines(); lines != nil {
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				if seg.Start >= 0 && (best == -1 || seg.Start < best) {
					best = seg.Start
				}
			}
		}
	}
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if start, ok := nodeStart(child); ok && (best == -1 || start < best) {
			best = start
		}
	}
	return best, best >= 0
}

func blockKind(n ast.Node) BlockKind {
	switch n.(type) {
	case *ast.Paragraph:
		return BlockParagraph
	case *ast.Heading:
		return BlockHeading
	case *ast.List:
		return BlockList
	case *ast.FencedCodeBlock:
		return BlockCodeFence
	case *ast.CodeBlock:
		return BlockCode
	case *ast.Blockquote:
		return BlockBlockquote
	case *ast.HTMLBlock:
		return BlockHTML
	case *ast.ThematicBreak:
		return BlockThematic
	default:
		return BlockOther
	}
}

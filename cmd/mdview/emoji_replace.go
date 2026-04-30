package main

import (
	"io"
	"strings"
)

// emojiReader wraps an io.Reader and replaces :shortcode: patterns
// with their unicode emoji equivalents in the input stream, before
// the Markdown parser sees the text. This ensures table column widths
// account for emoji display width.
type emojiReader struct {
	r    io.Reader
	buf  []byte // raw input buffer
	out  []byte // processed output ready to be read
}

func newEmojiReader(r io.Reader) *emojiReader {
	return &emojiReader{r: r}
}

func (e *emojiReader) Read(p []byte) (int, error) {
	// Return buffered output first.
	if len(e.out) > 0 {
		n := copy(p, e.out)
		e.out = e.out[n:]
		return n, nil
	}

	// Read more input.
	tmp := make([]byte, 8192)
	n, err := e.r.Read(tmp)
	if n > 0 {
		e.buf = append(e.buf, tmp[:n]...)
	}

	// Process complete lines from buf.
	for {
		i := indexByte(e.buf, '\n')
		if i < 0 {
			// If EOF, process remaining buffer.
			if err == io.EOF && len(e.buf) > 0 {
				e.out = append(e.out, []byte(replaceEmoji(string(e.buf)))...)
				e.buf = nil
			}
			break
		}
		line := string(e.buf[:i+1])
		e.buf = e.buf[i+1:]
		e.out = append(e.out, []byte(replaceEmoji(line))...)
	}

	if len(e.out) > 0 {
		n := copy(p, e.out)
		e.out = e.out[n:]
		return n, err
	}
	return 0, err
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func replaceEmoji(s string) string {
	if !strings.Contains(s, ":") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] != ':' {
			b.WriteByte(s[i])
			i++
			continue
		}
		// Look for closing :
		end := strings.IndexByte(s[i+1:], ':')
		if end < 0 {
			b.WriteByte(s[i])
			i++
			continue
		}
		name := s[i+1 : i+1+end]
		if emoji, ok := emojiShortcodes[name]; ok {
			b.WriteString(emoji)
			i = i + 1 + end + 1
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

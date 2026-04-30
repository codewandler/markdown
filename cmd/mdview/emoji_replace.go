package main

import (
	"io"
	"strings"
)

// emojiWriter wraps an io.Writer and replaces :shortcode: patterns
// with their unicode emoji equivalents before writing.
type emojiWriter struct {
	w   io.Writer
	buf []byte // partial line buffer
}

func newEmojiWriter(w io.Writer) *emojiWriter {
	return &emojiWriter{w: w}
}

func (e *emojiWriter) Write(p []byte) (int, error) {
	e.buf = append(e.buf, p...)

	// Process complete lines.
	for {
		i := indexOf(e.buf, '\n')
		if i < 0 {
			break
		}
		line := string(e.buf[:i+1])
		e.buf = e.buf[i+1:]
		replaced := replaceEmoji(line)
		if _, err := io.WriteString(e.w, replaced); err != nil {
			return len(p), err
		}
	}
	return len(p), nil
}

func (e *emojiWriter) Flush() error {
	if len(e.buf) > 0 {
		replaced := replaceEmoji(string(e.buf))
		e.buf = nil
		_, err := io.WriteString(e.w, replaced)
		return err
	}
	return nil
}

func indexOf(b []byte, c byte) int {
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

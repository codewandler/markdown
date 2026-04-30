package main

import "strings"

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

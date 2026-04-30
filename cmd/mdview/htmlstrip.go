package main

import "strings"

// stripHTML removes common HTML tags from the input that don't render
// meaningfully in a terminal. Strips <div>...</div> wrappers,
// standalone block tags, and inline <br> tags.
func stripHTML(input string) string {
	if !strings.Contains(input, "<") {
		return input
	}

	var b strings.Builder
	b.Grow(len(input))
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Strip <div ...> and </div> tags.
		if matchTag(trimmed, "div") || trimmed == "</div>" {
			continue
		}

		// Strip standalone HTML tags that are just noise.
		if isNoiseTag(trimmed) {
			continue
		}

		// Strip inline <br>, <br/>, <br />
		line = stripInlineTags(line)

		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

func matchTag(s, tag string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "<"+tag) &&
		(len(s) > len(tag)+1 && (s[len(tag)+1] == ' ' || s[len(tag)+1] == '>'))
}

func containsAttr(tag, attr, val string) bool {
	lower := strings.ToLower(tag)
	return strings.Contains(lower, attr+"=\""+val+"\"") ||
		strings.Contains(lower, attr+"='"+val+"'")
}

func isNoiseTag(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	noiseTags := []string{
		"<div>", "</div>", "<div ", "<p>", "</p>",
		"<span>", "</span>", "<span ",
		"<center>", "</center>",
	}
	for _, tag := range noiseTags {
		if strings.HasPrefix(lower, tag) {
			return true
		}
	}
	return false
}

func stripInlineTags(s string) string {
	s = strings.ReplaceAll(s, "<br>", "")
	s = strings.ReplaceAll(s, "<br/>", "")
	s = strings.ReplaceAll(s, "<br />", "")
	return s
}

package html

import "strings"

// escapeHTML escapes &, <, >, and " for use in HTML content and
// attribute values. This covers the characters required by the
// CommonMark spec for safe HTML output.
func escapeHTML(s string) string {
	// Fast path: nothing to escape.
	if !strings.ContainsAny(s, "&<>\"") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/8)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// escapeURL percent-encodes a URL for use in href and src attributes.
// Already-encoded %XX sequences are preserved. Characters that are
// safe in URLs (unreserved + sub-delimiters + : / ? # [ ] @ ! $ & '
// ( ) * + , ; = %) are passed through unchanged.
func escapeURL(s string) string {
	// Fast path: nothing to encode.
	needsEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !urlSafe[c] {
			needsEscape = true
			break
		}
		// Bare % not followed by two hex digits needs encoding.
		if c == '%' && !(i+2 < len(s) && isHexDigit(s[i+1]) && isHexDigit(s[i+2])) {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '%' && i+2 < len(s) && isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
			// Preserve already-encoded %XX.
			b.WriteString(s[i : i+3])
			i += 2
			continue
		}
		if c == '%' {
			// Bare percent — encode it.
			b.WriteString("%25")
			continue
		}
		if urlSafe[c] {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0f])
		}
	}
	return b.String()
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

const hexDigits = "0123456789ABCDEF"

// urlSafe marks bytes that do not need percent-encoding in URLs.
// This includes unreserved characters (RFC 3986), sub-delimiters,
// and the general-delimiter characters used in URL structure.
var urlSafe = func() [256]bool {
	var t [256]bool
	// Unreserved: ALPHA / DIGIT / - . _ ~
	for c := 'a'; c <= 'z'; c++ {
		t[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		t[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		t[c] = true
	}
	for _, c := range "-._~" {
		t[c] = true
	}
	// Sub-delimiters: ! $ & ' ( ) * + , ; =
	for _, c := range "!$&'()*+,;=" {
		t[c] = true
	}
	// General delimiters used in URL structure: : / ? # @
	// Note: [ and ] are percent-encoded per CommonMark spec.
	for _, c := range ":/?#@" {
		t[c] = true
	}
	// Percent sign (handled specially for %XX preservation).
	t['%'] = true
	return t
}()

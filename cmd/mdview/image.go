package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/eliukblau/pixterm/pkg/ansimage"
	"golang.org/x/term"
)

// replaceAllImages finds ![alt](src) patterns in the input and replaces
// local image references with ANSI-rendered pixel art. Remote URLs are
// left as-is for the Markdown renderer to handle.
func replaceAllImages(input string, baseDir string) string {
	// Replace HTML <img> tags with rendered images.
	input = replaceHTMLImages(input, baseDir)

	if !strings.Contains(input, "![") {
		return input
	}

	var b strings.Builder
	b.Grow(len(input))
	i := 0
	for i < len(input) {
		// Look for ![
		idx := strings.Index(input[i:], "![")
		if idx < 0 {
			b.WriteString(input[i:])
			break
		}
		b.WriteString(input[i : i+idx])
		pos := i + idx

		// Parse ![alt](src)
		altEnd := strings.Index(input[pos+2:], "](")
		if altEnd < 0 {
			b.WriteString("![")
			i = pos + 2
			continue
		}
		altEnd += pos + 2
		srcEnd := strings.IndexByte(input[altEnd+2:], ')')
		if srcEnd < 0 {
			b.WriteString("![")
			i = pos + 2
			continue
		}
		srcEnd += altEnd + 2

		alt := input[pos+2 : altEnd]
		src := input[altEnd+2 : srcEnd]

		if !isURL(src) {
			rendered := renderImage(src, baseDir)
			if rendered != "" {
				b.WriteString(rendered)
			} else {
				b.WriteString(fmt.Sprintf("[image: %s]", alt))
			}
		} else {
			// Keep remote images as markdown for the renderer.
			b.WriteString(input[pos : srcEnd+1])
		}
		i = srcEnd + 1
	}
	return b.String()
}

// renderImage renders a local image file to ANSI colored blocks.
// Animated GIFs are skipped (they render poorly as static ANSI art).
func renderImage(src string, baseDir string) string {
	if !filepath.IsAbs(src) {
		src = filepath.Join(baseDir, src)
	}

	if _, err := os.Stat(src); err != nil {
		return ""
	}

	// Skip GIFs — they render poorly as static ANSI art.
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".gif" {
		return ""
	}

	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}

	if width < 2 {
		width = 80
	}
	// Use a large height so the image is only constrained by width.
	height := width * 2
	img, err := ansimage.NewScaledFromFile(src, height, width,
		color.Black, ansimage.ScaleModeFit, ansimage.NoDithering)
	if err != nil {
		return fmt.Sprintf("[image: %s (%v)]", src, err)
	}

	return img.Render()
}

// replaceHTMLImages finds <img src="..."> tags and replaces local
// image references with ANSI-rendered pixel art.
func replaceHTMLImages(input string, baseDir string) string {
	if !strings.Contains(input, "<img") {
		return input
	}
	var b strings.Builder
	b.Grow(len(input))
	i := 0
	for i < len(input) {
		idx := strings.Index(input[i:], "<img")
		if idx < 0 {
			b.WriteString(input[i:])
			break
		}
		b.WriteString(input[i : i+idx])
		pos := i + idx

		// Find the closing >
		end := strings.IndexByte(input[pos:], '>')
		if end < 0 {
			b.WriteString(input[pos:])
			break
		}
		tag := input[pos : pos+end+1]

		// Extract src attribute.
		src := extractAttr(tag, "src")
		if src != "" && !isURL(src) {
			rendered := renderImage(src, baseDir)
			if rendered != "" {
				b.WriteString(rendered)
			} else {
				alt := extractAttr(tag, "alt")
				if alt == "" {
					alt = src
				}
				b.WriteString(fmt.Sprintf("[image: %s]", alt))
			}
		} else {
			// Remote or no src — keep as-is.
			b.WriteString(tag)
		}
		i = pos + end + 1
	}
	return b.String()
}

// extractAttr extracts the value of an HTML attribute from a tag string.
func extractAttr(tag, attr string) string {
	// Look for attr="value" or attr='value'
	for _, q := range []byte{'"', '\''} {
		pattern := attr + "=" + string(q)
		idx := strings.Index(tag, pattern)
		if idx < 0 {
			continue
		}
		start := idx + len(pattern)
		end := strings.IndexByte(tag[start:], q)
		if end < 0 {
			continue
		}
		return tag[start : start+end]
	}
	return ""
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "ftp://")
}

package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/dolmen-go/kittyimg"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// replaceAllImages finds ![alt](src) and <img src="..."> patterns in the
// input and replaces local image references with kitty terminal graphics.
func replaceAllImages(input string, baseDir string) string {
	input = replaceHTMLImages(input, baseDir)

	if !strings.Contains(input, "![") {
		return input
	}

	var b strings.Builder
	b.Grow(len(input))
	i := 0
	for i < len(input) {
		idx := strings.Index(input[i:], "![")
		if idx < 0 {
			b.WriteString(input[i:])
			break
		}
		b.WriteString(input[i : i+idx])
		pos := i + idx

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
			b.WriteString(input[pos : srcEnd+1])
		}
		i = srcEnd + 1
	}
	return b.String()
}

// replaceHTMLImages finds <img src="..."> tags and replaces local
// image references with rendered output.
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

		end := strings.IndexByte(input[pos:], '>')
		if end < 0 {
			b.WriteString(input[pos:])
			break
		}
		tag := input[pos : pos+end+1]

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
			b.WriteString(tag)
		}
		i = pos + end + 1
	}
	return b.String()
}

// renderImage renders a local image file using the kitty terminal
// graphics protocol. Returns the escape sequence string, or "" on error.
func renderImage(src string, baseDir string) string {
	if !filepath.IsAbs(src) {
		src = filepath.Join(baseDir, src)
	}

	f, err := os.Open(src)
	if err != nil {
		return ""
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return ""
	}

	var b strings.Builder
	if err := kittyimg.Fprint(&b, img); err != nil {
		return ""
	}
	return b.String()
}

func extractAttr(tag, attr string) string {
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

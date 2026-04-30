package main

import (
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

type segment struct {
	content string
	isImage bool
}

// splitImages splits input into segments of markdown text and rendered images.
// Images are rendered separately so they bypass the Markdown parser.
func splitImages(input string, baseDir string) []segment {
	var segs []segment
	var cur strings.Builder

	flushText := func() {
		if cur.Len() > 0 {
			segs = append(segs, segment{content: cur.String()})
			cur.Reset()
		}
	}

	i := 0
	for i < len(input) {
		// Check for <img> tag.
		if strings.HasPrefix(input[i:], "<img") {
			end := strings.IndexByte(input[i:], '>')
			if end >= 0 {
				tag := input[i : i+end+1]
				src := extractAttr(tag, "src")
				if src != "" && !isURL(src) {
					rendered := renderImage(src, baseDir)
					if rendered != "" {
						flushText()
						segs = append(segs, segment{content: rendered + "\n", isImage: true})
						i += end + 1
						continue
					}
				}
			}
		}

		// Check for ![alt](src).
		if strings.HasPrefix(input[i:], "![") {
			altEnd := strings.Index(input[i+2:], "](")
			if altEnd >= 0 {
				altEnd += i + 2
				srcEnd := strings.IndexByte(input[altEnd+2:], ')')
				if srcEnd >= 0 {
					srcEnd += altEnd + 2
					src := input[altEnd+2 : srcEnd]
					if !isURL(src) {
						rendered := renderImage(src, baseDir)
						if rendered != "" {
							flushText()
							segs = append(segs, segment{content: rendered + "\n", isImage: true})
							i = srcEnd + 1
							continue
						}
					}
				}
			}
		}

		cur.WriteByte(input[i])
		i++
	}
	flushText()
	return segs
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

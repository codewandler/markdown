package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
				if src != "" {
					rendered := renderImage(src, baseDir)
					if rendered != "" {
						flushText()
						segs = append(segs, segment{content: rendered + "\n\n", isImage: true})
						i += end + 1
						continue
					}
				}
			}
		}

		// Check for [![alt](img-src)](link) — linked image (e.g. badges).
		if strings.HasPrefix(input[i:], "[![") {
			if end := matchLinkedImage(input[i:]); end > 0 {
				src := extractImageSrc(input[i+1 : i+end])
				if src != "" {
					rendered := renderImage(src, baseDir)
					if rendered != "" {
						flushText()
						segs = append(segs, segment{content: rendered + "\n\n", isImage: true})
						i += end
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
					rendered := renderImage(src, baseDir)
					if rendered != "" {
						flushText()
						segs = append(segs, segment{content: rendered + "\n\n", isImage: true})
						i = srcEnd + 1
						continue
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

// renderImage renders an image using the kitty terminal graphics protocol.
// Supports local files and HTTP/HTTPS URLs.
func renderImage(src string, baseDir string) string {
	var r io.ReadCloser
	if isURL(src) {
		// Rewrite shields.io SVG URLs to raster (PNG) equivalents.
		src = rewriteImageURL(src)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(src)
		if err != nil || resp.StatusCode != 200 {
			return ""
		}
		r = resp.Body
	} else {
		if !filepath.IsAbs(src) {
			src = filepath.Join(baseDir, src)
		}
		f, err := os.Open(src)
		if err != nil {
			return ""
		}
		r = f
	}
	defer r.Close()

	img, _, err := image.Decode(r)
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

// rewriteImageURL converts known SVG image URLs to raster equivalents.
func rewriteImageURL(u string) string {
	// shields.io: img.shields.io → raster.shields.io for PNG output.
	if strings.Contains(u, "img.shields.io") {
		return strings.Replace(u, "img.shields.io", "raster.shields.io", 1)
	}
	return u
}

// matchLinkedImage matches [![alt](img-src)](link-url) and returns
// the total length consumed, or 0 if no match.
func matchLinkedImage(s string) int {
	// s starts with "[!["
	// Find the inner ![alt](src) end.
	inner := strings.Index(s[1:], ")")
	if inner < 0 {
		return 0
	}
	inner += 2 // position after the inner )
	// Expect ](link)
	if inner >= len(s) || s[inner] != ']' {
		return 0
	}
	if inner+1 >= len(s) || s[inner+1] != '(' {
		return 0
	}
	linkEnd := strings.IndexByte(s[inner+2:], ')')
	if linkEnd < 0 {
		return 0
	}
	return inner + 2 + linkEnd + 1
}

// extractImageSrc extracts the src from ![alt](src).
func extractImageSrc(s string) string {
	if !strings.HasPrefix(s, "![") {
		return ""
	}
	altEnd := strings.Index(s[2:], "](")
	if altEnd < 0 {
		return ""
	}
	altEnd += 2
	srcEnd := strings.IndexByte(s[altEnd+2:], ')')
	if srcEnd < 0 {
		return ""
	}
	return s[altEnd+2 : altEnd+2+srcEnd]
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "ftp://")
}

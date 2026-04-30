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

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "ftp://")
}

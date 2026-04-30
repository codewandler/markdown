package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/codewandler/markdown/stream"
)

type fileRefScanner struct{}

func (fileRefScanner) TriggerBytes() string {
	return "./abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
}

func (fileRefScanner) ScanInline(input string, _ stream.InlineContext) (stream.InlineScanResult, bool) {
	consume, path, line, col, ok := scanFileReference(input)
	if !ok {
		return stream.InlineScanResult{}, false
	}
	source := input[:consume]
	attrs := []stream.Attribute{
		{Key: "path", Value: path},
		{Key: "line", Value: strconv.Itoa(line)},
	}
	if col > 0 {
		attrs = append(attrs, stream.Attribute{Key: "column", Value: strconv.Itoa(col)})
	}
	return stream.InlineScanResult{
		Consume: consume,
		Event: stream.Event{
			Kind: stream.EventInline,
			Inline: &stream.InlineData{
				Type:         "file-ref",
				Source:       source,
				Text:         source,
				DisplayWidth: utf8.RuneCountInString(source),
				Attrs:        attrs,
			},
		},
	}, true
}

func scanFileReference(input string) (consume int, path string, line int, col int, ok bool) {
	if input == "" || !isFileRefStart(input[0]) {
		return 0, "", 0, 0, false
	}
	pathEnd := 0
	for pathEnd < len(input) {
		c := input[pathEnd]
		if isFileRefPathByte(c) {
			pathEnd++
			continue
		}
		break
	}
	if pathEnd == 0 || pathEnd >= len(input) || input[pathEnd] != ':' {
		return 0, "", 0, 0, false
	}
	path = input[:pathEnd]
	if !looksLikeFilePath(path) {
		return 0, "", 0, 0, false
	}
	lineStart := pathEnd + 1
	lineEnd := scanDigits(input, lineStart)
	if lineEnd == lineStart {
		return 0, "", 0, 0, false
	}
	parsedLine, err := strconv.Atoi(input[lineStart:lineEnd])
	if err != nil || parsedLine <= 0 {
		return 0, "", 0, 0, false
	}
	line = parsedLine
	consume = lineEnd
	if consume < len(input) && input[consume] == ':' {
		colStart := consume + 1
		colEnd := scanDigits(input, colStart)
		if colEnd > colStart {
			parsedCol, err := strconv.Atoi(input[colStart:colEnd])
			if err == nil && parsedCol > 0 {
				col = parsedCol
				consume = colEnd
			}
		}
	}
	if consume < len(input) {
		rn, _ := utf8.DecodeRuneInString(input[consume:])
		if isFileRefTrailingRune(rn) {
			return 0, "", 0, 0, false
		}
	}
	return consume, path, line, col, true
}

func scanDigits(input string, start int) int {
	for start < len(input) && input[start] >= '0' && input[start] <= '9' {
		start++
	}
	return start
}

func isFileRefStart(c byte) bool {
	return c == '.' || c == '/' || c == '_' || isASCIIAlphaNum(c)
}

func isFileRefPathByte(c byte) bool {
	return isASCIIAlphaNum(c) || c == '_' || c == '-' || c == '.' || c == '/' || c == '\\'
}

func isASCIIAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func looksLikeFilePath(path string) bool {
	if path == "" || strings.HasSuffix(path, ".") || strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\") {
		return false
	}
	base := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	if base == "." || base == "" || !strings.Contains(base, ".") {
		return false
	}
	return true
}

func isFileRefTrailingRune(rn rune) bool {
	return rn == '_' || rn == '-' || rn == '/' || rn == '\\' || unicode.IsLetter(rn) || unicode.IsDigit(rn)
}

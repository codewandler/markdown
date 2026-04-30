package main

import (
	"strings"

	"github.com/codewandler/markdown/stream"
)

type emojiScanner struct{}

func (emojiScanner) TriggerBytes() string { return ":" }

func (emojiScanner) ScanInline(input string, _ stream.InlineContext) (stream.InlineScanResult, bool) {
	if len(input) < 3 || input[0] != ':' {
		return stream.InlineScanResult{}, false
	}
	end := strings.IndexByte(input[1:], ':')
	if end < 0 {
		return stream.InlineScanResult{}, false
	}
	name := input[1 : 1+end]
	emoji, ok := emojiShortcodes[name]
	if !ok {
		return stream.InlineScanResult{}, false
	}
	source := input[:len(name)+2]
	return stream.InlineScanResult{
		Consume: len(source),
		Event: stream.Event{
			Kind: stream.EventInline,
			Inline: &stream.InlineData{
				Type:         "emoji",
				Source:       source,
				Text:         emoji,
				DisplayWidth: 2,
			},
		},
	}, true
}

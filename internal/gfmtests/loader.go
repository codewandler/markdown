package gfmtests

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

const Version = "0.29"

//go:embed testdata/gfm-0.29.json
var corpus029 []byte

// Example is one GFM specification example.
type Example struct {
	Markdown  string `json:"markdown"`
	HTML      string `json:"html"`
	Example   int    `json:"example"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Section   string `json:"section"`
	Extension string `json:"extension"` // "table", "strikethrough", "autolink", "tagfilter", "disabled", or ""
}

// Load returns the pinned GFM test corpus.
func Load() ([]Example, error) {
	return Decode(corpus029)
}

// Decode parses GFM JSON test data.
func Decode(data []byte) ([]Example, error) {
	var examples []Example
	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("decode GFM corpus: %w", err)
	}
	for i, ex := range examples {
		if ex.Example <= 0 {
			return nil, fmt.Errorf("GFM corpus item %d has invalid example number %d", i, ex.Example)
		}
		if ex.Section == "" {
			return nil, fmt.Errorf("GFM example %d has empty section", ex.Example)
		}
	}
	return examples, nil
}

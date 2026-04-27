package commonmarktests

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

const Version = "0.31.2"

//go:embed testdata/commonmark-0.31.2.json
var corpus0312 []byte

// Example is one CommonMark specification example.
type Example struct {
	Markdown  string `json:"markdown"`
	HTML      string `json:"html"`
	Example   int    `json:"example"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Section   string `json:"section"`
}

// Load returns the pinned CommonMark test corpus.
func Load() ([]Example, error) {
	return Decode(corpus0312)
}

// Decode parses CommonMark JSON test data.
func Decode(data []byte) ([]Example, error) {
	var examples []Example
	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("decode CommonMark corpus: %w", err)
	}
	for i, ex := range examples {
		if ex.Example <= 0 {
			return nil, fmt.Errorf("CommonMark corpus item %d has invalid example number %d", i, ex.Example)
		}
		if ex.Section == "" {
			return nil, fmt.Errorf("CommonMark example %d has empty section", ex.Example)
		}
		if ex.StartLine <= 0 || ex.EndLine < ex.StartLine {
			return nil, fmt.Errorf("CommonMark example %d has invalid source lines %d..%d", ex.Example, ex.StartLine, ex.EndLine)
		}
	}
	return examples, nil
}

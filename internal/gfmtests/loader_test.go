package gfmtests

import "testing"

func TestLoad(t *testing.T) {
	examples, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(examples) == 0 {
		t.Fatal("no examples loaded")
	}
	t.Logf("loaded %d GFM %s examples", len(examples), Version)

	// Verify extension tags are present.
	extensions := map[string]int{}
	for _, ex := range examples {
		if ex.Extension != "" {
			extensions[ex.Extension]++
		}
	}
	for _, ext := range []string{"table", "strikethrough", "autolink"} {
		if extensions[ext] == 0 {
			t.Errorf("no examples with extension %q", ext)
		}
	}
	t.Logf("extensions: %v", extensions)
}

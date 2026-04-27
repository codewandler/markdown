package commonmarktests

import "testing"

func TestLoad(t *testing.T) {
	examples, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(examples) != 652 {
		t.Fatalf("expected 652 CommonMark %s examples, got %d", Version, len(examples))
	}
	first := examples[0]
	if first.Example != 1 || first.Section != "Tabs" {
		t.Fatalf("unexpected first example: %#v", first)
	}
	last := examples[len(examples)-1]
	if last.Example != 652 || last.Section != "Textual content" {
		t.Fatalf("unexpected last example: %#v", last)
	}
}

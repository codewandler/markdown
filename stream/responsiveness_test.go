package stream

import "testing"

func TestFencedCodeEmitsBeforeClosingFence(t *testing.T) {
	p := NewParser()
	events, err := p.Write([]byte("```go\nfmt.Println(\"one\")\nfmt.Println(\"two\")\n"))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, viewEvents(events), eventView{Kind: EventEnterBlock, Block: BlockFencedCode, Info: "go"})
	assertContains(t, viewEvents(events), eventView{Kind: EventText, Text: "fmt.Println(\"one\")"})
	assertContains(t, viewEvents(events), eventView{Kind: EventText, Text: "fmt.Println(\"two\")"})
}

func TestParagraphEmitsAtBlankLineBoundary(t *testing.T) {
	p := NewParser()
	if events, err := p.Write([]byte("alpha\nbeta\n")); err != nil {
		t.Fatal(err)
	} else if len(events) != 0 {
		t.Fatalf("paragraph emitted before boundary: %#v", events)
	}

	events, err := p.Write([]byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, viewEvents(events), eventView{Kind: EventEnterBlock, Block: BlockParagraph})
	assertContains(t, viewEvents(events), eventView{Kind: EventText, Text: "alpha"})
	assertContains(t, viewEvents(events), eventView{Kind: EventText, Text: "beta"})
}

func TestParagraphEmitsAtInterruptingBlock(t *testing.T) {
	p := NewParser()
	if events, err := p.Write([]byte("alpha\n")); err != nil {
		t.Fatal(err)
	} else if len(events) != 0 {
		t.Fatalf("paragraph emitted before boundary: %#v", events)
	}

	events, err := p.Write([]byte("# heading\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := viewEvents(events)
	assertContains(t, got, eventView{Kind: EventEnterBlock, Block: BlockParagraph})
	assertContains(t, got, eventView{Kind: EventText, Text: "alpha"})
	assertContains(t, got, eventView{Kind: EventEnterBlock, Block: BlockHeading, Level: 1})
	assertContains(t, got, eventView{Kind: EventText, Text: "heading"})
}

func TestIncompleteLineDoesNotEmitBeforeNewline(t *testing.T) {
	p := NewParser()
	if events, err := p.Write([]byte("# heading")); err != nil {
		t.Fatal(err)
	} else if len(events) != 0 {
		t.Fatalf("incomplete line emitted events: %#v", events)
	}

	events, err := p.Write([]byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, viewEvents(events), eventView{Kind: EventEnterBlock, Block: BlockHeading, Level: 1})
	assertContains(t, viewEvents(events), eventView{Kind: EventText, Text: "heading"})
}

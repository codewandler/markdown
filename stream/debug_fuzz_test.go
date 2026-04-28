package stream

import (
	"fmt"
	"testing"
)

func TestDebugFuzzFinding5(t *testing.T) {
	input := ">* *\n\n0"
	p := NewParser().(*parser)
	
	events, err := p.Write([]byte(input))
	if err != nil { t.Fatal(err) }
	fmt.Println("=== After Write ===")
	for i, ev := range events { fmt.Printf("  %d: %s %s text=%q\n", i, ev.Kind, ev.Block, ev.Text) }
	fmt.Printf("State: bq=%v inList=%v inListItem=%v stack=%d bqInside=%v blankLine=%v\n",
		p.inBlockquote, p.inList, p.inListItem, len(p.listStack), p.bqInsideListItem, p.listItemBlankLine)
	
	events, err = p.Flush()
	if err != nil { t.Fatal(err) }
	fmt.Println("=== After Flush ===")
	for i, ev := range events { fmt.Printf("  %d: %s %s text=%q\n", i, ev.Kind, ev.Block, ev.Text) }
	fmt.Printf("State: bq=%v inList=%v inListItem=%v stack=%d\n",
		p.inBlockquote, p.inList, p.inListItem, len(p.listStack))
}

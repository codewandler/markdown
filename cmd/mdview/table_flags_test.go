package main

import (
	"reflect"
	"testing"

	"github.com/codewandler/markdown/terminal"
)

func TestParseTableLayoutBuffered(t *testing.T) {
	layout, err := parseTableLayout("buffered", "", "ellipsis", 0)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Mode != terminal.TableModeBuffered {
		t.Fatalf("mode: got %v", layout.Mode)
	}
}

func TestParseTableLayoutFixed(t *testing.T) {
	layout, err := parseTableLayout("fixed", "16, 8,32", "clip", 0)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Mode != terminal.TableModeFixedWidth {
		t.Fatalf("mode: got %v", layout.Mode)
	}
	if !reflect.DeepEqual(layout.ColumnWidths, []int{16, 8, 32}) {
		t.Fatalf("widths: %#v", layout.ColumnWidths)
	}
	if layout.Overflow != terminal.TableOverflowClip {
		t.Fatalf("overflow: got %v", layout.Overflow)
	}
}

func TestParseTableLayoutAuto(t *testing.T) {
	layout, err := parseTableLayout("auto", "", "ellipsis", 100)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Mode != terminal.TableModeAutoWidth {
		t.Fatalf("mode: got %v", layout.Mode)
	}
	if layout.MaxWidth != 100 {
		t.Fatalf("max width: got %d", layout.MaxWidth)
	}
	if layout.Overflow != terminal.TableOverflowEllipsis {
		t.Fatalf("overflow: got %v", layout.Overflow)
	}
}

func TestParseTableLayoutRejectsFixedWithoutWidths(t *testing.T) {
	if _, err := parseTableLayout("fixed", "", "ellipsis", 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTableLayoutRejectsInvalidWidth(t *testing.T) {
	if _, err := parseTableLayout("fixed", "10,nope", "ellipsis", 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTableLayoutRejectsInvalidOverflow(t *testing.T) {
	if _, err := parseTableLayout("auto", "", "truncate", 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTableLayoutRejectsWidthsWithAuto(t *testing.T) {
	if _, err := parseTableLayout("auto", "10", "ellipsis", 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTableLayoutRejectsUnknownMode(t *testing.T) {
	if _, err := parseTableLayout("ragged", "", "ellipsis", 0); err == nil {
		t.Fatal("expected error")
	}
}

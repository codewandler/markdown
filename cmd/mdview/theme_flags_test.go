package main

import (
	"reflect"
	"testing"

	"github.com/codewandler/markdown/terminal"
)

func TestParseThemeMonokai(t *testing.T) {
	theme, err := parseTheme("monokai")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(theme, terminal.DefaultTheme()) {
		t.Fatalf("theme = %#v, want default", theme)
	}
}

func TestParseThemePlain(t *testing.T) {
	theme, err := parseTheme("plain")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(theme, terminal.NoColorTheme()) {
		t.Fatalf("theme = %#v, want no-color", theme)
	}
}

func TestParseThemeRejectsUnknown(t *testing.T) {
	if _, err := parseTheme("dracula"); err == nil {
		t.Fatal("expected error")
	}
}

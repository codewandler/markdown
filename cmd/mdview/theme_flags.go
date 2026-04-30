package main

import (
	"fmt"
	"strings"

	"github.com/codewandler/markdown/terminal"
)

func parseTheme(name string) (terminal.Theme, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "monokai", "default":
		return terminal.DefaultTheme(), nil
	case "nord":
		return terminal.NordTheme(), nil
	case "plain", "none", "no-color":
		return terminal.NoColorTheme(), nil
	default:
		return terminal.Theme{}, fmt.Errorf("invalid --theme %q: want monokai, nord, or plain", name)
	}
}

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codewandler/markdown/terminal"
)

func parseTableLayout(mode, widths, overflow string, maxWidth int) (terminal.TableLayout, error) {
	overflowMode, err := parseTableOverflow(overflow)
	if err != nil {
		return terminal.TableLayout{}, err
	}
	switch mode {
	case "", "buffered":
		if strings.TrimSpace(widths) != "" {
			return terminal.TableLayout{}, fmt.Errorf("-table-widths requires -table-mode fixed")
		}
		if maxWidth != 0 {
			return terminal.TableLayout{}, fmt.Errorf("-table-max-width requires -table-mode auto")
		}
		return terminal.TableLayout{Mode: terminal.TableModeBuffered}, nil
	case "fixed":
		parsedWidths, err := parseTableWidths(widths)
		if err != nil {
			return terminal.TableLayout{}, err
		}
		if len(parsedWidths) == 0 {
			return terminal.TableLayout{}, fmt.Errorf("-table-widths is required when -table-mode fixed")
		}
		if maxWidth != 0 {
			return terminal.TableLayout{}, fmt.Errorf("-table-max-width cannot be used with -table-mode fixed")
		}
		return terminal.TableLayout{
			Mode:         terminal.TableModeFixedWidth,
			ColumnWidths: parsedWidths,
			Overflow:     overflowMode,
		}, nil
	case "auto":
		if strings.TrimSpace(widths) != "" {
			return terminal.TableLayout{}, fmt.Errorf("-table-widths cannot be used with -table-mode auto")
		}
		if maxWidth < 0 {
			return terminal.TableLayout{}, fmt.Errorf("-table-max-width must be >= 0")
		}
		return terminal.TableLayout{
			Mode:     terminal.TableModeAutoWidth,
			MaxWidth: maxWidth,
			Overflow: overflowMode,
		}, nil
	default:
		return terminal.TableLayout{}, fmt.Errorf("invalid -table-mode %q: want buffered, fixed, or auto", mode)
	}
}

func parseTableOverflow(overflow string) (terminal.TableOverflow, error) {
	switch overflow {
	case "", "ellipsis":
		return terminal.TableOverflowEllipsis, nil
	case "clip":
		return terminal.TableOverflowClip, nil
	default:
		return terminal.TableOverflowEllipsis, fmt.Errorf("invalid -table-overflow %q: want ellipsis or clip", overflow)
	}
}

func parseTableWidths(s string) ([]int, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	widths := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty table width in %q", s)
		}
		width, err := strconv.Atoi(part)
		if err != nil || width <= 0 {
			return nil, fmt.Errorf("invalid table width %q", part)
		}
		widths = append(widths, width)
	}
	return widths, nil
}

// Command benchcompare reads Go benchmark output from stdin and
// produces Markdown comparison tables with speedup ratios.
//
// Usage:
//
//	go test -bench=BenchmarkRender -benchmem | go run ./cmd/benchcompare
//	go test -bench=BenchmarkParse -benchmem | go run ./cmd/benchcompare --baseline goldmark
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type result struct {
	group    string // e.g. "Render_Spec"
	variant  string // e.g. "ours", "glamour"
	nsOp     float64
	bOp      int64
	allocsOp int64
}

var benchLine = regexp.MustCompile(
	`^(Benchmark\S+)-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`,
)

func main() {
	baseline := flag.String("baseline", "ours", "baseline variant for ratio calculation")
	flag.Parse()

	var results []result
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		m := benchLine.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		name := m[1]
		nsOp, _ := strconv.ParseFloat(m[2], 64)
		bOp, _ := strconv.ParseInt(m[3], 10, 64)
		allocsOp, _ := strconv.ParseInt(m[4], 10, 64)

		// Split "BenchmarkRender_Spec/ours" into group="Render_Spec", variant="ours"
		name = strings.TrimPrefix(name, "Benchmark")
		parts := strings.SplitN(name, "/", 2)
		group := parts[0]
		variant := ""
		if len(parts) > 1 {
			variant = parts[1]
		}

		results = append(results, result{
			group:    group,
			variant:  variant,
			nsOp:     nsOp,
			bOp:      bOp,
			allocsOp: allocsOp,
		})
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "no benchmark results found on stdin")
		os.Exit(1)
	}

	// Group results.
	type groupData struct {
		name     string
		variants map[string]result
	}
	groupMap := map[string]*groupData{}
	var groupOrder []string
	for _, r := range results {
		g, ok := groupMap[r.group]
		if !ok {
			g = &groupData{name: r.group, variants: map[string]result{}}
			groupMap[r.group] = g
			groupOrder = append(groupOrder, r.group)
		}
		g.variants[r.variant] = r
	}

	// Collect all variant names.
	variantSet := map[string]bool{}
	for _, g := range groupMap {
		for v := range g.variants {
			variantSet[v] = true
		}
	}
	var variants []string
	for v := range variantSet {
		variants = append(variants, v)
	}
	sort.Strings(variants)
	// Move baseline to front.
	for i, v := range variants {
		if v == *baseline {
			variants = append(variants[:i], variants[i+1:]...)
			variants = append([]string{*baseline}, variants...)
			break
		}
	}

	// Print table.
	fmt.Println()
	fmt.Println("### Speed (ns/op, lower is better)")
	fmt.Println()

	// Header.
	header := "| Input |"
	sep := "| --- |"
	for _, v := range variants {
		header += fmt.Sprintf(" %s |", v)
		sep += " ---: |"
	}
	if len(variants) > 1 {
		header += " vs best |"
		sep += " ---: |"
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, gName := range groupOrder {
		g := groupMap[gName]
		row := fmt.Sprintf("| %s |", g.name)
		baseNs := float64(0)
		if br, ok := g.variants[*baseline]; ok {
			baseNs = br.nsOp
		}
		bestNs := math.MaxFloat64
		for _, r := range g.variants {
			if r.nsOp < bestNs {
				bestNs = r.nsOp
			}
		}
		for _, v := range variants {
			if r, ok := g.variants[v]; ok {
				row += fmt.Sprintf(" %s |", formatNs(r.nsOp))
			} else {
				row += " - |"
			}
		}
		if len(variants) > 1 {
			row += " " + ratioVsBest(baseNs, bestNs, "faster", "slower") + " |"
		}
		fmt.Println(row)
	}

	// Allocations table.
	fmt.Println()
	fmt.Println("### Allocations (allocs/op, lower is better)")
	fmt.Println()

	header = "| Input |"
	sep = "| --- |"
	for _, v := range variants {
		header += fmt.Sprintf(" %s |", v)
		sep += " ---: |"
	}
	if len(variants) > 1 {
		header += " vs best |"
		sep += " ---: |"
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, gName := range groupOrder {
		g := groupMap[gName]
		baseAllocs := float64(0)
		if br, ok := g.variants[*baseline]; ok {
			baseAllocs = float64(br.allocsOp)
		}
		bestAllocs := float64(math.MaxInt64)
		for _, r := range g.variants {
			if float64(r.allocsOp) < bestAllocs {
				bestAllocs = float64(r.allocsOp)
			}
		}
		row := fmt.Sprintf("| %s |", g.name)
		for _, v := range variants {
			if r, ok := g.variants[v]; ok {
				row += fmt.Sprintf(" %s |", formatInt(r.allocsOp))
			} else {
				row += " - |"
			}
		}
		if len(variants) > 1 {
			row += " " + ratioVsBest(baseAllocs, bestAllocs, "fewer", "more") + " |"
		}
		fmt.Println(row)
	}

	// Memory table.
	fmt.Println()
	fmt.Println("### Memory (B/op, lower is better)")
	fmt.Println()

	header = "| Input |"
	sep = "| --- |"
	for _, v := range variants {
		header += fmt.Sprintf(" %s |", v)
		sep += " ---: |"
	}
	if len(variants) > 1 {
		header += " vs best |"
		sep += " ---: |"
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, gName := range groupOrder {
		g := groupMap[gName]
		baseBop := float64(0)
		if br, ok := g.variants[*baseline]; ok {
			baseBop = float64(br.bOp)
		}
		bestBop := float64(math.MaxInt64)
		for _, r := range g.variants {
			if float64(r.bOp) < bestBop {
				bestBop = float64(r.bOp)
			}
		}
		row := fmt.Sprintf("| %s |", g.name)
		for _, v := range variants {
			if r, ok := g.variants[v]; ok {
				row += fmt.Sprintf(" %s |", formatBytes(r.bOp))
			} else {
				row += " - |"
			}
		}
		if len(variants) > 1 {
			row += " " + ratioVsBest(baseBop, bestBop, "less", "more") + " |"
		}
		fmt.Println(row)
	}
	fmt.Println()
}

func ratioVsBest(value, best float64, winWord, loseWord string) string {
	if best <= 0 {
		return "-"
	}
	if value == best {
		return "**best**"
	}
	if value < best {
		// We're better than "best" (shouldn't happen, but handle it)
		ratio := best / value
		return fmt.Sprintf("**%.1fx %s**", ratio, winWord)
	}
	ratio := value / best
	return fmt.Sprintf("%.1fx %s", ratio, loseWord)
}

func formatNs(ns float64) string {
	switch {
	case ns >= 1e9:
		return fmt.Sprintf("%.2fs", ns/1e9)
	case ns >= 1e6:
		return fmt.Sprintf("%.1fms", ns/1e6)
	case ns >= 1e3:
		return fmt.Sprintf("%.1fus", ns/1e3)
	default:
		return fmt.Sprintf("%.0fns", ns)
	}
}

func formatInt(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	}
	return fmt.Sprintf("%d", n)
}

func formatBytes(b int64) string {
	if b >= 1<<30 {
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	}
	if b >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	}
	if b >= 1<<10 {
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	}
	return fmt.Sprintf("%d B", b)
}

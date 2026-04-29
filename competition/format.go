package competition

import "fmt"

// FormatNs formats nanoseconds into a human-readable duration string.
func FormatNs(ns float64) string {
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

// FormatCount formats a count (allocations, etc.) with K/M suffixes.
func FormatCount(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	}
	return fmt.Sprintf("%d", n)
}

// FormatBytes formats a byte count with KB/MB/GB suffixes.
func FormatBytes(b int64) string {
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

// FormatPct formats a percentage with one decimal place.
func FormatPct(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}

// FormatRatio computes and formats the ratio between value and baseline.
// Lower is better: if value < baseline, it's "Nx faster/fewer/less".
// If value == baseline, it's "**best**".
// If value > baseline, it's "Nx slower/more".
func FormatRatio(value, baseline float64, winWord, loseWord string) string {
	if baseline <= 0 || value <= 0 {
		return "-"
	}
	if value == baseline {
		return "**best**"
	}
	if value < baseline {
		ratio := baseline / value
		return fmt.Sprintf("**%.1fx %s**", ratio, winWord)
	}
	ratio := value / baseline
	return fmt.Sprintf("%.1fx %s", ratio, loseWord)
}

// BoldBest wraps s in bold if isBest is true.
func BoldBest(s string, isBest bool) string {
	if isBest {
		return "**" + s + "**"
	}
	return s
}

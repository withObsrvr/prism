package gateway

import (
	"fmt"
	"strings"
	"time"
)

// FormatNumber formats an integer with thousand separators: 1284392 → "1,284,392"
func FormatNumber(n int64) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatAbbrev formats a number with abbreviation: 1284392 → "1.28M"
func FormatAbbrev(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// FormatAge returns a human-readable age string: "3.8s ago", "2 min ago", etc.
func FormatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.1fs ago", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// ShortHash truncates a hex hash: "8f2a7c1b3c4d5e6f" → "8f2a7c...5e6f"
func ShortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:6] + "..." + h[len(h)-4:]
}

// ShortAddress truncates a Stellar address: "GABC7DEF..." → "GABC...MNOP"
func ShortAddress(a string) string {
	if len(a) <= 12 {
		return a
	}
	return a[:4] + "..." + a[len(a)-4:]
}

// FormatStroops formats stroops as a comma-separated number string.
func FormatStroops(s int64) string {
	return FormatNumber(s)
}

// FormatXLM converts stroops to XLM with 7 decimal places.
func FormatXLM(stroops int64) string {
	xlm := float64(stroops) / 10_000_000
	return fmt.Sprintf("%.7f", xlm)
}

// FormatCloseTime formats a close time in seconds.
func FormatCloseTime(seconds float64) string {
	return fmt.Sprintf("%.1fs", seconds)
}

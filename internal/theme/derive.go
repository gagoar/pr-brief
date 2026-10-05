package theme

import (
	"math"
	"strconv"
)

// MIX holds the percentages beautiful-mermaid mixes the foreground into the
// background with when a colour is not given (src/theme.ts, MIX).
var MIX = struct{ TextSec, Line, Arrow, NodeFill, NodeStroke, GroupHeader, InnerStroke, KeyBadge int }{
	TextSec: 60, Line: 50, Arrow: 85, NodeFill: 3, NodeStroke: 20, GroupHeader: 5, InnerStroke: 12, KeyBadge: 10,
}

func rgb(h string) [3]int {
	var out [3]int
	for i := 0; i < 3; i++ {
		v, _ := strconv.ParseInt(h[1+2*i:3+2*i], 16, 32)
		out[i] = int(v)
	}
	return out
}

func hex6(c [3]int) string {
	const digits = "0123456789abcdef"
	b := []byte{'#', 0, 0, 0, 0, 0, 0}
	for i, v := range c {
		b[1+2*i], b[2+2*i] = digits[v>>4], digits[v&15]
	}
	return string(b)
}

// Mix is CSS color-mix(in srgb, fg pct%, bg) for opaque colours.
func Mix(fg, bg string, pct int) string {
	a, b := rgb(fg), rgb(bg)
	var out [3]int
	for i := range out {
		v := float64(a[i])*float64(pct)/100 + float64(b[i])*float64(100-pct)/100
		out[i] = int(math.RoundToEven(v))
	}
	return hex6(out)
}

// Luminance is WCAG relative luminance.
func Luminance(h string) float64 {
	ch := func(v int) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	c := rgb(h)
	return 0.2126*ch(c[0]) + 0.7152*ch(c[1]) + 0.0722*ch(c[2])
}

// Contrast is the WCAG contrast ratio of two colours.
func Contrast(a, b string) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Derived holds every colour the renderer uses, computed the way beautiful-mermaid does.
type Derived struct {
	BG, FG, TextSec, Line, Arrow, NodeFill, NodeStroke, GroupHeader, InnerStroke string
	Dark                                                                         bool
	Added, Modified, Removed                                                     string
}

// Derive fills the optional colours from the two required ones.
func (t Theme) Derive() Derived {
	or := func(v, fallback string) string {
		if v != "" {
			return v
		}
		return fallback
	}
	bg, fg := t.BG, t.FG
	d := Derived{
		BG: bg, FG: fg,
		TextSec:     or(t.Muted, Mix(fg, bg, MIX.TextSec)),
		Line:        or(t.Line, Mix(fg, bg, MIX.Line)),
		Arrow:       or(t.Accent, Mix(fg, bg, MIX.Arrow)),
		NodeFill:    or(t.Surface, Mix(fg, bg, MIX.NodeFill)),
		NodeStroke:  or(t.Border, Mix(fg, bg, MIX.NodeStroke)),
		GroupHeader: Mix(fg, bg, MIX.GroupHeader),
		InnerStroke: Mix(fg, bg, MIX.InnerStroke),
		Dark:        Luminance(bg) < 0.4,
	}
	s := t.Status
	if s == nil {
		if d.Dark {
			s = &statusOnDark
		} else {
			s = &statusOnLight
		}
	}
	d.Added, d.Modified, d.Removed = s.Added, s.Modified, s.Removed
	return d
}

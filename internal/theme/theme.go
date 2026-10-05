// Package theme holds the diagram themes of pr-brief and turns a theme into
// the Mermaid style lines (init line, classDef, linkStyle) that GitHub renders.
//
// The colour model, the mixing rule and the style constants come from
// lukilabs/beautiful-mermaid (MIT, Copyright (c) 2026 Craft Docs). See UPSTREAM.md.
package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// MaxFileBytes is the size limit of a custom theme file.
const MaxFileBytes = 8 * 1024

// MinContrast is the WCAG ratio a custom theme's text must reach.
const MinContrast = 4.5

// Colors is beautiful-mermaid's DiagramColors: two required colours and five optional ones.
type Colors struct {
	BG      string `json:"bg"`
	FG      string `json:"fg"`
	Line    string `json:"line,omitempty"`
	Accent  string `json:"accent,omitempty"`
	Muted   string `json:"muted,omitempty"`
	Surface string `json:"surface,omitempty"`
	Border  string `json:"border,omitempty"`
}

// Status holds the three node-state colours. beautiful-mermaid has none, so every
// theme carries them (built-in themes from their editor palette, custom themes tuned
// by background lightness unless the file says otherwise).
type Status struct {
	Added    string `json:"added"`
	Modified string `json:"modified"`
	Removed  string `json:"removed"`
}

// Theme is a named set of colours.
type Theme struct {
	Name string `json:"-"`
	Colors
	Status *Status `json:"status,omitempty"`
	Schema string  `json:"$schema,omitempty"`
}

var hexRe = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// normHex lower-cases a hex colour and expands #rgb. GitHub only accepts hex.
func normHex(field, v string) (string, error) {
	if !hexRe.MatchString(v) {
		return "", fmt.Errorf("%s %q must be a hex colour such as #1a2b3c", field, v)
	}
	v = strings.ToLower(v)
	if len(v) == 4 {
		v = "#" + string(v[1]) + string(v[1]) + string(v[2]) + string(v[2]) + string(v[3]) + string(v[3])
	}
	return v, nil
}

// Parse reads a custom theme file. Unknown keys are errors.
func Parse(data []byte, name string) (Theme, error) {
	if len(data) > MaxFileBytes {
		return Theme{}, fmt.Errorf("theme file is %d bytes; the limit is %d", len(data), MaxFileBytes)
	}
	var t Theme
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return Theme{}, fmt.Errorf("invalid theme: %w", err)
	}
	if dec.More() {
		return Theme{}, errors.New("invalid theme: trailing data after the JSON value")
	}
	t.Name = name
	if err := t.normalise(); err != nil {
		return Theme{}, err
	}
	if err := t.checkContrast(); err != nil {
		return Theme{}, err
	}
	return t, nil
}

func (t *Theme) normalise() error {
	if t.BG == "" || t.FG == "" {
		return errors.New("invalid theme: bg and fg are required")
	}
	fields := []struct {
		name string
		p    *string
		opt  bool
	}{
		{"bg", &t.BG, false}, {"fg", &t.FG, false}, {"line", &t.Line, true}, {"accent", &t.Accent, true},
		{"muted", &t.Muted, true}, {"surface", &t.Surface, true}, {"border", &t.Border, true},
	}
	for _, f := range fields {
		if f.opt && *f.p == "" {
			continue
		}
		v, err := normHex(f.name, *f.p)
		if err != nil {
			return fmt.Errorf("invalid theme: %w", err)
		}
		*f.p = v
	}
	if t.Status != nil {
		for _, s := range []struct {
			name string
			p    *string
		}{{"status.added", &t.Status.Added}, {"status.modified", &t.Status.Modified}, {"status.removed", &t.Status.Removed}} {
			if *s.p == "" {
				return fmt.Errorf("invalid theme: %s is required when status is set", s.name)
			}
			v, err := normHex(s.name, *s.p)
			if err != nil {
				return fmt.Errorf("invalid theme: %w", err)
			}
			*s.p = v
		}
	}
	return nil
}

// checkContrast rejects a custom theme whose text cannot be read.
func (t Theme) checkContrast() error {
	d := t.Derive()
	for _, c := range []struct{ what, bg string }{{"the background", d.BG}, {"a node fill", d.NodeFill}} {
		if r := Contrast(d.FG, c.bg); r < MinContrast {
			return fmt.Errorf("invalid theme: fg %s on %s %s has contrast %.2f:1; it needs at least %.1f:1", d.FG, c.what, c.bg, r, MinContrast)
		}
	}
	return nil
}

// ID is what the description's begin marker records: the built-in name, or
// custom:<8 hex of a hash of the normalised theme>.
func (t Theme) ID() string {
	if _, ok := builtin[t.Name]; ok {
		return t.Name
	}
	c := t
	c.Name, c.Schema = "", ""
	data, _ := json.Marshal(c)
	sum := sha256.Sum256(data)
	return "custom:" + hex.EncodeToString(sum[:])[:8]
}

// Names lists the built-in themes, default first.
func Names() []string {
	out := []string{DefaultName}
	var rest []string
	for n := range builtin {
		if n != DefaultName {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// IsBuiltin reports whether name is a built-in theme.
func IsBuiltin(name string) bool { _, ok := builtin[name]; return ok }

// Get returns a built-in theme.
func Get(name string) (Theme, error) {
	t, ok := builtin[name]
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme %q (built-in: %s)", name, strings.Join(Names(), ", "))
	}
	t.Name = name
	return t, nil
}

// Default returns the default theme.
func Default() Theme { t, _ := Get(DefaultName); return t }

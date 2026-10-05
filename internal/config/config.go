// Package config reads, merges, validates and writes the two-key pr-brief
// configuration. Everything else is fixed convention (see package convention).
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gagoar/pr-brief/internal/convention"
)

// SchemaURL is the published JSON Schema location.
const SchemaURL = "https://raw.githubusercontent.com/gagoar/pr-brief/main/schema/pr-brief.schema.json"

// Scope names a config file location.
type Scope string

const (
	ScopeRepo Scope = "repo"
	ScopeUser Scope = "user"
)

// Source names where a resolved value came from.
const (
	SourceFlag    = "flag"
	SourceRepo    = "repo"
	SourceUser    = "user"
	SourceDefault = "default"
)

// File is the on-disk shape. Unknown keys are rejected on load.
type File struct {
	Schema  string   `json:"$schema,omitempty"`
	Version int      `json:"version"`
	Style   string   `json:"style,omitempty"`
	Improve *Improve `json:"improve,omitempty"`
}

// Improve holds the settings of `pr-brief improve`.
type Improve struct {
	Previous string `json:"previous,omitempty"`
}

// Resolved is the merged result with the origin of each field.
type Resolved struct {
	Style          string `json:"style"`
	StyleSource    string `json:"styleSource"`
	Previous       string `json:"improve.previous"`
	PreviousSource string `json:"improve.previousSource"`
}

// Options controls Resolve.
type Options struct {
	RepoPath  string // path to .pr-brief.json; "" means none
	UserPath  string // path to the user file; "" means none
	FlagStyle string // --style value; "" means unset
	CI        bool   // CI ignores the user file and flags
}

// Parse decodes and validates config bytes. Unknown keys are errors.
func Parse(data []byte) (File, error) {
	var f File
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return File{}, fmt.Errorf("invalid config: %w", err)
	}
	if dec.More() {
		return File{}, errors.New("invalid config: trailing data after JSON value")
	}
	if err := f.Validate(); err != nil {
		return File{}, err
	}
	return f, nil
}

// Validate checks the values against the allowed sets.
func (f File) Validate() error {
	if f.Version != convention.ConfigVersion {
		return fmt.Errorf("invalid config: version must be %d, got %d", convention.ConfigVersion, f.Version)
	}
	if f.Style != "" && !contains(convention.Styles, f.Style) {
		return fmt.Errorf("invalid config: style %q must be one of %s", f.Style, strings.Join(convention.Styles, ", "))
	}
	if f.Improve != nil && f.Improve.Previous != "" && !contains(convention.PreviousModes, f.Improve.Previous) {
		return fmt.Errorf("invalid config: improve.previous %q must be one of %s", f.Improve.Previous, strings.Join(convention.PreviousModes, ", "))
	}
	return nil
}

// Load reads and validates one file. A missing file returns (zero, false, nil).
func Load(path string) (File, bool, error) {
	if path == "" {
		return File{}, false, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, false, nil
	}
	if err != nil {
		return File{}, false, err
	}
	f, err := Parse(data)
	if err != nil {
		return File{}, false, fmt.Errorf("%s: %w", path, err)
	}
	return f, true, nil
}

// Resolve merges flag > repo > user > default, field by field.
func Resolve(o Options) (Resolved, error) {
	r := Resolved{
		Style: convention.DefaultStyle, StyleSource: SourceDefault,
		Previous: convention.DefaultPrevious, PreviousSource: SourceDefault,
	}
	type layer struct {
		src  string
		file File
	}
	var layers []layer // lowest precedence first
	if !o.CI {
		uf, ok, err := Load(o.UserPath)
		if err != nil {
			return r, err
		}
		if ok {
			layers = append(layers, layer{SourceUser, uf})
		}
	}
	rf, ok, err := Load(o.RepoPath)
	if err != nil {
		return r, err
	}
	if ok {
		layers = append(layers, layer{SourceRepo, rf})
	}
	for _, l := range layers {
		if l.file.Style != "" {
			r.Style, r.StyleSource = l.file.Style, l.src
		}
		if l.file.Improve != nil && l.file.Improve.Previous != "" {
			r.Previous, r.PreviousSource = l.file.Improve.Previous, l.src
		}
	}
	if !o.CI && o.FlagStyle != "" {
		if !contains(convention.Styles, o.FlagStyle) {
			return r, fmt.Errorf("--style %q must be one of %s", o.FlagStyle, strings.Join(convention.Styles, ", "))
		}
		r.Style, r.StyleSource = o.FlagStyle, SourceFlag
	}
	return r, nil
}

// Set changes one key in the file for the scope and writes it atomically.
// Allowed keys: "style", "improve.previous".
func Set(path, key, value string) error {
	f, _, err := Load(path)
	if err != nil {
		return err
	}
	if f.Version == 0 {
		f.Version = convention.ConfigVersion
	}
	if f.Schema == "" {
		f.Schema = SchemaURL
	}
	switch key {
	case "style":
		f.Style = value
	case "improve.previous":
		if f.Improve == nil {
			f.Improve = &Improve{}
		}
		f.Improve.Previous = value
	default:
		return fmt.Errorf("unknown key %q (allowed: style, improve.previous)", key)
	}
	if err := f.Validate(); err != nil {
		return err
	}
	return write(path, f)
}

// Init writes a starter file with the defaults. It refuses to overwrite.
func Init(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	return write(path, File{
		Schema: SchemaURL, Version: convention.ConfigVersion,
		Style:   convention.DefaultStyle,
		Improve: &Improve{Previous: convention.DefaultPrevious},
	})
}

// Get returns one resolved value by key.
func (r Resolved) Get(key string) (string, error) {
	switch key {
	case "style":
		return r.Style, nil
	case "improve.previous":
		return r.Previous, nil
	}
	return "", fmt.Errorf("unknown key %q (allowed: style, improve.previous)", key)
}

// RepoPath returns <git toplevel>/.pr-brief.json, or <dir>/.pr-brief.json
// when dir is not inside a git work tree.
func RepoPath(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	if out, err := cmd.Output(); err == nil {
		return filepath.Join(strings.TrimSpace(string(out)), ".pr-brief.json")
	}
	return filepath.Join(dir, ".pr-brief.json")
}

// UserPath returns the per-user config location.
func UserPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "pr-brief", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "pr-brief", "config.json")
}

func write(path string, f File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pr-brief-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

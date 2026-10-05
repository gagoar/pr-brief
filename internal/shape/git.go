// Package shape turns a git diff into the flows a PR description draws:
// Input -> changed functions -> Output.
package shape

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Hunk is one -U0 hunk: old and new line ranges (1-based, count may be 0).
type Hunk struct {
	OldStart, OldCount int
	NewStart, NewCount int
	Added              []string // added line texts
	Removed            []string // removed line texts
}

// FileChange is one changed path.
type FileChange struct {
	Path    string // new path
	OldPath string // set for renames
	Status  string // A, M, D, R
	Added   int
	Deleted int
	Binary  bool
	Hunks   []Hunk
}

// Lines is the weight of the change.
func (f FileChange) Lines() int { return f.Added + f.Deleted }

type git struct{ dir string }

func (g git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// mergeBase picks the base commit: the explicit ref, or the merge base with the
// first of origin/main, origin/master, main, master that exists.
func (g git) mergeBase(base, head string) (string, error) {
	cands := []string{base}
	if base == "" {
		cands = []string{"origin/HEAD", "origin/main", "origin/master", "main", "master"}
	}
	for _, c := range cands {
		if _, err := g.run("rev-parse", "--verify", "--quiet", c+"^{commit}"); err != nil {
			continue
		}
		out, err := g.run("merge-base", c, head)
		if err == nil {
			return strings.TrimSpace(out), nil
		}
	}
	if base != "" {
		return "", fmt.Errorf("base %q not found", base)
	}
	return "", fmt.Errorf("cannot find a base branch (tried origin/HEAD, origin/main, origin/master, main, master); pass --base")
}

// changes lists changed files between base and head (head "" = working tree).
func (g git) changes(base, head string) ([]FileChange, error) {
	args := func(extra ...string) []string {
		a := append([]string{"diff", "-M", "--no-color"}, extra...)
		a = append(a, base)
		if head != "" {
			a = append(a, head)
		}
		return a
	}
	ns, err := g.run(args("--name-status", "-z")...)
	if err != nil {
		return nil, err
	}
	var files []FileChange
	parts := strings.Split(strings.TrimRight(ns, "\x00"), "\x00")
	for i := 0; i < len(parts); i++ {
		st := parts[i]
		if st == "" {
			continue
		}
		switch st[0] {
		case 'R', 'C':
			if i+2 >= len(parts) {
				break
			}
			files = append(files, FileChange{Status: "R", OldPath: parts[i+1], Path: parts[i+2]})
			i += 2
		default:
			if i+1 >= len(parts) {
				break
			}
			files = append(files, FileChange{Status: string(st[0]), Path: parts[i+1]})
			i++
		}
	}

	num, err := g.run(args("--numstat", "-z")...)
	if err != nil {
		return nil, err
	}
	stats := map[string][2]int{}
	bin := map[string]bool{}
	np := strings.Split(strings.TrimRight(num, "\x00"), "\x00")
	for i := 0; i < len(np); i++ {
		f := strings.SplitN(np[i], "\t", 3)
		if len(f) < 3 {
			continue
		}
		path := f[2]
		if path == "" && i+2 < len(np) { // rename: next two entries are old, new
			path = np[i+2]
			i += 2
		}
		if f[0] == "-" {
			bin[path] = true
			continue
		}
		a, _ := strconv.Atoi(f[0])
		d, _ := strconv.Atoi(f[1])
		stats[path] = [2]int{a, d}
	}
	for i := range files {
		s := stats[files[i].Path]
		files[i].Added, files[i].Deleted = s[0], s[1]
		files[i].Binary = bin[files[i].Path]
	}
	return files, nil
}

var hunkRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// hunks reads the -U0 hunks of one file.
func (g git) hunks(base, head string, f FileChange) ([]Hunk, error) {
	a := []string{"diff", "-M", "--no-color", "-U0", base}
	if head != "" {
		a = append(a, head)
	}
	a = append(a, "--")
	if f.OldPath != "" {
		a = append(a, f.OldPath)
	}
	a = append(a, f.Path)
	out, err := g.run(a...)
	if err != nil {
		return nil, err
	}
	return parseHunks(out), nil
}

func parseHunks(diff string) []Hunk {
	var hs []Hunk
	for _, line := range strings.Split(diff, "\n") {
		if m := hunkRe.FindStringSubmatch(line); m != nil {
			h := Hunk{OldCount: 1, NewCount: 1}
			h.OldStart, _ = strconv.Atoi(m[1])
			h.NewStart, _ = strconv.Atoi(m[3])
			if m[2] != "" {
				h.OldCount, _ = strconv.Atoi(m[2])
			}
			if m[4] != "" {
				h.NewCount, _ = strconv.Atoi(m[4])
			}
			hs = append(hs, h)
			continue
		}
		if len(hs) == 0 {
			continue
		}
		h := &hs[len(hs)-1]
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			h.Added = append(h.Added, line[1:])
		case strings.HasPrefix(line, "-"):
			h.Removed = append(h.Removed, line[1:])
		}
	}
	return hs
}

// show returns a file's content at a ref ("" ref reads the working tree).
func (g git) show(ref, path string) (string, error) {
	if ref == "" {
		out, err := exec.Command("cat", g.dir+"/"+path).Output()
		return string(out), err
	}
	return g.run("show", ref+":"+path)
}

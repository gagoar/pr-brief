package body

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const stampLayout = "20060102T150405Z"

// StateDir returns ~/.local/state/pr-brief (or $XDG_STATE_HOME/pr-brief).
func StateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "pr-brief")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "pr-brief")
	}
	return filepath.Join(home, ".local", "state", "pr-brief")
}

// PRDir is the backup directory of one PR.
func PRDir(stateDir, host, owner, repo, pr string) string {
	return filepath.Join(stateDir, host, owner, repo, pr)
}

// Backup writes the current description before any change and returns its path.
// It never overwrites an existing file; same-second backups get a numeric suffix.
func Backup(stateDir, host, owner, repo, pr, text string, now time.Time) (string, error) {
	dir := PRDir(stateDir, host, owner, repo, pr)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// The zero-padded counter keeps lexical order equal to creation order.
	base := now.UTC().Format(stampLayout)
	for i := 0; ; i++ {
		name := fmt.Sprintf("%s-%02d.md", base, i)
		p := filepath.Join(dir, name)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.WriteString(text); err != nil {
			f.Close()
			return "", err
		}
		return p, f.Close()
	}
}

// List returns the backup file names for a PR, oldest first.
func List(stateDir, host, owner, repo, pr string) ([]string, error) {
	entries, err := os.ReadDir(PRDir(stateDir, host, owner, repo, pr))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Restore returns a saved description. at == "" means the oldest backup, which
// is the description as it was before pr-brief first touched it. Otherwise at
// is a file-name prefix such as 20261005T140200Z.
func Restore(stateDir, host, owner, repo, pr, at string) (string, error) {
	names, err := List(stateDir, host, owner, repo, pr)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no backups for %s/%s#%s", owner, repo, pr)
	}
	pick := names[0]
	if at != "" {
		pick = ""
		for _, n := range names {
			if strings.HasPrefix(n, at) {
				pick = n
				break
			}
		}
		if pick == "" {
			return "", fmt.Errorf("no backup matching %q (have: %s)", at, strings.Join(names, ", "))
		}
	}
	data, err := os.ReadFile(filepath.Join(PRDir(stateDir, host, owner, repo, pr), pick))
	return string(data), err
}

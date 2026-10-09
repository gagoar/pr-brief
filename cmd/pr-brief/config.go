package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gagoar/pr-brief/internal/config"
)

func scopePath(scope, cwd string) (string, error) {
	switch config.Scope(scope) {
	case config.ScopeRepo:
		return config.RepoPath(cwd), nil
	case config.ScopeUser:
		if p := config.UserPath(); p != "" {
			return p, nil
		}
		return "", fmt.Errorf("cannot find the user config directory")
	}
	return "", fmt.Errorf("--scope must be user or repo, got %q", scope)
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: pr-brief config show|get|set|init|validate")
		return 2
	}
	sub, rest := args[0], args[1:]
	fs := flag.NewFlagSet("config "+sub, flag.ContinueOnError)
	fs.SetOutput(stderr)
	scope := fs.String("scope", "user", "user or repo")
	asJSON := fs.Bool("json", false, "print JSON")
	ci := fs.Bool("ci", false, "resolve as CI does (repo file and defaults only)")
	// Allow flags after positional arguments (config set style ste --scope repo).
	var pos []string
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) > 0 {
			pos = append(pos, rest[0])
			rest = rest[1:]
		}
	}
	cwd, _ := os.Getwd()

	switch sub {
	case "show":
		r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), UserPath: config.UserPath(), CI: *ci})
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief config:", err)
			return 1
		}
		if *asJSON {
			data, _ := json.MarshalIndent(r, "", "  ")
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		fmt.Fprintf(stdout, "style            %s  (%s)\n", r.Style, r.StyleSource)
		fmt.Fprintf(stdout, "improve.previous %s  (%s)\n", r.Previous, r.PreviousSource)
		fmt.Fprintf(stdout, "diagram.theme    %s  (%s)\n", r.Theme, r.ThemeSource)
		pattern := r.TicketsPattern
		if pattern == "" {
			pattern = "(built-in Jira and Linear keys)"
		}
		fmt.Fprintf(stdout, "tickets.pattern  %s  (%s)\n", pattern, r.TicketsPatternSource)
		return 0
	case "get":
		if len(pos) != 1 {
			fmt.Fprintln(stderr, "usage: pr-brief config get <style|improve.previous|diagram.theme|tickets.pattern>")
			return 2
		}
		r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), UserPath: config.UserPath(), CI: *ci})
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief config:", err)
			return 1
		}
		v, err := r.Get(pos[0])
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief config:", err)
			return 2
		}
		fmt.Fprintln(stdout, v)
		return 0
	case "set":
		if len(pos) != 2 {
			fmt.Fprintln(stderr, "usage: pr-brief config set <style|improve.previous|diagram.theme|tickets.pattern> <value> [--scope user|repo]")
			return 2
		}
		p, err := scopePath(*scope, cwd)
		if err == nil {
			err = config.Set(p, pos[0], pos[1])
		}
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief config:", err)
			return 1
		}
		fmt.Fprintf(stdout, "set %s=%s in %s\n", pos[0], pos[1], p)
		return 0
	case "init":
		p, err := scopePath(*scope, cwd)
		if err == nil {
			err = config.Init(p)
		}
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief config:", err)
			return 1
		}
		fmt.Fprintln(stdout, "wrote", p)
		return 0
	case "validate":
		files := pos
		if len(files) == 0 {
			files = []string{config.RepoPath(cwd), config.UserPath()}
		}
		bad := 0
		for _, f := range files {
			_, ok, err := config.Load(f)
			switch {
			case err != nil:
				fmt.Fprintln(stdout, "FAIL", err)
				bad++
			case ok:
				fmt.Fprintln(stdout, "ok  ", f)
			}
		}
		if bad > 0 {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "pr-brief config: unknown subcommand %q\n", sub)
	return 2
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/gagoar/pr-brief/internal/shape"
)

func runShape(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("shape", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repository directory")
	base := fs.String("base", "", "base ref (default: merge base with the default branch)")
	head := fs.String("head", "", "head ref (default: HEAD)")
	working := fs.Bool("working-tree", false, "compare the base with the working tree, untracked files included")
	title := fs.String("title", "", "PR title, used to recognise fixes")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rep, err := shape.Analyze(shape.Options{Dir: *dir, Base: *base, Head: *head, WorkingTree: *working, Title: *title})
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief shape:", err)
		return 1
	}
	data, _ := json.MarshalIndent(rep, "", "  ")
	fmt.Fprintln(stdout, string(data))
	return 0
}

package main

import (
	"fmt"
	"io"
)

func runShape(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "pr-brief shape: not implemented yet")
	return 1
}

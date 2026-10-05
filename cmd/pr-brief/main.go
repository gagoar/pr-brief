package main

import (
	"fmt"
	"os"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pr-brief <shape|gate|lint|config|body|version>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "pr-brief: unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

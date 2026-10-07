// Command pr-brief gates and improves pull request descriptions.
package main

import (
	"fmt"
	"io"
	"os"
)

var version = "0.1.0-dev"

const usage = `usage: pr-brief <command> [flags]

commands:
  shape     analyse the diff and print the flows to draw (JSON)
  gate      check a PR description (--hook, --ci, --file, --stdin)
  lint      run the STE100 linter on prose
  init      set a repo up: .pr-brief.json, and optionally the PR template and workflow
  config    show | get | set | init | validate
  diagram   render one flow as a themed Mermaid diagram
  theme     list | show | validate
  body      improve | past | restore | uncomment
  links     link the Read-these-first files to the file or its diff in the PR
  version   print the version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	args := os.Args[2:]
	var code int
	switch os.Args[1] {
	case "version", "--version":
		fmt.Println(version)
	case "lint":
		code = runLint(args, os.Stdin, os.Stdout, os.Stderr)
	case "gate":
		code = runGate(args, os.Stdin, os.Stdout, os.Stderr)
	case "config":
		code = runConfig(args, os.Stdout, os.Stderr)
	case "body":
		code = runBody(args, os.Stdin, os.Stdout, os.Stderr)
	case "shape":
		code = runShape(args, os.Stdout, os.Stderr)
	case "init":
		code = runInit(args, os.Stdout, os.Stderr)
	case "diagram":
		code = runDiagram(args, os.Stdin, os.Stdout, os.Stderr)
	case "theme":
		code = runTheme(args, os.Stdout, os.Stderr)
	case "links":
		code = runLinks(args, os.Stdin, os.Stdout, os.Stderr)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "pr-brief: unknown command %q\n\n%s", os.Args[1], usage)
		code = 2
	}
	os.Exit(code)
}

// readInput reads a file, or stdin when path is "-" or empty.
func readInput(path string, stdin io.Reader) (string, error) {
	if path == "" || path == "-" {
		data, err := io.ReadAll(stdin)
		return string(data), err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

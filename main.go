package main

import (
	_ "embed"
	"os"

	"nanospec/internal/cli"
)

//go:embed execution-rules.md
var executionRules string

func main() {
	os.Exit(cli.RunWithExecutionRules(os.Args[1:], os.Stdout, os.Stderr, executionRules))
}

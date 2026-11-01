package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"nanospec/internal/validator"
)

const validateHelp = "Usage: nanospec validate [--sync-views] [cycle-dir]\n\n" +
	"Validate one Cycle directory. The target defaults to the\n" +
	"current directory. Templates and rules define the specification; this command\n" +
	"checks only the stable, machine-checkable subset.\n\n" +
	"Options:\n" +
	"  -h, --help    Show this help\n" +
	"  --sync-views  Update Dependency Waves, Dependency Graph and Progress Tree.\n" +
	"                Uses Story depends_on, Wave entry order and Acceptance Status.\n" +
	"                All other content is preserved. Without this flag, no files change.\n\n" +
	"Exit codes:\n" +
	"  0  Validation passed\n" +
	"  1  Specification errors found\n" +
	"  2  Invalid usage or target"

func runValidate(args []string, stdout, stderr io.Writer) int {
	for _, argument := range args {
		if argument == "-h" || argument == "--help" {
			fmt.Fprintln(stdout, validateHelp)
			return 0
		}
	}

	target := ""
	syncViews := false
	flagCount := 0
	for _, argument := range args {
		if strings.HasPrefix(argument, "-") {
			flagCount++
			if argument != "--sync-views" {
				fmt.Fprintln(stderr, validateHelp)
				return 2
			}
			syncViews = true
			continue
		}
		if target != "" {
			fmt.Fprintln(stderr, validateHelp)
			return 2
		}
		target = argument
	}
	if flagCount > 1 {
		fmt.Fprintln(stderr, validateHelp)
		return 2
	}
	if target == "" {
		target = "."
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	var result validator.ValidationResult
	if syncViews {
		result = validator.SynchronizeCycleViews(absolute)
	} else {
		result = validator.ValidateSpec(absolute, false)
	}
	if result.InputError != "" {
		fmt.Fprintln(stderr, result.InputError)
		return 2
	}
	workingDirectory, _ := os.Getwd()
	for _, issue := range result.Warnings {
		fmt.Fprintf(stderr, "WARN %s: %s\n", relativePath(workingDirectory, issue.File), issue.Message)
	}
	for _, issue := range result.Errors {
		fmt.Fprintf(stderr, "ERROR %s: %s\n", relativePath(workingDirectory, issue.File), issue.Message)
	}
	if len(result.Errors) > 0 {
		fmt.Fprintf(stderr, "\nSpec validation failed: %d error(s), %d warning(s).\n", len(result.Errors), len(result.Warnings))
		return 1
	}

	fmt.Fprintf(stdout, "Spec validation passed: 0 errors, %d warning(s).\n", len(result.Warnings))
	if syncViews {
		fmt.Fprintf(stdout, "Views synchronized: %d file(s) updated.\n", len(result.Updates))
	}
	return 0
}

func relativePath(base, file string) string {
	relative, err := filepath.Rel(base, file)
	if err != nil {
		return file
	}
	return relative
}

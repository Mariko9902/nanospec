package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"nanospec/internal/validator"
)

const showHelp = "Usage: nanospec show <overview|graph> [cycle-dir]\n\n" +
	"Show one Cycle view without changing files. The Cycle directory\n" +
	"defaults to the current directory."

const readyHelp = "Usage: nanospec ready [cycle-dir]\n\n" +
	"List all candidates whose dependency acceptance states allow further checks.\n" +
	"The command does not select work, change state, or start execution."

const prepareHelp = "Usage: nanospec prepare [--json] <story-dir-or-story.md>...\n\n" +
	"Prepare only the explicitly selected Stories without changing files."

func runShow(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		fmt.Fprintln(stdout, showHelp)
		return 0
	}
	if len(args) < 1 || len(args) > 2 || !supportedView(args[0]) {
		fmt.Fprintln(stderr, showHelp)
		return 2
	}
	target := "."
	if len(args) == 2 {
		target = args[1]
	}
	snapshot, result := validator.LoadCycleSnapshot(target)
	if code := reportQueryResult(result, stderr); code != 0 {
		return code
	}
	view, err := snapshot.View(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprint(stdout, view)
	return 0
}

func runReady(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		fmt.Fprintln(stdout, readyHelp)
		return 0
	}
	if len(args) > 1 {
		fmt.Fprintln(stderr, readyHelp)
		return 2
	}
	target := "."
	if len(args) == 1 {
		target = args[0]
	}
	snapshot, result := validator.LoadCycleSnapshot(target)
	if code := reportQueryResult(result, stderr); code != 0 {
		return code
	}

	fmt.Fprintln(stdout, "Execution candidates (dependency state only):")
	candidates := snapshot.ReadyCandidates()
	if len(candidates) == 0 {
		fmt.Fprintln(stdout, "- none")
	}
	for _, candidate := range candidates {
		dependencies := "none"
		if len(candidate.Dependencies) > 0 {
			items := make([]string, 0, len(candidate.Dependencies))
			for _, dependency := range candidate.Dependencies {
				items = append(items, dependency.StoryID+"="+dependency.Acceptance)
			}
			dependencies = strings.Join(items, ", ")
		}
		fmt.Fprintf(stdout, "- %s | acceptance=%s | dependencies=%s\n", candidate.StoryID, candidate.Acceptance, dependencies)
	}
	fmt.Fprintln(stdout, "Before execution, confirm evidence applicability, current-environment availability, and any required human acceptance.")
	fmt.Fprintln(stdout, "This command does not select work, change state, or start execution.")
	return 0
}

func runPrepare(args []string, stdout, stderr io.Writer, executionRules string) int {
	if hasHelp(args) {
		fmt.Fprintln(stdout, prepareHelp)
		return 0
	}
	jsonOutput := false
	targets := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintln(stderr, prepareHelp)
			return 2
		}
		targets = append(targets, arg)
	}
	if len(targets) == 0 {
		fmt.Fprintln(stderr, prepareHelp)
		return 2
	}
	if strings.TrimSpace(executionRules) == "" {
		fmt.Fprintln(stderr, "NanoSpec execution rules are unavailable")
		return 2
	}
	packages := make([]validator.ExecutionPackage, 0, len(targets))
	selected := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		item, result := validator.BuildExecutionPackage(target, executionRules)
		if code := reportQueryResult(result, stderr); code != 0 {
			return code
		}
		key := item.CycleID + "/" + item.StoryID
		if _, duplicate := selected[key]; duplicate {
			fmt.Fprintln(stderr, "duplicate prepared Story: "+key)
			return 2
		}
		selected[key] = struct{}{}
		packages = append(packages, item)
	}
	validator.ResolvePreparationGates(packages)
	if jsonOutput {
		payload := struct {
			Schema    string                       `json:"schema"`
			Selection string                       `json:"selection"`
			Stories   []validator.ExecutionPackage `json:"stories"`
		}{Schema: "nanospec.prepare.v1", Selection: "explicit", Stories: packages}
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(payload); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		renderPreparation(stdout, packages)
	}
	for _, item := range packages {
		if item.Gate == "blocked" || item.Gate == "blocked_by_dependency" || item.Gate == "blocked_by_open_questions" || item.Gate == "awaiting_acceptance" || item.Gate == "accepted" {
			return 1
		}
	}
	return 0
}

func renderPreparation(output io.Writer, packages []validator.ExecutionPackage) {
	fmt.Fprintln(output, "# NanoSpec Execution Preparation")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "- Selection: explicit")
	fmt.Fprintf(output, "- Stories: %d\n", len(packages))
	for _, item := range packages {
		fmt.Fprintln(output)
		fmt.Fprintf(output, "## %s/%s\n\n", item.CycleID, item.StoryID)
		fmt.Fprintf(output, "- Spec Revision: %s\n", item.SpecRevision)
		fmt.Fprintf(output, "- Status: %s\n", item.Status)
		fmt.Fprintf(output, "- Acceptance: %s\n", item.Acceptance)
		fmt.Fprintf(output, "- Gate: %s\n", item.Gate)
		fmt.Fprintln(output, "- Required Confirmations:")
		for _, confirmation := range item.RequiredConfirmations {
			fmt.Fprintf(output, "  - %s\n", confirmation)
		}
		fmt.Fprintln(output)
		fmt.Fprint(output, item.ContextMarkdown)
	}
}

func reportQueryResult(result validator.ValidationResult, stderr io.Writer) int {
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
	return 0
}

func supportedView(name string) bool {
	return name == "overview" || name == "graph"
}

func hasHelp(args []string) bool {
	return len(args) == 1 && (args[0] == "-h" || args[0] == "--help")
}

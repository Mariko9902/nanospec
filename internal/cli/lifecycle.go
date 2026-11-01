package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"nanospec/internal/validator"
)

const checkpointHelp = "Usage: nanospec checkpoint --owner <id> --status <active|paused|blocked|done> --next <text> --situation <text> [--issue <text>] <story>\n\n" +
	"Atomically update the explicitly selected Story's current execution facts."

const reviseHelp = "Usage: nanospec revise --reason <text> <story>...\n\n" +
	"After semantic specification edits validate successfully, increment the Cycle revision once for the explicitly listed affected Stories."

func runCheckpoint(args []string, stdout, stderr io.Writer) int {
	if containsHelp(args) {
		fmt.Fprintln(stdout, checkpointHelp)
		return 0
	}
	flags := flag.NewFlagSet("checkpoint", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	owner := flags.String("owner", "", "")
	status := flags.String("status", "", "")
	next := flags.String("next", "", "")
	situation := flags.String("situation", "", "")
	issue := flags.String("issue", "", "")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 1 {
		fmt.Fprintln(stderr, checkpointHelp)
		return 2
	}
	result, validation := validator.SaveStoryCheckpoint(flags.Args()[0], validator.CheckpointRequest{
		Owner:     *owner,
		Status:    *status,
		Next:      *next,
		Situation: *situation,
		Issue:     *issue,
		At:        time.Now(),
	})
	if code := reportQueryResult(validation, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Checkpoint saved: %s/%s | lane=%s | status=%s | spec_revision=%s\n", result.CycleID, result.StoryID, result.LaneID, result.Status, result.SpecRevision)
	return 0
}

func runRevise(args []string, stdout, stderr io.Writer) int {
	if containsHelp(args) {
		fmt.Fprintln(stdout, reviseHelp)
		return 0
	}
	flags := flag.NewFlagSet("revise", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reason := flags.String("reason", "", "")
	if err := flags.Parse(args); err != nil || len(flags.Args()) == 0 || strings.TrimSpace(*reason) == "" {
		fmt.Fprintln(stderr, reviseHelp)
		return 2
	}
	result, validation := validator.ReviseStories(flags.Args(), validator.RevisionRequest{Reason: *reason, At: time.Now()})
	if code := reportQueryResult(validation, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Revision saved: %s | spec_revision=%s | stories=%s\n", result.CycleID, result.NewRevision, strings.Join(result.StoryIDs, ","))
	return 0
}

func containsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

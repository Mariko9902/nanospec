package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"nanospec/internal/validator"
)

const evidenceHelp = "Usage: nanospec evidence --spec-revision <rev-N> --implemented-by <id> --acceptance <id> --observed <text> --command <text> --result <text> --artifacts <text> --context <json> <story>\n\n" +
	"Create or replace current evidence for one Acceptance ID.\n" +
	"Context JSON requires spec_revision, code_revision, environment and observed_at."

const verifyHelp = "Usage: nanospec verify --spec-revision <rev-N> --verified-by <id> --mode <self|independent> --result <pass|rework> <story>\n\n" +
	"Record a complete review result for the explicitly selected Story."

const acceptHelp = "Usage: nanospec accept --spec-revision <rev-N> --accepted-by <id> --method <human|verified> <story>\n\n" +
	"After explicit acceptance authority is established, freeze passed evidence as acceptance/outcome.md."

const rejectHelp = "Usage: nanospec reject --spec-revision <rev-N> --rejected-by <id> --reason <text> <story>\n\n" +
	"Reject a ready delivery and return its evidence to rework."

func runEvidence(args []string, stdout, stderr io.Writer) int {
	if containsHelp(args) {
		fmt.Fprintln(stdout, evidenceHelp)
		return 0
	}
	flags := flag.NewFlagSet("evidence", flag.ContinueOnError)
	specRevision := flags.String("spec-revision", "", "")
	flags.SetOutput(io.Discard)
	implementedBy := flags.String("implemented-by", "", "")
	acceptance := flags.String("acceptance", "", "")
	observed := flags.String("observed", "", "")
	command := flags.String("command", "", "")
	resultText := flags.String("result", "", "")
	artifacts := flags.String("artifacts", "", "")
	context := flags.String("context", "", "")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 1 {
		fmt.Fprintln(stderr, evidenceHelp)
		return 2
	}
	result, validation := validator.RecordStoryEvidence(flags.Args()[0], validator.EvidenceCaptureRequest{
		SpecRevision:  *specRevision,
		ImplementedBy: *implementedBy, AcceptanceID: *acceptance, Observed: *observed,
		Command: *command, Result: *resultText, Artifacts: *artifacts, Context: *context, At: time.Now(),
	})
	if code := reportQueryResult(validation, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Evidence saved: %s/%s | acceptance=%s\n", result.CycleID, result.StoryID, result.AcceptanceID)
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	if containsHelp(args) {
		fmt.Fprintln(stdout, verifyHelp)
		return 0
	}
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	specRevision := flags.String("spec-revision", "", "")
	flags.SetOutput(io.Discard)
	verifiedBy := flags.String("verified-by", "", "")
	mode := flags.String("mode", "", "")
	resultText := flags.String("result", "", "")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 1 {
		fmt.Fprintln(stderr, verifyHelp)
		return 2
	}
	result, validation := validator.VerifyStory(flags.Args()[0], validator.VerificationRequest{
		SpecRevision: *specRevision,
		VerifiedBy:   *verifiedBy, Mode: validator.VerificationMode(*mode), Result: *resultText, At: time.Now(),
	})
	if code := reportQueryResult(validation, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Verification saved: %s/%s | status=%s | rework_count=%d\n", result.CycleID, result.StoryID, result.Status, result.ReworkCount)
	return 0
}

func runAccept(args []string, stdout, stderr io.Writer) int {
	if containsHelp(args) {
		fmt.Fprintln(stdout, acceptHelp)
		return 0
	}
	flags := flag.NewFlagSet("accept", flag.ContinueOnError)
	specRevision := flags.String("spec-revision", "", "")
	flags.SetOutput(io.Discard)
	acceptedBy := flags.String("accepted-by", "", "")
	method := flags.String("method", "", "")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 1 {
		fmt.Fprintln(stderr, acceptHelp)
		return 2
	}
	result, validation := validator.AcceptStory(flags.Args()[0], validator.AcceptanceRequest{
		SpecRevision: *specRevision,
		AcceptedBy:   *acceptedBy, Method: validator.AcceptancePolicy(*method), At: time.Now(),
	})
	if code := reportQueryResult(validation, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Acceptance saved: %s/%s | method=%s | outcome=acceptance/outcome.md\n", result.CycleID, result.StoryID, *method)
	return 0
}

func runReject(args []string, stdout, stderr io.Writer) int {
	if containsHelp(args) {
		fmt.Fprintln(stdout, rejectHelp)
		return 0
	}
	flags := flag.NewFlagSet("reject", flag.ContinueOnError)
	specRevision := flags.String("spec-revision", "", "")
	flags.SetOutput(io.Discard)
	rejectedBy := flags.String("rejected-by", "", "")
	reason := flags.String("reason", "", "")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 1 {
		fmt.Fprintln(stderr, rejectHelp)
		return 2
	}
	result, validation := validator.RejectStory(flags.Args()[0], validator.RejectionRequest{
		SpecRevision: *specRevision,
		RejectedBy:   *rejectedBy, Reason: *reason, At: time.Now(),
	})
	if code := reportQueryResult(validation, stderr); code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "Rejection saved: %s/%s | rework_count=%d\n", result.CycleID, result.StoryID, result.ReworkCount)
	return 0
}

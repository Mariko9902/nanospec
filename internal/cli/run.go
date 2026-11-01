package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
)

const protocolIdentity = "nanospec.protocol.v1"

const help = "Usage: nanospec <command> [options] [target]\n\n" +
	"NanoSpec specification tools.\n\n" +
	"Commands:\n" +
	"  validate      Validate one Cycle directory.\n" +
	"  show          Show a generated Cycle view without changing files.\n" +
	"  ready         List execution candidates without selecting one.\n" +
	"  prepare       Prepare explicitly selected Stories for execution.\n" +
	"  checkpoint    Update current execution facts for one selected Story.\n" +
	"  revise        Record one semantic revision for selected affected Stories.\n" +
	"  evidence      Record observed evidence for one Acceptance ID.\n" +
	"  verify        Record complete review as passed or requiring rework.\n" +
	"  accept        Freeze passed evidence after explicit acceptance.\n" +
	"  reject        Reject a ready delivery and return it to rework.\n" +
	"  version       Show tool and embedded protocol identity.\n\n" +
	"Options:\n" +
	"  -h, --help    Show this help\n\n" +
	"Exit codes:\n" +
	"  0  Command completed\n" +
	"  1  Command found specification errors\n" +
	"  2  Invalid usage or target"

// Run parses command arguments and returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWithExecutionRules(args, stdout, stderr, "")
}

// RunWithExecutionRules runs commands with the execution rules embedded in this build.
func RunWithExecutionRules(args []string, stdout, stderr io.Writer, executionRules string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, help)
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, help)
		return 2
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "show":
		return runShow(args[1:], stdout, stderr)
	case "ready":
		return runReady(args[1:], stdout, stderr)
	case "prepare":
		return runPrepare(args[1:], stdout, stderr, executionRules)
	case "checkpoint":
		return runCheckpoint(args[1:], stdout, stderr)
	case "revise":
		return runRevise(args[1:], stdout, stderr)
	case "evidence":
		return runEvidence(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "accept":
		return runAccept(args[1:], stdout, stderr)
	case "reject":
		return runReject(args[1:], stdout, stderr)
	case "version":
		return runVersion(args[1:], stdout, stderr, executionRules)
	}

	fmt.Fprintf(stderr, "Unknown command: %s\n\n", args[0])
	fmt.Fprintln(stderr, help)
	return 2
}

func runVersion(args []string, stdout, stderr io.Writer, executionRules string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, "Usage: nanospec version\n\nShow the tool build, protocol identity, and embedded execution-rules digest.")
		return 0
	}
	if len(args) != 0 {
		fmt.Fprintln(stderr, "Usage: nanospec version")
		return 2
	}
	digest := sha256.Sum256([]byte(executionRules))
	fmt.Fprintf(stdout, "nanospec: %s\n", buildIdentity())
	fmt.Fprintf(stdout, "protocol: %s\n", protocolIdentity)
	fmt.Fprintf(stdout, "execution_rules_digest: sha256:%x\n", digest)
	return 0
}

func buildIdentity() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	version := info.Main.Version
	revision := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if version == "" || version == "(devel)" {
		version = revision
	}
	if strings.TrimSpace(version) == "" {
		version = "devel"
	}
	if modified {
		version += "+modified"
	}
	return version
}

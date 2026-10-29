package probe

import (
	"flag"
	"fmt"
	"io"
)

const usage = "usage: photocurator probe generate --out <dir>\n       photocurator probe mutate --dir <dir>\n"

// Run executes the probe subcommand; args are the arguments after "probe". It returns the
// process exit code so tests drive the same entry as the binary.
func Run(args []string, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "mutate":
		return runMutate(args[1:], stderr)
	case "generate":
		return runGenerate(args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "unknown probe command %q\n%s", args[0], usage)
		return 2
	}
}

func runGenerate(args []string, stderr io.Writer) int {
	fl := flag.NewFlagSet("probe generate", flag.ContinueOnError)
	fl.SetOutput(stderr)
	out := fl.String("out", "", "output directory (must not exist or be empty)")
	if err := fl.Parse(args); err != nil {
		return 2
	}
	if *out == "" {
		fmt.Fprint(stderr, "--out is required\n"+usage)
		return 2
	}
	if err := Generate(*out); err != nil {
		fmt.Fprintf(stderr, "probe generate: %v\n", err)
		return 1
	}
	return 0
}

// Command entidc compiles, lints, inspects, compares and publishes the
// EntID rule and conformance artifacts.
//
// Exit codes:
//
//	0  success
//	1  usage error
//	2  the inputs are rejected: diagnostics were reported
//	3  internal error
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/entid-org/spec/internal/diagnostics"
	"github.com/entid-org/spec/internal/version"
)

// globalCountry is the label used in human output for a definition without
// country code.
const globalCountry = "GLOBAL"

// Exit codes of the CLI.
const (
	exitOK       = 0
	exitUsage    = 1
	exitRejected = 2
	exitInternal = 3
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type command struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

func run(args []string, stdout, stderr io.Writer) int {
	commands := []command{
		{"fmt", "rewrite the rule and conformance sources in their canonical form", runFmt},
		{"lint", "check the sources beyond compilation: idempotence, provenance and data policy", runLint},
		{"compile", "compile the rules and the conformance corpus into publishable artifacts", runCompile},
		{"verify", "compile in memory and run the whole conformance suite", runVerify},
		{"inspect", "describe a compiled bundle", runInspect},
		{"diff", "classify the changes between two compiled bundles", runDiff},
		{"check-generated", "rebuild everything in temporary directories and compare the committed bytes", runCheckGenerated},
		{"version", "print the compiler version", runVersion},
	}
	if len(args) == 0 {
		usage(stderr, commands)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		usage(stdout, commands)
		return exitOK
	}
	for _, c := range commands {
		if c.name == args[0] {
			return c.run(args[1:], stdout, stderr)
		}
	}
	_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
	usage(stderr, commands)
	return exitUsage
}

func usage(out io.Writer, commands []command) {
	_, _ = fmt.Fprintf(out, "%s %s\n\n", version.Name, version.Compiler)
	_, _ = fmt.Fprintf(out, "Usage:\n  %s <command> [flags]\n\nCommands:\n", version.Name)
	for _, c := range commands {
		_, _ = fmt.Fprintf(out, "  %-16s %s\n", c.name, c.summary)
	}
	_, _ = fmt.Fprintf(out, "\nExit codes:\n  0 success\n  1 usage error\n  2 inputs rejected\n  3 internal error\n")
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	_, _ = fmt.Fprintf(stdout, "%s %s\n", version.Name, version.Compiler)
	return exitOK
}

// report renders the diagnostics and returns the matching exit code.
func report(bag *diagnostics.Bag, asJSON bool, stdout, stderr io.Writer) int {
	if bag == nil || bag.Len() == 0 {
		return exitOK
	}
	if asJSON {
		if err := bag.WriteJSON(stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "cannot render the diagnostics: %v\n", err)
			return exitInternal
		}
	} else if err := bag.WriteText(stderr); err != nil {
		_, _ = fmt.Fprintf(stderr, "cannot render the diagnostics: %v\n", err)
		return exitInternal
	}
	if bag.HasErrors() {
		return exitRejected
	}
	return exitOK
}

func fail(stderr io.Writer, format string, args ...any) int {
	_, _ = fmt.Fprintf(stderr, format+"\n", args...)
	return exitInternal
}

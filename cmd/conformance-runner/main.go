// Command conformance-runner confronts an engine with the conformance corpus.
//
// It is the only program that reads expected results. The engine under test
// answers requests through a small executable — the testee — and never sees an
// expectation, so it cannot declare itself conformant by comparing too weakly.
//
//	conformance-runner --corpus dist/businessid-conformance-2026.08.0.binpb -- ./bin/testee --bundle rules.binpb
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	"github.com/libbusinessid/spec/internal/register"
	"github.com/libbusinessid/spec/internal/runner"
)

func main() {
	corpus := flag.String("corpus", "", "path to the conformance bundle (.binpb)")
	timeout := flag.Duration("timeout", 0, "bound on the whole run")
	only := flag.String("operation", "",
		"restrict to one operation, for diagnosis only; a restricted run is never a conformance verdict")
	sweep := flag.String("register", "",
		"sweep an issuer's complete register: <id>=<file>, where <id> is named in "+manifestPath+
			"; the run then asks only whether every identifier the issuer has handed out is accepted")
	flag.Parse()

	ok, err := run(*corpus, flag.Args(), *timeout, *only, *sweep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "conformance-runner: %v\n", err)
		os.Exit(1)
	}
	if !ok {
		os.Exit(1)
	}
}

// run returns whether the engine is conformant. Exiting is left to main so
// that the whole path stays testable.
func run(corpusPath string, command []string, timeout time.Duration, only, sweep string) (bool, error) {
	if len(command) == 0 {
		return false, fmt.Errorf("give the testee command after --")
	}
	if sweep != "" {
		if only != "" {
			return false, fmt.Errorf("--operation and --register cannot be combined")
		}
		return runSweep(sweep, command, timeout, manifestPath)
	}
	if corpusPath == "" {
		return false, fmt.Errorf("--corpus is required")
	}

	raw, err := os.ReadFile(filepath.Clean(corpusPath))
	if err != nil {
		return false, fmt.Errorf("cannot read the corpus: %w", err)
	}
	var bundle conformancev1.ConformanceBundle
	if err := proto.Unmarshal(raw, &bundle); err != nil {
		return false, fmt.Errorf("the corpus is not a conformance bundle: %w", err)
	}

	cases := bundle.GetCases()
	restricted := false
	if only != "" {
		want, ok := operationByName(only)
		if !ok {
			return false, fmt.Errorf("unknown operation %q", only)
		}
		kept := cases[:0:0]
		for _, c := range cases {
			if c.GetOperation() == want {
				kept = append(kept, c)
			}
		}
		cases, restricted = kept, true
	}
	if len(cases) == 0 {
		return false, fmt.Errorf("the corpus carries no case to run")
	}

	res, err := runner.Run(context.Background(), cases, runner.Options{Command: command, Timeout: timeout})
	if err != nil {
		return false, err
	}

	report(res, bundle.GetRulesVersion(), restricted)
	// A restricted run never states conformance, whatever it found.
	return res.Conformant() && !restricted, nil
}

// runSweep confronts an engine with every identifier one issuer has handed out.
//
// This is not a conformance verdict and does not claim to be. The corpus is
// what settles conformance, because a corpus case states what should happen
// down to the reason code, both for values that are valid and for values that
// are not. A register holds only identifiers that exist, so a sweep can ask one
// question - is every one of them accepted - and answers it several million
// times. It catches the false refusal, which section 1.2 calls the worst defect
// this project can commit, and which a corpus of a few hundred hand picked
// cases can only sample.
//
// It reads and sends in one pass: five million decoded cases would cost
// gigabytes to hold, and each one is finished with before the next is needed.
func runSweep(arg string, command []string, timeout time.Duration, manifestFile string) (bool, error) {
	def, path, err := parseSweep(arg, manifestFile)
	if err != nil {
		return false, err
	}
	dump, err := openDump(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = dump.Close() }()

	fmt.Printf("sweeping %s (%s) through %s/%s\n", def.ID, def.Authority, def.Kind, def.Country)

	// An error while reading the dump has to void the run rather than end it
	// early: a truncated sweep that refused nothing looks exactly like a
	// complete one, and that is the one confusion worth ruling out.
	var readErr error
	cases := func(yield func(*conformancev1.ConformanceCase) bool) {
		for c, err := range register.Sweep(dump, def) {
			if err != nil {
				readErr = err
				return
			}
			if !yield(c) {
				return
			}
		}
	}

	if timeout == 0 {
		timeout = runner.DefaultSweepTimeout
	}
	res, err := runner.RunStream(context.Background(), cases, runner.Options{
		Command: command, Timeout: timeout, RefusalOnly: true,
	})
	if err != nil {
		return false, err
	}
	if readErr != nil {
		return false, readErr
	}
	if res.Cases == 0 {
		return false, fmt.Errorf("the dump carried no identifier; refusing to call that a sweep")
	}

	fmt.Printf("%s: %d identifiers, %d refused\n", def.ID, res.Cases, len(res.Diffs))
	for i, d := range res.Diffs {
		if i == 20 {
			fmt.Printf("... and %d more\n", len(res.Diffs)-20)
			break
		}
		fmt.Printf("  REFUSED %s: %s\n", d.CaseID, d.Field)
	}
	if len(res.Diffs) > 0 {
		fmt.Println("a refusal here is either a regression, or the issuer now emits a form the rule " +
			"does not know; both need a person to look")
		return false, nil
	}
	fmt.Println("no identifier of the register was refused")
	return true, nil
}

func report(res runner.Result, version string, restricted bool) {
	byCase := map[string][]runner.Diff{}
	for _, d := range res.Diffs {
		byCase[d.CaseID] = append(byCase[d.CaseID], d)
	}
	ids := make([]string, 0, len(byCase))
	for id := range byCase {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		fmt.Printf("%s\n", id)
		for _, d := range byCase[id] {
			fmt.Printf("    %s: want %s, got %s\n", d.Field, d.Want, d.Got)
		}
	}

	failed := len(ids)
	fmt.Printf("\nrules %s: %d cases, %d matched, %d differed\n",
		version, res.Cases, res.Cases-failed, failed)

	switch {
	case restricted:
		fmt.Println("this run was restricted to one operation and is not a conformance verdict")
	case res.Conformant():
		fmt.Println("conformant")
	default:
		fmt.Println("not conformant")
	}
}

func operationByName(name string) (conformancev1.Operation, bool) {
	full := "OPERATION_" + upper(name)
	v, ok := conformancev1.Operation_value[full]
	return conformancev1.Operation(v), ok && v != 0
}

func upper(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 'a' - 'A'
		}
	}
	return string(out)
}

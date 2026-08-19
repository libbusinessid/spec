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
	"github.com/libbusinessid/spec/internal/runner"
)

func main() {
	corpus := flag.String("corpus", "", "path to the conformance bundle (.binpb)")
	timeout := flag.Duration("timeout", 0, "bound on the whole run")
	only := flag.String("operation", "",
		"restrict to one operation, for diagnosis only; a restricted run is never a conformance verdict")
	flag.Parse()

	ok, err := run(*corpus, flag.Args(), *timeout, *only)
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
func run(corpusPath string, command []string, timeout time.Duration, only string) (bool, error) {
	if corpusPath == "" {
		return false, fmt.Errorf("--corpus is required")
	}
	if len(command) == 0 {
		return false, fmt.Errorf("give the testee command after --")
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

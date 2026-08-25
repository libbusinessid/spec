// Command coverage enforces the line and block coverage thresholds of the
// repository from a Go coverage profile.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/entid-org/spec/internal/coverage"
)

func main() {
	os.Exit(run())
}

func run() int {
	profile := flag.String("profile", "coverage.out", "coverage profile produced by `go test -coverprofile`")
	lineMin := flag.Float64("line-min", 95, "minimum line coverage in percent")
	blockMin := flag.Float64("branch-min", 90, "minimum block coverage in percent")
	out := flag.String("out", "", "optional JSON summary path")
	flag.Parse()

	summary, err := parse(*profile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cannot read the profile: %v\n", err)
		return 3
	}
	if *out != "" {
		encoded, err := json.MarshalIndent(summary, "", "  ")
		if err == nil {
			_ = os.WriteFile(*out, append(encoded, '\n'), 0o600)
		}
	}
	failures := summary.Report(os.Stdout, *lineMin, *blockMin)
	for _, failure := range failures {
		_, _ = fmt.Fprintln(os.Stderr, failure)
	}
	if len(failures) > 0 {
		return 2
	}
	return 0
}

// parse opens the profile and closes it before the caller can exit.
func parse(path string) (coverage.Summary, error) {
	file, err := os.Open(path) //nolint:gosec // the path is supplied by the operator
	if err != nil {
		return coverage.Summary{}, err
	}
	defer func() { _ = file.Close() }()
	return coverage.Parse(file)
}

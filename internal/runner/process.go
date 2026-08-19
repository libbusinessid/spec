package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
)

// Options configures one conformance run.
type Options struct {
	// Command is the testee to spawn, argv style.
	Command []string
	// Timeout bounds the whole run. Zero means the default.
	Timeout time.Duration
}

// Result is the verdict of a run.
type Result struct {
	Cases int
	Diffs []Diff
}

// Conformant reports whether the engine matched the corpus on every case.
func (r Result) Conformant() bool { return len(r.Diffs) == 0 }

const defaultTimeout = 10 * time.Minute

// Run spawns the testee and confronts it with every case.
//
// A run that cannot complete returns an error rather than a verdict: a broken
// exchange must never be reported as conformance, nor as a mere set of
// differences.
func Run(ctx context.Context, cases []*conformancev1.ConformanceCase, opts Options) (Result, error) {
	if len(opts.Command) == 0 {
		return Result{}, fmt.Errorf("no testee command was given")
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.Command[0], opts.Command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Result{}, fmt.Errorf("cannot open the testee input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("cannot open the testee output: %w", err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("cannot start the testee %q: %w", opts.Command[0], err)
	}

	diffs, sessErr := runSession(stdin, stdout, cases)
	_ = stdin.Close()
	waitErr := cmd.Wait()

	if sessErr != nil {
		return Result{}, withStderr(sessErr, stderr.String())
	}
	if waitErr != nil {
		return Result{}, withStderr(fmt.Errorf("the testee exited with an error: %w", waitErr), stderr.String())
	}
	return Result{Cases: len(cases), Diffs: diffs}, nil
}

// withStderr attaches what the testee printed, which is usually where the real
// cause is.
func withStderr(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w\nthe testee wrote on stderr:\n%s", err, stderr)
}

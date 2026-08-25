package runner

import (
	"context"
	"fmt"
	"iter"
	"os/exec"
	"slices"
	"strings"
	"time"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
)

// Options configures one conformance run.
type Options struct {
	// Command is the testee to spawn, argv style.
	Command []string
	// Timeout bounds the whole run. Zero means the default.
	Timeout time.Duration
	// RefusalOnly reduces every comparison to the one question a register can
	// settle: was this identifier refused. It exists for register sweeps and
	// for nothing else; see the note in compare. A normal corpus run must leave
	// it false, because a corpus case states the canonical value, the reason
	// code and the checksum, and letting a run ignore them would turn
	// conformance into a weaker claim than it looks.
	RefusalOnly bool
}

// Result is the verdict of a run.
type Result struct {
	Cases int
	Diffs []Diff
}

// Conformant reports whether the engine matched the corpus on every case.
func (r Result) Conformant() bool { return len(r.Diffs) == 0 }

// defaultTimeout bounds a corpus run, which is a few seconds of work: a run
// that takes minutes has hung. A register sweep is a different shape of job -
// tens of millions of cases, tens of minutes - and carries its own budget.
const defaultTimeout = 10 * time.Minute

// defaultSweepTimeout bounds a register sweep. The largest register in the
// manifest, the forty four million SIRET, takes about ten minutes; the budget
// leaves room for a register several times larger without leaving a hung run
// to sit forever.
const defaultSweepTimeout = 2 * time.Hour

// DefaultSweepTimeout is defaultSweepTimeout, for the command line.
const DefaultSweepTimeout = defaultSweepTimeout

// Run spawns the testee and confronts it with every case.
//
// A run that cannot complete returns an error rather than a verdict: a broken
// exchange must never be reported as conformance, nor as a mere set of
// differences.
func Run(ctx context.Context, cases []*conformancev1.ConformanceCase, opts Options) (Result, error) {
	return RunStream(ctx, slices.Values(cases), opts)
}

// RunStream is Run over a stream of cases.
//
// A register sweep confronts an engine with every identifier its issuer has
// ever handed out - 5695465 of them for Companies House alone - and holding
// that many decoded cases in memory costs gigabytes for no reason: each one is
// sent, answered and finished with before the next is needed. The stream form
// keeps a sweep flat in memory, and the whole corpus is still a slice because
// it is small and read from a file.
func RunStream(ctx context.Context, cases iter.Seq[*conformancev1.ConformanceCase], opts Options) (Result, error) {
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

	diffs, sent, sessErr := runSession(stdin, stdout, cases, opts.RefusalOnly)
	_ = stdin.Close()
	waitErr := cmd.Wait()

	// A run that outlives its budget kills the testee, and the session then
	// sees the pipe close and reports that the testee stopped answering. That
	// reads as a crashed engine and sends the reader hunting a bug that is not
	// there. Register sweeps made this routine rather than rare: the SIRET
	// sweep runs for about ten minutes, which is exactly the default budget.
	if ctx.Err() != nil {
		return Result{}, fmt.Errorf(
			"the run outlived its %s budget after %d cases; raise --timeout", timeout, sent)
	}
	if sessErr != nil {
		return Result{}, withStderr(sessErr, stderr.String())
	}
	if waitErr != nil {
		return Result{}, withStderr(fmt.Errorf("the testee exited with an error: %w", waitErr), stderr.String())
	}
	return Result{Cases: sent, Diffs: diffs}, nil
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

package artifact

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// RULES_VERSION names a month that has begun.
//
// Section 7.4 makes it YYYY.MM.PATCH, where YYYY.MM is when the business data
// was cut and PATCH is a counter within that month. The counter has no upper
// bound, and treating it as a day is the mistake this guards: 2026.08.31 was
// followed by 2026.09.0 in August, so four versions claimed September while the
// data was cut five weeks earlier.
//
// tools/next_rules_version.sh derives the next value from every version the
// file has ever held, so it cannot roll over and cannot reuse a version an
// engine has pinned. This is what catches the day someone edits the file by
// hand instead.
func TestRulesVersionNamesAMonthThatHasBegun(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "RULES_VERSION"))
	if err != nil {
		t.Fatalf("reading RULES_VERSION: %v", err)
	}
	version := strings.TrimSpace(string(raw))
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		t.Fatalf("RULES_VERSION is %q, want YYYY.MM.PATCH", version)
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		t.Fatalf("RULES_VERSION year %q: %v", parts[0], err)
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("RULES_VERSION month %q: %v", parts[1], err)
	}
	if month < 1 || month > 12 {
		t.Fatalf("RULES_VERSION names month %d, which is not one", month)
	}
	if _, err := strconv.Atoi(parts[2]); err != nil {
		t.Fatalf("RULES_VERSION patch %q: %v", parts[2], err)
	}

	// The commit date, never the wall clock: the version names the month the
	// data was cut, and this repository judges "now" by SOURCE_DATE_EPOCH or by
	// the commit, which is also what forbidigo requires of every artifact.
	now := referenceDate(t)
	if year > now.Year() || (year == now.Year() && month > int(now.Month())) {
		t.Errorf("RULES_VERSION is %q, naming a month that has not begun: today is %s.\n"+
			"The third component is a counter, not a day, and has no upper bound: "+
			"2026.08.31 is followed by 2026.08.32. Run tools/next_rules_version.sh --write.",
			version, now.Format("2006.01"))
	}
}

// referenceDate is SOURCE_DATE_EPOCH when set, and the date of HEAD otherwise.
// A test that read the wall clock would judge the month it happens to run in
// rather than the month the data was cut, and would answer differently on a
// machine whose clock is wrong.
func referenceDate(t *testing.T) time.Time {
	t.Helper()
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		seconds, err := strconv.ParseInt(epoch, 10, 64)
		if err != nil {
			t.Fatalf("SOURCE_DATE_EPOCH is %q: %v", epoch, err)
		}
		return time.Unix(seconds, 0).UTC()
	}
	out, err := exec.Command("git", "log", "-1", "--pretty=%ct").Output()
	if err != nil {
		t.Skipf("no SOURCE_DATE_EPOCH and no git history to date the commit: %v", err)
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		t.Fatalf("the commit date is %q: %v", strings.TrimSpace(string(out)), err)
	}
	return time.Unix(seconds, 0).UTC()
}

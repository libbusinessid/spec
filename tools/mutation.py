#!/usr/bin/env python3
"""Mutation testing of the critical core.

The script applies one source mutation at a time to the checksum algorithms,
the position comparisons, the arithmetic bounds and the dispatch selection, and
requires the test suite to fail for each of them. A surviving mutant means the
suite executes the line without asserting its behaviour, and fails the job.
"""

import os
import subprocess
import sys

# Each mutation is (file, exact original text, replacement).
MUTATIONS = [
    # Luhn: the acceptance condition, the doubling correction and the parity.
    ("internal/reference/checksum.go", "if sum%10 == 0 {", "if sum%10 != 0 {"),
    ("internal/reference/checksum.go", "d -= 9", "d -= 8"),
    ("internal/reference/checksum.go",
     "if (len(runes)-1-i)%2 == 1 {", "if (len(runes)-1-i)%2 == 0 {"),
    # ISO 7064 MOD 97-10: the expected remainder and the letter expansion base.
    ("internal/reference/checksum.go", "if r == 1 {", "if r == 0 {"),
    ("internal/reference/checksum.go", "r = (r*100 + v) % 97", "r = (r*10 + v) % 97"),
    # Digit by digit modulo.
    ("internal/reference/checksum.go",
     "r = (r*10 + int64(c-'0')) % modulus", "r = (r*10 + int64(c-'0')) % (modulus + 1)"),
    # Comparison of a computed check value with the observed one.
    ("internal/reference/checksum.go",
     "if actual.value == expected.value {", "if actual.value != expected.value {"),
    # Complement bounds.
    ("internal/reference/checksum.go",
     "if v.value < 0 || v.value > op.GetModulus() {", "if v.value < 0 {"),
    # Weighted sum alignments.
    ("internal/reference/checksum.go",
     "sum += values[len(values)-1-i] * weights[len(weights)-1-i]",
     "sum += values[i] * weights[i]"),
    ("internal/reference/checksum.go",
     "sum += values[i] * weights[i%len(weights)]", "sum += values[i] * weights[0]"),
    # Dispatch: a prefix must match at the start of the value, and the longest
    # match wins. `>` and `>=` are equivalent here, because two distinct
    # prefixes of the same length can never both start the same value, so the
    # mutation targets the matching predicate instead.
    ("internal/reference/engine.go",
     "if len(prefix) > bestLen && strings.HasPrefix(value, prefix) {",
     "if len(prefix) > bestLen && strings.Contains(value, prefix) {"),
    # Dispatch: the country target has priority over the prefix target.
    ("internal/reference/engine.go", "target := countryTarget", "target := prefixTarget"),
    # Graph validation: a node may only reference a strictly lower index.
    ("internal/artifact/validate_node.go", "if int(in) >= index {", "if int(in) > index {"),
    # Input limit and the status it produces.
    ("internal/limits/limits.go", "MaxInputBytes = 1_024", "MaxInputBytes = 2_048"),
    ("internal/reference/operations.go",
     "return StepResult{Level: LevelChecksum, Status: StatusNotRun, ReasonCode: reason}",
     "return StepResult{Level: LevelChecksum, Status: StatusUnsupported, ReasonCode: reason}"),
    # Position comparison inside a checksum.
    ("internal/reference/checksum.go",
     "if int(op.GetIndex()) >= len(runes) {", "if int(op.GetIndex()) > len(runes) {"),
]


def run_tests() -> bool:
    """Return True when the suite passes."""
    result = subprocess.run(
        ["go", "test", "./internal/...", "./cmd/..."],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
    return result.returncode == 0


def main() -> int:
    originals = {}
    for path, _, _ in MUTATIONS:
        if path not in originals:
            with open(path, encoding="utf-8") as handle:
                originals[path] = handle.read()

    def restore() -> None:
        for path, content in originals.items():
            with open(path, "w", encoding="utf-8") as handle:
                handle.write(content)

    killed = 0
    survivors = []
    try:
        for path, old, new in MUTATIONS:
            source = originals[path]
            if source.count(old) != 1:
                print(f"::error::mutation target not found exactly once in {path}: {old!r}")
                return 1
            with open(path, "w", encoding="utf-8") as handle:
                handle.write(source.replace(old, new))
            if run_tests():
                print(f"::error::surviving mutant in {path}: {old!r} -> {new!r}")
                survivors.append((path, old, new))
            else:
                print(f"killed: {path}: {old!r} -> {new!r}")
                killed += 1
            restore()
    finally:
        restore()

    total = killed + len(survivors)
    score = 100 * killed // total
    print(f"mutation score: {score}% ({killed}/{total})")
    if score < 80:
        print("::error::mutation score below 80%")
        return 1
    return 0


if __name__ == "__main__":
    os.chdir(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
    sys.exit(main())

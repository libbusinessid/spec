#!/usr/bin/env bash
# Fail when the committed generated code is not what the pinned generators
# produce, and say what to do about it.
#
# This script never tolerates a difference. It only turns the raw diff into
# an actionable message, because the usual cause is a generator bump in
# `tools/pinned/go.mod` that left the version comment of the generated files
# behind.
#
# Usage: check_generated.sh [path...]
set -euo pipefail

paths=("$@")
if [[ ${#paths[@]} -eq 0 ]]; then
  paths=(gen proto testdata)
fi

# An untracked file is a difference too: a new schema produces a new
# generated file, which `git diff` alone would never report.
untracked="$(git ls-files --others --exclude-standard -- "${paths[@]}")"

if git diff --quiet -- "${paths[@]}" && [[ -z "${untracked}" ]]; then
  echo "the committed generated code matches the pinned generators"
  exit 0
fi

diff_output="$(git diff -- "${paths[@]}")"
changed="$(git diff --name-only -- "${paths[@]}")"
if [[ -n "${untracked}" ]]; then
  changed="$(printf '%s\n%s' "${changed}" "${untracked}" | sed '/^$/d')"
fi

echo "::error::the committed generated code does not match the pinned generators; run 'make generate' and commit the result"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "### The committed generated code is stale"
    echo
    echo "Run \`make generate\` and commit the result."
    echo
    echo "The generated files record which plugin produced them, so bumping a"
    echo "generator pinned in \`tools/pinned/go.mod\` without regenerating leaves"
    echo "that record stale. Regenerating is what the specification requires:"
    echo "section 12.3 asks a Protobuf update to rebuild the artifacts and to"
    echo "explain the byte differences, so state in the pull request whether the"
    echo "published digests moved."
    echo
    echo "Files out of date:"
    echo
    echo '```'
    echo "${changed}"
    echo '```'
    echo
    echo "<details><summary>Diff</summary>"
    echo
    echo '```diff'
    printf '%s\n' "${diff_output}" | sed -n '1,200p'
    echo '```'
    echo
    echo "</details>"
  } >>"${GITHUB_STEP_SUMMARY}"
fi

printf '%s\n' "${diff_output}"
exit 1

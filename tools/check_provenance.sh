#!/usr/bin/env bash
# Fail when a modified rule file does not carry provenance, or when a rule
# change is not accompanied by the output of `entidc diff`.
#
# Usage: check_provenance.sh <base-ref>
set -euo pipefail

base="${1:-origin/main}"
git fetch --quiet origin "${base#origin/}" 2>/dev/null || true

changed_rules="$(git diff --name-only "${base}"...HEAD -- 'rules/**/*.hcl' || true)"
if [[ -z "${changed_rules}" ]]; then
  echo "no rule changed"
  exit 0
fi

echo "changed rules:"
echo "${changed_rules}"

status=0
for file in ${changed_rules}; do
  [[ -f "${file}" ]] || continue
  if grep -q '^\s*identifier "' "${file}" && ! grep -q '^\s*source {' "${file}"; then
    echo "::error file=${file}::a definition without a source block is refused"
    status=1
  fi
done

# A rule change must also change the conformance corpus.
#
# A change that leaves behaviour untouched is the one exception, and it needs a
# way out that is not "add a case until the job goes green": a case written to
# silence CI teaches nothing and makes the corpus worse. The way out is a
# trailer naming the reason, so the exception is stated in the history rather
# than hidden in a filler case. `entidc diff` cannot settle this on its
# own - it reports that the program changed, which is true and says nothing
# about behaviour.
exempt="$(git log --format=%B "${base}"...HEAD | sed -n 's/^No-Behaviour-Change:[[:space:]]*//p' | head -1)"
changed_cases="$(git diff --name-only "${base}"...HEAD -- 'conformance/**/*.jsonl' || true)"
if [[ -z "${changed_cases}" ]]; then
  if [[ -n "${exempt}" ]]; then
    echo "::notice::no conformance case changed, declared as behaviour preserving: ${exempt}"
  else
    echo "::error::a rule change must come with conformance cases, or a commit"
    echo "::error::trailer 'No-Behaviour-Change: <reason>' when behaviour is preserved"
    status=1
  fi
fi

# Publish the classified diff between the base bundle and the new one.
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT
export SOURCE_DATE_EPOCH="$(git log -1 --pretty=%ct)"
go run ./cmd/entidc compile --out "${tmp}/new" >/dev/null

worktree="${tmp}/base"
git worktree add --quiet --detach "${worktree}" "${base}"
( cd "${worktree}" && go run ./cmd/entidc compile --out "${tmp}/old" >/dev/null ) || {
  echo "the base revision does not compile; skipping the diff"
  git worktree remove --force "${worktree}"
  exit "${status}"
}
git worktree remove --force "${worktree}"

version="$(cat RULES_VERSION)"
base_version="$(git show "${base}:RULES_VERSION" | tr -d '[:space:]')"
echo "### entidc diff"
go run ./cmd/entidc diff \
  "${tmp}/old/entid-rules-${base_version}.binpb" \
  "${tmp}/new/entid-rules-${version}.binpb" | tee "${tmp}/diff.txt"

if grep -q '^restriction' "${tmp}/diff.txt"; then
  echo "::warning::this pull request restricts an existing rule and requires a reinforced review"
  if [[ -n "${exempt}" ]]; then
    echo "::error::a restriction is a change of behaviour; No-Behaviour-Change does not apply"
    status=1
  fi
fi

exit "${status}"

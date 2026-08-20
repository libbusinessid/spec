#!/usr/bin/env bash
# Fail when the business data changed without RULES_VERSION changing with it.
#
# Section 7.4 makes rules_version the version of the business data. Two bundles
# carrying the same rules_version and different bytes is the failure this
# prevents: an engine pins a version, receives different rules under it, and has
# no way to notice.
#
# Usage: check_rules_version.sh <base-ref>
set -euo pipefail

base="${1:-origin/main}"
git fetch --quiet origin "${base#origin/}" 2>/dev/null || true

# What matters is what reaches the bundle, not what sits in the directory. The
# canonical stream of section 7.2 carries the rule sources, the JSONL cases and
# the binary fixtures they reference; anything else under those paths is
# tooling, and demanding a version bump for it would train people to bump the
# version for nothing - which is exactly how a bump stops meaning anything.
changed="$(git diff --name-only "${base}"...HEAD \
  -- 'rules/**/*.hcl' 'conformance/**/*.jsonl' 'testdata/bundles/**' || true)"
if [[ -z "${changed}" ]]; then
  echo "no business data changed"
  exit 0
fi

echo "business data changed:"
while IFS= read -r f; do printf '  %s\n' "${f}"; done <<< "${changed}"

if [[ -z "$(git diff --name-only "${base}"...HEAD -- RULES_VERSION)" ]]; then
  current="$(cat RULES_VERSION)"
  echo "::error::the rules or the conformance cases changed while RULES_VERSION stayed at ${current}."
  echo "::error::section 7.4 versions the business data with rules_version, so two bundles must never carry the same version and different bytes."
  exit 1
fi

echo "RULES_VERSION moved to $(cat RULES_VERSION)"

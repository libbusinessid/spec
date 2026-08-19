#!/usr/bin/env bash
# Open the rule synchronization pull request in one engine repository.
#
# The automation can never approve or merge its own pull request: it only pushes
# a branch and opens the request.
#
# Usage: open_downstream_pr.sh <owner/repo> <rules-version> <tag>
set -euo pipefail

repo="$1"
version="$2"
tag="$3"
branch="rules/${version}"

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

gh repo clone "${repo}" "${work}/engine" -- --depth 1
cd "${work}/engine"

git switch -c "${branch}"

resources="$(cat <<'EOF'
businessid-go:internal/rules/businessid-rules.binpb:internal/rules/businessid-conformance.binpb
businessid-swift:Sources/BusinessID/Resources/businessid-rules.binpb:Tests/BusinessIDTests/Resources/businessid-conformance.binpb
businessid-kotlin:src/main/resources/businessid-rules.binpb:src/test/resources/businessid-conformance.binpb
businessid-typescript:src/rules/businessid-rules.binpb:test/resources/businessid-conformance.binpb
EOF
)"

name="${repo##*/}"
line="$(printf '%s\n' "${resources}" | grep "^${name}:")"
rules_path="$(printf '%s' "${line}" | cut -d: -f2)"
conformance_path="$(printf '%s' "${line}" | cut -d: -f3)"

mkdir -p "$(dirname "${rules_path}")" "$(dirname "${conformance_path}")"
cp "${GITHUB_WORKSPACE}/artifacts/businessid-rules-${version}.binpb" "${rules_path}"
cp "${GITHUB_WORKSPACE}/artifacts/businessid-conformance-${version}.binpb" "${conformance_path}"
cp "${GITHUB_WORKSPACE}/rules.lock" rules.lock

git add "${rules_path}" "${conformance_path}" rules.lock
git -c user.name="libbusinessid-bot" -c user.email="bot@libbusinessid.invalid" \
  commit -m "rules: update to ${version}"
git push --set-upstream origin "${branch}"

gh pr create \
  --repo "${repo}" \
  --title "rules: update to ${version}" \
  --body "$(cat <<BODY
Automated synchronization of the LibBusinessID rules.

- rules version: \`${version}\`
- source tag: \`${tag}\`
- artifacts verified: SHA-256 and GitHub artifact attestation (owner, repository, workflow, commit and tag)

The engine CI must run the whole conformance suite before this pull request can
be merged. This automation cannot approve or merge its own pull request.
BODY
)"

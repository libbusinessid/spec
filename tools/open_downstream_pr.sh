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

# gh authenticates its own API calls from GH_TOKEN, but git does not: the clone
# succeeded, the branch and the commit were made, and the push then asked for a
# username on a runner with no terminal. This wires the token into git's
# credential helper for the whole script.
gh auth setup-git
gh repo clone "${repo}" "${work}/engine" -- --depth 1
cd "${work}/engine"

git switch -c "${branch}"

# The bundle is an input to the engine's generator, so it lands under spec/
# beside the schemas and the corpus -- the same place tools/sync_engines.sh
# writes, and the same place the engine's own lock attests. This script used to
# copy it into Sources/BusinessID/Resources and src/main/resources, which is the
# bundle as a resource of the published package: engine.md section 1.2 forbids
# exactly that, and the guard that catches the phrase read the documents and
# never the tooling.
mkdir -p spec
cp "${GITHUB_WORKSPACE}/artifacts/businessid-rules-${version}.binpb" spec/businessid-rules.binpb
cp "${GITHUB_WORKSPACE}/artifacts/businessid-conformance-${version}.binpb" spec/businessid-conformance.binpb
gzip -dc "${GITHUB_WORKSPACE}/artifacts/businessid-conformance-${version}.jsonl.gz" > spec/businessid-conformance.jsonl
for schema in rules.proto conformance.proto testee.proto ir.md features.md; do
  cp "${GITHUB_WORKSPACE}/artifacts/${schema}" "spec/${schema}"
done
cp "${GITHUB_WORKSPACE}/rules.lock" rules.lock
git add spec rules.lock
git -c user.name="libbusinessid-bot" -c user.email="bot@libbusinessid.invalid" \
  commit -m "rules: update to ${version}"
# A re-run after a fixed defect has to replace its own branch rather than be
# refused as non fast forward, and --force-with-lease is the safe way to do it.
# But a lease needs something to compare against: in a shallow clone that never
# fetched this branch git refuses with "stale info", which is what a plain
# --force-with-lease did on the second attempt. So the branch is fetched first
# when it exists, and pushed plainly when it does not.
if git ls-remote --exit-code --heads origin "${branch}" >/dev/null 2>&1; then
  git fetch --depth 1 origin "${branch}"
  git push --force-with-lease --set-upstream origin "${branch}"
else
  git push --set-upstream origin "${branch}"
fi

# --head and --base explicitly: gh infers them from the checkout it is run in,
# and inside a shallow clone of another repository it refused with "you must
# first push the current branch to a remote" on a branch it had just pushed.
gh pr create \
  --repo "${repo}" \
  --head "${branch}" \
  --base main \
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

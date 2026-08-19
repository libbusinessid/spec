#!/usr/bin/env bash
# Compare the rules.lock published by each engine with the latest release of
# this repository. Run by the scheduled workflow.
set -euo pipefail

owner="${GITHUB_REPOSITORY_OWNER:-libbusinessid}"
version="$(cat RULES_VERSION)"
status=0

for engine in businessid-go businessid-swift businessid-kotlin businessid-typescript; do
  lock="$(gh api "repos/${owner}/${engine}/contents/rules.lock" \
    --jq '.content' 2>/dev/null | base64 --decode || true)"
  if [[ -z "${lock}" ]]; then
    echo "::warning::${engine} publishes no rules.lock"
    status=1
    continue
  fi
  engine_version="$(printf '%s' "${lock}" | sed -n 's/^rules_version = "\(.*\)"$/\1/p')"
  if [[ "${engine_version}" != "${version}" ]]; then
    echo "::warning::${engine} is on rules ${engine_version}, the spec publishes ${version}"
    status=1
  else
    echo "ok ${engine} ${engine_version}"
  fi
done

exit "${status}"

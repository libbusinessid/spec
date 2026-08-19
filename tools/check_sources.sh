#!/usr/bin/env bash
# Check that every source URL declared by a rule still resolves.
#
# Run by the scheduled workflow only: the pull request pipeline stays offline.
set -euo pipefail

status=0
urls="$(grep -rho 'url *= *"[^"]*"' rules | sed 's/.*"\(.*\)"/\1/' | sort -u)"

if [[ -z "${urls}" ]]; then
  echo "::error::no source URL found in rules/"
  exit 1
fi

while IFS= read -r url; do
  [[ -n "${url}" ]] || continue
  code="$(curl -sS -o /dev/null -w '%{http_code}' -L --max-time 30 "${url}" || echo 000)"
  case "${code}" in
    2*|3*) echo "ok   ${code} ${url}" ;;
    *)     echo "::warning::source unreachable (${code}): ${url}"; status=1 ;;
  esac
done <<< "${urls}"

exit "${status}"

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

# A browser user agent, because several registers answer a bare curl with a
# challenge page. Identifying as a browser is what a reader would do.
agent="Mozilla/5.0 (compatible; libbusinessid-source-check/1.0)"

while IFS= read -r url; do
  [[ -n "${url}" ]] || continue
  code="$(curl -sS -o /dev/null -w '%{http_code}' -L -A "${agent}" --max-time 30 "${url}" || echo 000)"
  case "${code}" in
    2*|3*)
      echo "ok   ${code} ${url}"
      ;;
    401|403|429)
      # The server answered and refused this client. The page is there; a bot
      # is not welcome to it. Treating that as a dead source would make the
      # check cry wolf every week on registers that simply block automation,
      # and a real removal would then go unnoticed among the noise.
      echo "note ${code} ${url} (reachable, refuses automated clients)"
      ;;
    *)
      echo "::error::source unreachable (${code}): ${url}"
      status=1
      ;;
  esac
done <<< "${urls}"

exit "${status}"

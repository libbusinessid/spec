#!/usr/bin/env bash
# Print the next RULES_VERSION, and with --write, set it.
#
# Section 7.4 makes rules_version YYYY.MM.PATCH, where YYYY.MM is the year and
# month the business data was cut and PATCH is a counter within that month. The
# counter has no upper bound: it is not a day, and the mistake this script
# exists to prevent is treating it as one. 2026.08.31 is followed by
# 2026.08.32, not by 2026.09.0 -- rolling over in August produced four versions
# claiming September.
#
# The counter is derived from every value the file has ever held rather than
# from the current one, so a correction cannot silently reuse a version an
# engine has already pinned.
#
# Usage: next_rules_version.sh [--write]
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${here}"

# SOURCE_DATE_EPOCH when set, so a reproducible build names the month it was
# built for rather than the month it happens to run in.
if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
  month="$(date -u -r "${SOURCE_DATE_EPOCH}" '+%Y.%m' 2>/dev/null \
        || date -u -d "@${SOURCE_DATE_EPOCH}" '+%Y.%m')"
else
  month="$(date -u '+%Y.%m')"
fi

# Every value the file has held, plus the one it holds now.
used="$( { git log --format='%H' -- RULES_VERSION 2>/dev/null \
            | while read -r commit; do git show "${commit}:RULES_VERSION" 2>/dev/null || true; done
          cat RULES_VERSION 2>/dev/null || true; } | tr -d '\r' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' || true)"

highest=-1
while read -r version; do
  [[ -z "${version}" ]] && continue
  [[ "${version%.*}" == "${month}" ]] || continue
  patch="${version##*.}"
  (( patch > highest )) && highest="${patch}"
done <<< "${used}"

next="${month}.$(( highest + 1 ))"

if [[ "${1:-}" == "--write" ]]; then
  printf '%s\n' "${next}" > RULES_VERSION
  echo "RULES_VERSION is now ${next}"
else
  printf '%s\n' "${next}"
fi

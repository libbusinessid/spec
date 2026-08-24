#!/usr/bin/env bash
# Assemble one engine's spec/PROVENANCE.md.
#
# It had two writers and drifted between them, which is the shape of defect this
# repository keeps finding: tools/sync_engines.sh assembled it for a developer
# sync while the release automation copied nine files and not this one, so a
# released engine named commit 4bf7699 and rules 2026.08.32 beside a lock naming
# b264614 and 2026.08.33. Two engines reported it independently. One writer now.
#
# The figures come from the bundle rather than from anyone's memory of it: the
# four hand written files this replaced still described seven definitions and
# 185 IR nodes when the bundle carried ninety four and 2375.
#
# Usage: write_provenance.sh <spec-root> <dist-dir> <version> <commit> <engine> <output>
set -euo pipefail
here="$1"
dist="$2"
version="$3"
commit="$4"
engine="$5"
output="$6"
lang="${engine#businessid-}"

inspect="$(cd "${here}" && go run ./cmd/businessidc inspect "${dist}/businessid-rules-${version}.binpb")"
definitions="$(printf '%s\n' "${inspect}" | sed -n 's/^identifiers *\([0-9]*\).*/\1/p')"
nodes="$(printf '%s\n' "${inspect}" | sed -n 's/^programs *[0-9]* (\([0-9]*\) nodes).*/\1/p')"
capabilities="$(printf '%s\n' "${inspect}" | grep -cE '^ +[0-9]+ [A-Z]')"
unused_ops="$(sed -n 's/^\([0-9]*\) of [0-9]* operations.*/\1/p' "${here}/docs/generated/coverage.md" | head -1)"
total_ops="$(sed -n 's/^[0-9]* of \([0-9]*\) operations.*/\1/p' "${here}/docs/generated/coverage.md" | head -1)"
used_ops="$((total_ops - unused_ops))"

{
  printf '# Where these files come from, and what to build\n\n'
  printf 'Copied from `github.com/libbusinessid/spec` at commit\n'
  printf '`%s`, rules version\n`%s`, stability `%s`.\n\n' \
    "${commit}" "${version}" "$(jq -r .stability "${dist}/businessid-manifest-${version}.json")"
  sed -e "s/{{DEFINITIONS}}/${definitions}/g" \
      -e "s/{{NODES}}/${nodes}/g" \
      -e "s/{{CAPABILITIES}}/${capabilities}/g" \
      -e "s/{{USED_OPS}}/${used_ops}/g" \
      -e "s/{{TOTAL_OPS}}/${total_ops}/g" \
      "${here}/docs/spec/provenance/body.md"
  printf '\n'
  cat "${here}/docs/spec/provenance/${lang}.md"
} > "${output}"

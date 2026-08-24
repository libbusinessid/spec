#!/usr/bin/env bash
# Copy the compiled artifacts of this repository into the engine checkouts.
#
# Before the first release the engines carry a local copy of the bundle rather
# than a downloaded, attested one. Keeping that copy in step was a manual job,
# and it drifted: the PROVENANCE.md header of three engines still named
# 2026.08.0 while their rules.lock had moved to 2026.08.2. Everything a resync
# has to touch is written here so that stops happening.
#
# It only ever writes inside the sibling checkouts. It creates no branch, no
# commit and no pull request: publishing stays a human decision.
#
# Usage: sync_engines.sh [engine-root]   (default: the parent of this repository)
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
root="${1:-$(dirname "${here}")}"
version="$(cat "${here}/RULES_VERSION")"
commit="$(git -C "${here}" rev-parse HEAD)"
dist="${here}/dist"

if [[ ! -f "${dist}/businessid-rules-${version}.binpb" ]]; then
  echo "error: ${dist} carries no bundle for ${version}; run businessidc compile --release first" >&2
  exit 1
fi

sha() { sha256sum "$1" | cut -d' ' -f1; }

# The figures PROVENANCE.md quotes come from the bundle, not from anyone's
# memory of it.
inspect="$(cd "${here}" && go run ./cmd/businessidc inspect "${dist}/businessid-rules-${version}.binpb")"
definitions="$(printf '%s\n' "${inspect}" | sed -n 's/^identifiers *\([0-9]*\).*/\1/p')"
nodes="$(printf '%s\n' "${inspect}" | sed -n 's/^programs *[0-9]* (\([0-9]*\) nodes).*/\1/p')"
capabilities="$(printf '%s\n' "${inspect}" | grep -cE '^ +[0-9]+ [A-Z]')"
unused_ops="$(sed -n 's/^\([0-9]*\) of [0-9]* operations.*/\1/p' "${here}/docs/generated/coverage.md" | head -1)"
total_ops="$(sed -n 's/^[0-9]* of \([0-9]*\) operations.*/\1/p' "${here}/docs/generated/coverage.md" | head -1)"
used_ops="$((total_ops - unused_ops))"

for engine in businessid-go businessid-swift businessid-kotlin businessid-typescript; do
  target="${root}/${engine}"
  [[ -d "${target}/spec" ]] || { echo "skip ${engine}: no spec directory"; continue; }

  cp "${dist}/businessid-rules-${version}.binpb"       "${target}/spec/businessid-rules.binpb"
  cp "${dist}/businessid-conformance-${version}.binpb" "${target}/spec/businessid-conformance.binpb"
  gzip -dc "${dist}/businessid-conformance-${version}.jsonl.gz" \
                                                       > "${target}/spec/businessid-conformance.jsonl"
  for f in rules.proto conformance.proto testee.proto ir.md features.md; do
    cp "${dist}/${f}" "${target}/spec/${f}"
  done
  cp "${here}/docs/spec/spec.md" "${target}/spec/spec.md"

  # The contracts an engine is written against, which it cannot read from here.
  # They were left out of the first version of this script, and the TypeScript
  # engine found it the only way anyone could: by being told to read documents
  # its checkout did not have.
  cp "${here}/docs/spec/engine.md" "${target}/spec/engine.md"
  case "${engine}" in
    businessid-go)         cp "${here}/docs/spec/engine-go.md"         "${target}/spec/engine-go.md" ;;
    businessid-swift)      cp "${here}/docs/spec/engine-swift.md"      "${target}/spec/engine-swift.md" ;;
    businessid-kotlin)     cp "${here}/docs/spec/engine-kotlin.md"     "${target}/spec/engine-kotlin.md" ;;
    businessid-typescript) cp "${here}/docs/spec/engine-typescript.md" "${target}/spec/engine-typescript.md" ;;
  esac

  cat > "${target}/rules.lock" <<LOCK
# Pre-release lock, produced locally from the spec repository.
#
# \`attestation_identity\` is deliberately absent: no release has been tagged
# yet, so there is no attested workflow identity to record. Once the first
# release exists, the downstream pull request replaces this file with an
# attested one, and the generator resolves the bundle from that release
# instead of the local copy under spec/.
rules_version = "${version}"
format_version = 1
rules_sha256 = "$(sha "${dist}/businessid-rules-${version}.binpb")"
conformance_sha256 = "$(sha "${dist}/businessid-conformance-${version}.binpb")"
# The JSONL is shipped decompressed, so its digest is taken on what lands in
# spec/ rather than on the archive. It went unlisted for a while: an engine
# could not verify it, verify-lock.sh would not have noticed a drift, and
# engine tests cite its case ids as provenance. Found by the Swift engine.
#
# Decompressed is also the only stable choice. The archive embeds a timestamp
# taken from SOURCE_DATE_EPOCH, so its digest moves with the source commit while
# its content does not: reproducible at a fixed commit, different across two.
#
# And expect this field to stay put while conformance_sha256 moves. That looks
# exactly like the drift it was added to catch, and is not: the JSONL is the
# reviewed source and carries no rules version, while the compiled corpus injects
# one into every expected report. A version bump alone moves one and not the
# other. Both measured by the Swift engine, on the release archive.
conformance_jsonl_sha256 = "$(gzip -dc "${dist}/businessid-conformance-${version}.jsonl.gz" | sha256sum | cut -d' ' -f1)"
rules_proto_sha256 = "$(sha "${dist}/rules.proto")"
conformance_proto_sha256 = "$(sha "${dist}/conformance.proto")"
testee_proto_sha256 = "$(sha "${dist}/testee.proto")"
ir_doc_sha256 = "$(sha "${dist}/ir.md")"
features_doc_sha256 = "$(sha "${dist}/features.md")"
stability = "$(jq -r .stability "${dist}/businessid-manifest-${version}.json")"
source_commit = "${commit}"
LOCK

  # One writer, in tools/write_provenance.sh: this used to assemble it here
  # while the release automation copied nine files and not this one, so a
  # released engine named one commit beside a lock naming another.
  "${here}/tools/write_provenance.sh" "${here}" "${dist}" "${version}" \
    "${commit}" "${engine}" "${target}/spec/PROVENANCE.md"

  # The header of PROVENANCE.md names the same commit and version as the lock.
  # It is the first thing a reader of the engine repository sees, so a stale
  # header misstates which rules the engine was generated from.
  prov="${target}/spec/PROVENANCE.md"
  if [[ -f "${prov}" ]]; then
    python3 - "${prov}" "${commit}" "${version}" <<'PY'
import re, sys
path, commit, version = sys.argv[1:4]
with open(path, encoding="utf-8") as f:
    text = f.read()
text = re.sub(r"`[0-9a-f]{40}`, rules version\n`[^`]*`",
              "`%s`, rules version\n`%s`" % (commit, version), text, count=1)
text = re.sub(r"`[0-9a-f]{40}`, rules version `[^`]*`",
              "`%s`, rules version `%s`" % (commit, version), text, count=1)
with open(path, "w", encoding="utf-8") as f:
    f.write(text)
PY
  fi

  echo "synced ${engine} to ${version}"
done

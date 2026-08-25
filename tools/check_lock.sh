#!/usr/bin/env bash
# Verify that a generated rules.lock matches the downloaded artifacts.
#
# Usage: check_lock.sh <rules.lock> <artifact-dir> <rules-version>
#
# This script used to hand-list the digests it checked, and it listed seven of
# the eight. The one it skipped was conformance_jsonl_sha256 -- the very field
# whose absence from one writer made the first release ship a lock of seven
# digests where every engine verifies eight. A checker that silently covers less
# than the lock declares is worse than no checker: it reports ok and exits 0 on a
# corrupted artifact.
#
# So the list is no longer written here. Every *_sha256 the lock declares must
# map to a file, every mapping must be checked, and a digest this script cannot
# place is an error rather than a line it walks past.
set -euo pipefail

lock="$1"
dir="$2"
version="$3"

# sha256sum is GNU; macOS ships shasum. A developer running this locally must get
# a verdict, not "command not found" swallowed by a pipeline.
if command -v sha256sum >/dev/null 2>&1; then
  digest_of() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  digest_of() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  echo "::error::neither sha256sum nor shasum is available" >&2
  exit 1
fi

value() { sed -n "s/^$1 = \"\{0,1\}\([^\"]*\)\"\{0,1\}$/\1/p" "${lock}"; }

# The file each digest covers. Adding a digest to the lock without adding it here
# stops the run rather than reducing what is verified.
file_for() {
  case "$1" in
    rules_sha256)             echo "${dir}/entid-rules-${version}.binpb" ;;
    conformance_sha256)       echo "${dir}/entid-conformance-${version}.binpb" ;;
    conformance_jsonl_sha256) echo "${dir}/entid-conformance-${version}.jsonl.gz" ;;
    rules_proto_sha256)       echo "${dir}/rules.proto" ;;
    conformance_proto_sha256) echo "${dir}/conformance.proto" ;;
    testee_proto_sha256)      echo "${dir}/testee.proto" ;;
    ir_doc_sha256)            echo "${dir}/ir.md" ;;
    features_doc_sha256)      echo "${dir}/features.md" ;;
    *)                        return 1 ;;
  esac
}

if [[ "$(value rules_version)" != "${version}" ]]; then
  echo "::error::rules.lock declares $(value rules_version), expected ${version}"
  exit 1
fi

declared="$(sed -n 's/^\([a-z0-9_]*sha256\) = .*/\1/p' "${lock}")"
if [[ -z "${declared}" ]]; then
  echo "::error::rules.lock declares no digest at all; either it is empty or its shape changed"
  exit 1
fi

# Verifying every digest the lock declares is not enough on its own: a lock that
# declares seven where the contract fixes eight would pass, verifying seven. That
# is the original defect moved one step. So the count comes from the normative
# list in engine.md section 16, which the release publishes beside the artifacts,
# and never from a number written here.
contract="${dir}/engine.md"
if [[ ! -f "${contract}" ]]; then
  echo "::error::${contract} is missing; the normative field list cannot be read and the lock cannot be judged complete"
  exit 1
fi
required="$(sed -n '/^```lock-fields$/,/^```$/p' "${contract}" | sed -n 's/^\([a-z0-9_]*sha256\)$/\1/p')"
if [[ -z "${required}" ]]; then
  echo "::error::${contract} carries no lock-fields block; the normative list moved or was deleted"
  exit 1
fi
missing=""
while read -r field; do
  [[ -z "${field}" ]] && continue
  grep -q "^${field} = " "${lock}" || missing="${missing} ${field}"
done <<< "${required}"
if [[ -n "${missing}" ]]; then
  echo "::error::rules.lock is missing digests the contract requires:${missing}"
  exit 1
fi

checked=0
while read -r field; do
  [[ -z "${field}" ]] && continue
  if ! file="$(file_for "${field}")"; then
    echo "::error::rules.lock declares ${field} and this script does not know which file it covers"
    exit 1
  fi
  want="$(value "${field}")"
  if [[ -z "${want}" ]]; then
    echo "::error::${field} is declared with no value"
    exit 1
  fi
  actual="$(digest_of "${file}")"
  if [[ "${want}" != "${actual}" ]]; then
    echo "::error::${field} mismatch for ${file}: lock=${want} actual=${actual}"
    exit 1
  fi
  echo "ok ${field} ${file}"
  checked=$((checked + 1))
done <<< "${declared}"

# The count is stated so that a lock which quietly loses a digest reads as a
# shorter run rather than as the same success.
echo "ok ${checked} digests verified"

identity="$(value attestation_identity)"
if [[ -z "${identity}" ]]; then
  echo "::error::rules.lock carries no attestation identity"
  exit 1
fi
echo "ok attestation identity ${identity}"

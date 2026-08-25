#!/usr/bin/env bash
# Verify that a generated rules.lock matches the downloaded artifacts.
#
# Usage: check_lock.sh <rules.lock> <artifact-dir> <rules-version>
set -euo pipefail

lock="$1"
dir="$2"
version="$3"

value() { sed -n "s/^$1 = \"\{0,1\}\([^\"]*\)\"\{0,1\}$/\1/p" "${lock}"; }

expect_sha() {
  local field="$1" file="$2"
  local want actual
  want="$(value "${field}")"
  actual="$(sha256sum "${file}" | cut -d' ' -f1)"
  if [[ "${want}" != "${actual}" ]]; then
    echo "::error::${field} mismatch for ${file}: lock=${want} actual=${actual}"
    exit 1
  fi
  echo "ok ${field} ${file}"
}

if [[ "$(value rules_version)" != "${version}" ]]; then
  echo "::error::rules.lock declares $(value rules_version), expected ${version}"
  exit 1
fi

expect_sha rules_sha256 "${dir}/entid-rules-${version}.binpb"
expect_sha conformance_sha256 "${dir}/entid-conformance-${version}.binpb"
expect_sha rules_proto_sha256 "${dir}/rules.proto"
expect_sha conformance_proto_sha256 "${dir}/conformance.proto"
expect_sha testee_proto_sha256 "${dir}/testee.proto"
expect_sha ir_doc_sha256 "${dir}/ir.md"
expect_sha features_doc_sha256 "${dir}/features.md"

identity="$(value attestation_identity)"
if [[ -z "${identity}" ]]; then
  echo "::error::rules.lock carries no attestation identity"
  exit 1
fi
echo "ok attestation identity ${identity}"

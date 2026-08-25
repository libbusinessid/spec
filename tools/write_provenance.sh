#!/usr/bin/env bash
# Copy one engine's spec/PROVENANCE.md out of a built dist.
#
# It used to assemble the file, reading five figures out of the rendered
# coverage.md with sed patterns matching prose nothing forced to stay stable.
# The compiler assembles it now, so the note travels with the release, is
# attested with everything else, and an engine that has verified a release no
# longer has to clone this repository to write the last file of its sync. The
# Swift engine measured that no published release could be synchronized because
# the writer postdated both tags.
#
# Usage: write_provenance.sh <spec-root> <dist-dir> <version> <commit> <engine> <output>
# The spec root, version and commit are accepted and ignored: the caller's
# signature did not change, and dist already holds the assembled note.
set -euo pipefail
dist="$2"
engine="$5"
output="$6"
cp "${dist}/provenance-${engine#businessid-}.md" "${output}"

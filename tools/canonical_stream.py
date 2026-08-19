#!/usr/bin/env python3
"""Independent reference implementation of the LibBusinessID canonical source
stream, used to cross-check the Go implementation without relying on it.

Usage:
    canonical_stream.py rules rules/=rules
    canonical_stream.py conformance conformance/=conformance fixtures/=testdata

For the conformance domain, only the fixtures actually referenced by a
`load_ruleset` case are incorporated, exactly like the compiler. Files are
normalized as specified in spec.md section 7.2, refined by decision ND-001 of
docs/normative-decisions.md for binary fixtures.
"""

import hashlib
import os
import struct
import sys


def normalize(content: bytes, binary: bool) -> bytes:
    """Normalize a text entry; a binary entry is incorporated verbatim.

    See docs/normative-decisions.md, decision ND-001.
    """
    if binary:
        return content
    content.decode("utf-8")
    if content[:3] == b"\xef\xbb\xbf":
        raise ValueError("byte order mark")
    return content.replace(b"\r\n", b"\n").replace(b"\r", b"\n")


def collect(prefix: str, directory: str, extensions: tuple[str, ...]) -> list[tuple[str, bytes]]:
    out = []
    for base, dirs, files in os.walk(directory):
        dirs[:] = sorted(d for d in dirs if not d.startswith(".") and d not in ("dist", "vendor"))
        for name in sorted(files):
            if not name.endswith(extensions):
                continue
            path = os.path.join(base, name)
            rel = os.path.relpath(path, directory).replace(os.sep, "/")
            binary = not name.endswith((".hcl", ".jsonl"))
            with open(path, "rb") as handle:
                out.append((prefix + rel, normalize(handle.read(), binary)))
    return out


def digest(domain: str, entries: list[tuple[str, bytes]]) -> str:
    h = hashlib.sha256()
    h.update(domain.encode("ascii"))
    for path, content in sorted(entries, key=lambda e: e[0].encode("utf-8")):
        encoded = path.encode("utf-8")
        h.update(struct.pack(">Q", len(encoded)))
        h.update(encoded)
        h.update(struct.pack(">Q", len(content)))
        h.update(content)
    return h.hexdigest()


def referenced_fixtures(entries: list[tuple[str, bytes]]) -> set[str]:
    """Return the fixture paths referenced by a load_ruleset case."""
    import json

    out: set[str] = set()
    for path, content in entries:
        if not path.endswith(".jsonl"):
            continue
        for line in content.decode("utf-8").splitlines():
            if not line.strip():
                continue
            case = json.loads(line)
            if case.get("fixture"):
                out.add(case["fixture"])
    return out


def main() -> int:
    if len(sys.argv) < 3:
        print(__doc__, file=sys.stderr)
        return 2
    domain = {"rules": "LIBBUSINESSID-SOURCE-V1\n",
              "conformance": "LIBBUSINESSID-CONFORMANCE-SOURCE-V1\n"}[sys.argv[1]]
    entries: list[tuple[str, bytes]] = []
    fixture_roots: list[tuple[str, str]] = []
    for spec in sys.argv[2:]:
        prefix, directory = spec.split("=", 1)
        if prefix == "fixtures/":
            fixture_roots.append((prefix, directory))
            continue
        extensions = (".hcl",) if sys.argv[1] == "rules" else (".jsonl",)
        entries.extend(collect(prefix, directory, extensions))
    wanted = referenced_fixtures(entries)
    for prefix, directory in fixture_roots:
        for rel in sorted(wanted):
            path = os.path.join(directory, rel)
            with open(path, "rb") as handle:
                entries.append((prefix + rel, handle.read()))
    print(digest(domain, entries))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

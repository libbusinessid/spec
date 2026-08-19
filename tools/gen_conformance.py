#!/usr/bin/env python3
"""Author the reviewed JSONL conformance corpus.

Every expectation written here comes from the published algorithm descriptions
re-implemented independently in tools/vectors.py, and from the normative
pipeline of docs/spec/spec.md. The Go compiler and the Go reference interpreter
are never consulted: the corpus is the oracle, not their output.
"""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from vectors import be_check, el_check, fr_vat_key, lei_check_digits, luhn_check_digit  # noqa: E402

KEY_ORDER = [
    "id", "description", "kind", "countryCode", "input", "profile", "operation",
    "expected", "tags", "sourceIds", "fixture", "expectedEngineError",
    "generated", "dataClassification", "redistributionBasis",
]
EXPECTED_ORDER = [
    "kind", "canonicalValue", "countryCode", "status", "reasonCode",
    "messageKey", "format", "checksum",
]
STEP_ORDER = ["status", "reasonCode", "messageKey"]

SYNTHETIC = "synthetic"
BASIS = ("Synthetic value produced by the documented generator of DATA_POLICY.md section 4; "
         "not derived from any register, extract, user submission or telemetry.")


def order(obj, keys):
    return {k: obj[k] for k in keys if k in obj}


def line(case):
    if "expected" in case:
        e = order(case["expected"], EXPECTED_ORDER)
        for step in ("format", "checksum"):
            if step in e:
                e[step] = order(e[step], STEP_ORDER)
        case["expected"] = e
    return json.dumps(order(case, KEY_ORDER), ensure_ascii=False, separators=(",", ":"))


def validate_case(kid, desc, kind, country, value, canonical, out_country,
                  fmt, checksum, tags, sources=(), profile="compatible",
                  operation="validate", expected_kind=None, classification=SYNTHETIC,
                  basis=BASIS):
    expected = {"canonicalValue": canonical, "format": fmt, "checksum": checksum}
    if expected_kind is not None:
        expected["kind"] = expected_kind
    if out_country is not None:
        expected["countryCode"] = out_country
    case = {
        "id": kid, "description": desc, "kind": kind, "input": value,
        "profile": profile, "operation": operation, "expected": expected,
        "tags": sorted(set(tags)), "dataClassification": classification,
        "redistributionBasis": basis,
    }
    if country is not None:
        case["countryCode"] = country
    if sources:
        case["sourceIds"] = sorted(set(sources))
    return case


def canon_case(kid, desc, kind, country, value, canonical, out_country, status, reason,
               tags, sources=(), profile="compatible", expected_kind=None, message_key=None,
               classification=SYNTHETIC, basis=BASIS):
    expected = {"canonicalValue": canonical, "status": status, "reasonCode": reason}
    if expected_kind is not None:
        expected["kind"] = expected_kind
    if out_country is not None:
        expected["countryCode"] = out_country
    if message_key is not None:
        expected["messageKey"] = message_key
    case = {
        "id": kid, "description": desc, "kind": kind, "input": value,
        "profile": profile, "operation": "canonicalize", "expected": expected,
        "tags": sorted(set(tags)), "dataClassification": classification,
        "redistributionBasis": basis,
    }
    if country is not None:
        case["countryCode"] = country
    if sources:
        case["sourceIds"] = sorted(set(sources))
    return case


VALID = {"status": "valid", "reasonCode": "ok"}


def step(status, reason, key=None):
    out = {"status": status, "reasonCode": reason}
    if key is not None:
        out["messageKey"] = key
    return out


def not_run(reason):
    return step("not_run", reason)


INVALID_FMT_NOT_RUN = not_run("not_run_format_invalid")
UNSUP_FMT_NOT_RUN = not_run("not_run_format_unsupported")


def write(path, cases):
    cases = sorted(cases, key=lambda c: c["id"])
    ids = [c["id"] for c in cases]
    assert len(ids) == len(set(ids)), "duplicate case id in " + path
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        for case in cases:
            handle.write(line(case) + "\n")
    print(f"{path}: {len(cases)} cases")

#!/usr/bin/env python3
"""SIREN conformance corpus."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gen_conformance import (INVALID_FMT_NOT_RUN, VALID, canon_case, step, validate_case, write)  # noqa: E402
from vectors import luhn_check_digit, luhn_ok  # noqa: E402

SRC = ["fr-insee-siren"]
V1 = "01234567" + luhn_check_digit("01234567")   # 012345674
V2 = "12345678" + luhn_check_digit("12345678")   # 123456782
V3 = "00000000" + luhn_check_digit("00000000")   # 000000000
assert luhn_ok(V1) and luhn_ok(V2) and luhn_ok(V3)
MUT1 = V1[:8] + str((int(V1[8]) + 1) % 10)
assert not luhn_ok(MUT1)

cases = [
    validate_case("siren-valid-001", "Synthetic SIREN with a correct Luhn check digit",
                  "siren", None, V1, V1, "FR", VALID, VALID,
                  ["luhn", "synthetic", "valid"], SRC),
    validate_case("siren-valid-002", "Second independent synthetic SIREN",
                  "siren", None, V2, V2, "FR", VALID, VALID,
                  ["luhn", "synthetic", "valid"], SRC),
    validate_case("siren-valid-003", "Lower bound of the SIREN range, all zero digits",
                  "siren", None, V3, V3, "FR", VALID, VALID,
                  ["boundary", "luhn", "synthetic", "valid"], SRC),
    validate_case("siren-valid-country-004", "Explicit FR country context selects the same target",
                  "siren", "FR", V1, V1, "FR", VALID, VALID,
                  ["country", "synthetic", "valid"], SRC),
    validate_case("siren-normalization-spaces-005", "Spaces are removed by canonicalization",
                  "siren", None, "012 345 674", V1, "FR", VALID, VALID,
                  ["normalization", "synthetic", "valid"], SRC),
    validate_case("siren-normalization-dots-006", "Dots and dashes are removed by canonicalization",
                  "siren", None, "012.345-674", V1, "FR", VALID, VALID,
                  ["normalization", "synthetic", "valid"], SRC),
    validate_case("siren-alias-kind-007", "The fr_siren alias resolves to the canonical siren kind",
                  "siren", None, V1, V1, "FR", VALID, VALID,
                  ["alias", "synthetic", "valid"], SRC),
    validate_case("siren-empty-010", "An empty value is proven invalid by the first assertion",
                  "siren", None, "", "", "FR",
                  step("invalid", "empty", "fr.siren.empty"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], SRC),
    validate_case("siren-too-short-011", "Eight digits are too short",
                  "siren", None, V1[:8], V1[:8], "FR",
                  step("invalid", "invalid_length", "fr.siren.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], SRC),
    validate_case("siren-too-long-012", "Ten digits are too long",
                  "siren", None, V1 + "0", V1 + "0", "FR",
                  step("invalid", "invalid_length", "fr.siren.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], SRC),
    validate_case("siren-letter-first-013", "A letter in the first position is refused",
                  "siren", None, "A1234567" + V1[8], "A1234567" + V1[8], "FR",
                  step("invalid", "invalid_characters", "fr.siren.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], SRC),
    validate_case("siren-letter-middle-014", "A letter in a middle position is refused",
                  "siren", None, V1[:4] + "A" + V1[5:], V1[:4] + "A" + V1[5:], "FR",
                  step("invalid", "invalid_characters", "fr.siren.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], SRC),
    validate_case("siren-letter-last-015", "A letter in the check digit position is refused",
                  "siren", None, V1[:8] + "A", V1[:8] + "A", "FR",
                  step("invalid", "invalid_characters", "fr.siren.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], SRC),
]

# One mutation per check digit value: exactly one of the ten values is correct.
for digit in range(10):
    value = V1[:8] + str(digit)
    if luhn_ok(value):
        continue
    cases.append(validate_case(
        f"siren-checksum-mutation-{digit}-02{digit}",
        f"Mutating the SIREN check digit to {digit} breaks the Luhn check",
        "siren", None, value, value, "FR", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], SRC))

# One mutation per position of the body, keeping the published check digit.
for position in range(8):
    body = list(V1)
    body[position] = str((int(body[position]) + 1) % 10)
    value = "".join(body)
    if luhn_ok(value):
        continue
    cases.append(validate_case(
        f"siren-body-mutation-{position}-03{position}",
        f"Mutating the SIREN digit at position {position} breaks the Luhn check",
        "siren", None, value, value, "FR", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], SRC))

cases += [
    validate_case("siren-unsupported-country-040", "A country without target is unsupported, never invalid",
                  "siren", "DE", V1, V1, "DE",
                  step("unsupported", "unsupported_country"), step("not_run", "not_run_format_unsupported"),
                  ["country", "synthetic", "unsupported"], SRC),
    validate_case("siren-strict-profile-041", "The strict_current profile does not change the SIREN rules",
                  "siren", None, V1, V1, "FR", VALID, VALID,
                  ["profile", "synthetic", "valid"], SRC, profile="strict_current"),
    validate_case("siren-validate-format-050", "validate_format reports checksum not_run/not_requested",
                  "siren", None, V1, V1, "FR", VALID, step("not_run", "not_requested"),
                  ["format", "synthetic", "valid"], SRC, operation="validate_format"),
    validate_case("siren-validate-format-invalid-051", "validate_format keeps the format failure",
                  "siren", None, V1[:8], V1[:8], "FR",
                  step("invalid", "invalid_length", "fr.siren.length"), INVALID_FMT_NOT_RUN,
                  ["format", "invalid", "synthetic"], SRC, operation="validate_format"),
    validate_case("siren-validate-checksum-052", "validate_checksum returns the same report as validate",
                  "siren", None, MUT1, MUT1, "FR", VALID, step("invalid", "invalid_checksum"),
                  ["checksum", "invalid", "synthetic"], SRC, operation="validate_checksum"),
    canon_case("siren-canonicalize-060", "Canonicalization removes the separators",
               "siren", None, "012 345-674", V1, "FR", "valid", "ok",
               ["canonicalize", "normalization", "synthetic"], SRC),
    canon_case("siren-canonicalize-alias-061", "The fr_siren alias resolves to the canonical kind",
               "siren", None, V1, V1, "FR", "valid", "ok",
               ["alias", "canonicalize", "synthetic"], SRC),
    canon_case("siren-canonicalize-unsupported-country-062", "An unsupported country keeps the pre-canonical value",
               "siren", "DE", "012 345 674", V1, "DE", "unsupported", "unsupported_country",
               ["canonicalize", "country", "synthetic", "unsupported"], SRC),
]

# The kind alias cases must declare the resolved kind explicitly.
alias = validate_case("siren-alias-kind-007", "The fr_siren alias resolves to the canonical siren kind",
                      "fr_siren", None, V1, V1, "FR", VALID, VALID,
                      ["alias", "synthetic", "valid"], SRC, expected_kind="siren")
cases = [c for c in cases if c["id"] != "siren-alias-kind-007"] + [alias]
canon_alias = canon_case("siren-canonicalize-alias-061", "The fr_siren alias resolves to the canonical kind",
                         "fr_siren", None, V1, V1, "FR", "valid", "ok",
                         ["alias", "canonicalize", "synthetic"], SRC, expected_kind="siren")
cases = [c for c in cases if c["id"] != "siren-canonicalize-alias-061"] + [canon_alias]

write(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "conformance", "national", "fr_siren.jsonl"), cases)

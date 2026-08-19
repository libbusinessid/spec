#!/usr/bin/env python3
"""EUID, LEI and dispatch conformance corpora."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gen_conformance import (INVALID_FMT_NOT_RUN, UNSUP_FMT_NOT_RUN, VALID, canon_case, step,
                             validate_case, write)  # noqa: E402
from vectors import lei_check_digits, lei_ok, luhn_check_digit, luhn_ok  # noqa: E402

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
EUID_SRC = ["eu-2015-884-euid", "fr-insee-siren"]
LEI_SRC = ["gleif-lei-structure"]
BE_SRC = ["be-fps-finance-vat", "eu-vies-number-structure"]
SIREN_SRC = ["fr-insee-siren"]

SIREN = "01234567" + luhn_check_digit("01234567")
BADSIREN = SIREN[:8] + str((int(SIREN[8]) + 1) % 10)
EUID = "FRTVX." + SIREN

euid = [
    validate_case("euid-fr-valid-001", "Synthetic French EUID whose registration number is a SIREN",
                  "euid", None, EUID, EUID, "FR", VALID, VALID,
                  ["composition", "synthetic", "valid"], EUID_SRC),
    validate_case("euid-fr-valid-002", "A second register identifier is accepted",
                  "euid", None, "FRRCS." + SIREN, "FRRCS." + SIREN, "FR", VALID, VALID,
                  ["composition", "synthetic", "valid"], EUID_SRC),
    validate_case("euid-fr-normalization-003", "Spaces are removed and the value is upper cased",
                  "euid", None, "fr tvx." + SIREN, EUID, "FR", VALID, VALID,
                  ["normalization", "synthetic", "valid"], EUID_SRC),
    validate_case("euid-fr-country-context-004", "The country context adds the FR prefix",
                  "euid", "FR", "TVX." + SIREN, EUID, "FR", VALID, VALID,
                  ["country", "synthetic", "valid"], EUID_SRC),
    validate_case("euid-fr-empty-010", "An empty value with a French context has no separator",
                  "euid", "FR", "", "FR", "FR",
                  step("invalid", "invalid_format", "euid.fr.separator"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], EUID_SRC),
    validate_case("euid-fr-missing-separator-011", "A value without a dot is refused",
                  "euid", None, "FRTVX" + SIREN, "FRTVX" + SIREN, "FR",
                  step("invalid", "invalid_format", "euid.fr.separator"), INVALID_FMT_NOT_RUN,
                  ["invalid", "structure", "synthetic"], EUID_SRC),
    validate_case("euid-fr-empty-register-012", "An empty register identifier is refused",
                  "euid", None, "FR." + SIREN, "FR." + SIREN, "FR",
                  step("invalid", "invalid_length", "euid.fr.register_length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], EUID_SRC),
    validate_case("euid-fr-register-too-long-013", "A register identifier of nine characters is refused",
                  "euid", None, "FRABCDEFGHI." + SIREN, "FRABCDEFGHI." + SIREN, "FR",
                  step("invalid", "invalid_length", "euid.fr.register_length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], EUID_SRC),
    validate_case("euid-fr-register-symbol-014", "A non alphanumeric register identifier is refused",
                  "euid", None, "FRT*X." + SIREN, "FRT*X." + SIREN, "FR",
                  step("invalid", "invalid_characters", "euid.fr.register_characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], EUID_SRC),
    validate_case("euid-fr-strict-profile-022", "The strict_current profile does not change the EUID rules",
                  "euid", None, EUID, EUID, "FR", VALID, VALID,
                  ["profile", "strict", "synthetic", "valid"], EUID_SRC, profile="strict_current"),
    validate_case("euid-fr-registration-length-015",
                  "The reused SIREN format reports its own length reason code",
                  "euid", None, "FRTVX." + SIREN[:8], "FRTVX." + SIREN[:8], "FR",
                  step("invalid", "invalid_length", "fr.siren.length"), INVALID_FMT_NOT_RUN,
                  ["composition", "invalid", "length", "synthetic"], EUID_SRC),
    validate_case("euid-fr-registration-characters-016",
                  "The reused SIREN format reports its own character reason code",
                  "euid", None, "FRTVX." + SIREN[:8] + "A", "FRTVX." + SIREN[:8] + "A", "FR",
                  step("invalid", "invalid_characters", "fr.siren.characters"), INVALID_FMT_NOT_RUN,
                  ["composition", "invalid", "synthetic"], EUID_SRC),
    validate_case("euid-fr-registration-checksum-017",
                  "The reused SIREN checksum rejects a broken registration number",
                  "euid", None, "FRTVX." + BADSIREN, "FRTVX." + BADSIREN, "FR", VALID,
                  step("invalid", "invalid_checksum"),
                  ["checksum", "composition", "invalid", "synthetic"], EUID_SRC),
    validate_case("euid-fr-unsupported-country-018", "A country without target is unsupported",
                  "euid", "DE", EUID, EUID, "DE",
                  step("unsupported", "unsupported_country"), UNSUP_FMT_NOT_RUN,
                  ["country", "synthetic", "unsupported"], EUID_SRC),
    validate_case("euid-fr-missing-country-019", "Without prefix and without country no target is selectable",
                  "euid", None, "TVX." + SIREN, "TVX." + SIREN, None,
                  step("unsupported", "missing_country_code"), UNSUP_FMT_NOT_RUN,
                  ["country", "synthetic", "unsupported"], EUID_SRC),
    canon_case("euid-fr-canonicalize-020", "Canonicalization keeps the dot separator",
               "euid", None, "fr tvx." + SIREN, EUID, "FR", "valid", "ok",
               ["canonicalize", "synthetic"], EUID_SRC),
    validate_case("euid-fr-validate-format-021", "validate_format stops before the reused checksum",
                  "euid", None, "FRTVX." + BADSIREN, "FRTVX." + BADSIREN, "FR", VALID,
                  step("not_run", "not_requested"),
                  ["composition", "format", "synthetic"], EUID_SRC, operation="validate_format"),
]
for digit in range(10):
    value = SIREN[:8] + str(digit)
    if luhn_ok(value):
        continue
    euid.append(validate_case(
        f"euid-fr-registration-mutation-{digit}-03{digit}",
        f"Mutating the EUID registration check digit to {digit} breaks the reused SIREN checksum",
        "euid", None, "FRTVX." + value, "FRTVX." + value, "FR", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "composition", "invalid", "mutation", "synthetic"], EUID_SRC))

write(os.path.join(ROOT, "conformance", "euid", "fr.jsonl"), euid)

LEI1 = "000000000000000000" + lei_check_digits("000000000000000000")
LEI2 = "000000ABCDEF123456" + lei_check_digits("000000ABCDEF123456")
assert lei_ok(LEI1) and lei_ok(LEI2)
lei = [
    validate_case("lei-valid-001", "Synthetic LEI built on an unassigned LOU prefix",
                  "lei", None, LEI1, LEI1, None, VALID, VALID,
                  ["iso7064", "synthetic", "valid"], LEI_SRC),
    validate_case("lei-valid-002", "Synthetic alphanumeric LEI",
                  "lei", None, LEI2, LEI2, None, VALID, VALID,
                  ["iso7064", "synthetic", "valid"], LEI_SRC),
    validate_case("lei-normalization-003", "Separators and lower case are removed by canonicalization",
                  "lei", None, "0000-00ab-cdef-1234-56" + lei_check_digits("000000ABCDEF123456"),
                  LEI2, None, VALID, VALID,
                  ["normalization", "synthetic", "valid"], LEI_SRC),
    validate_case("lei-global-keeps-country-004",
                  "A GLOBAL target keeps a well formed country context without using it",
                  "lei", "fr", LEI1, LEI1, "FR", VALID, VALID,
                  ["country", "global", "synthetic", "valid"], LEI_SRC),
    validate_case("lei-invalid-country-token-005", "A syntactically invalid country token is unsupported",
                  "lei", "FRA", LEI1, LEI1, "FRA",
                  step("unsupported", "unsupported_country"), UNSUP_FMT_NOT_RUN,
                  ["country", "global", "synthetic", "unsupported"], LEI_SRC),
    validate_case("lei-empty-010", "An empty value is proven invalid by the first assertion",
                  "lei", None, "", "", None,
                  step("invalid", "empty", "lei.empty"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], LEI_SRC),
    validate_case("lei-too-short-011", "Nineteen characters are too short",
                  "lei", None, LEI1[:19], LEI1[:19], None,
                  step("invalid", "invalid_length", "lei.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], LEI_SRC),
    validate_case("lei-too-long-012", "Twenty one characters are too long",
                  "lei", None, LEI1 + "0", LEI1 + "0", None,
                  step("invalid", "invalid_length", "lei.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], LEI_SRC),
    validate_case("lei-lowercase-symbol-013", "A non alphanumeric character is refused",
                  "lei", None, LEI1[:19] + "*", LEI1[:19] + "*", None,
                  step("invalid", "invalid_characters", "lei.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], LEI_SRC),
    validate_case("lei-letter-check-digit-014", "A letter in the check digit positions is refused",
                  "lei", None, LEI1[:18] + "9A", LEI1[:18] + "9A", None,
                  step("invalid", "invalid_characters", "lei.check_digits"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], LEI_SRC),
    validate_case("lei-strict-profile-015", "The strict_current profile does not change the LEI rules",
                  "lei", None, LEI1, LEI1, None, VALID, VALID,
                  ["profile", "strict", "synthetic", "valid"], LEI_SRC, profile="strict_current"),
    canon_case("lei-canonicalize-020", "Canonicalization removes the grouping dashes",
               "lei", None, "0000-0000-0000-0000-00" + LEI1[18:], LEI1, None, "valid", "ok",
               ["canonicalize", "synthetic"], LEI_SRC),
]
for digit in "0123456789":
    for position in (18, 19):
        value = LEI1[:position] + digit + LEI1[position + 1:]
        if lei_ok(value):
            continue
        lei.append(validate_case(
            f"lei-checksum-mutation-{position}-{digit}-03{position}{digit}",
            f"Mutating the LEI check digit at position {position} to {digit} breaks the ISO 7064 relation",
            "lei", None, value, value, None, VALID, step("invalid", "invalid_checksum"),
            ["checksum", "invalid", "mutation", "synthetic"], LEI_SRC))
write(os.path.join(ROOT, "conformance", "global", "lei.jsonl"), lei)

BE = "BE01234567" + __import__("vectors").be_check("01234567")
dispatch = [
    validate_case("dispatch-unknown-kind-001", "An unknown kind never runs a program",
                  "unknown_kind", None, "X", "X", None,
                  step("unsupported", "unsupported_kind"), UNSUP_FMT_NOT_RUN,
                  ["dispatch", "synthetic", "unsupported"]),
    validate_case("dispatch-malformed-kind-002", "A malformed kind token is unsupported",
                  "VAT!", None, "X", "X", None,
                  step("unsupported", "unsupported_kind"), UNSUP_FMT_NOT_RUN,
                  ["dispatch", "synthetic", "unsupported"], expected_kind="vat!"),
    validate_case("dispatch-kind-trim-003", "A kind token is trimmed and lower cased before resolution",
                  "  VAT  ", None, BE, BE, "BE", VALID, VALID,
                  ["dispatch", "normalization", "synthetic", "valid"], BE_SRC, expected_kind="vat"),
    validate_case("dispatch-country-mismatch-004", "An explicit country contradicting the prefix is invalid",
                  "vat", "FR", BE, BE, "FR",
                  step("invalid", "country_mismatch"), INVALID_FMT_NOT_RUN,
                  ["dispatch", "invalid", "synthetic"], BE_SRC),
    validate_case("dispatch-missing-country-005", "Without prefix and without country no target is selectable",
                  "vat", None, "0123456749", "0123456749", None,
                  step("unsupported", "missing_country_code"), UNSUP_FMT_NOT_RUN,
                  ["dispatch", "synthetic", "unsupported"], BE_SRC),
    validate_case("dispatch-unsupported-country-006", "A country alias without target is unsupported",
                  "vat", "UK", "0123456749", "0123456749", "GB",
                  step("unsupported", "unsupported_country"), UNSUP_FMT_NOT_RUN,
                  ["alias", "dispatch", "synthetic", "unsupported"], BE_SRC),
    validate_case("dispatch-invalid-country-token-007", "A malformed country token is unsupported",
                  "vat", "belgium", "0123456749", "0123456749", "belgium",
                  step("unsupported", "unsupported_country"), UNSUP_FMT_NOT_RUN,
                  ["dispatch", "synthetic", "unsupported"], BE_SRC),
    validate_case("dispatch-empty-country-008", "An empty country token behaves like an absent context",
                  "vat", "", BE, BE, "BE", VALID, VALID,
                  ["dispatch", "synthetic", "valid"], BE_SRC),
    validate_case("dispatch-longest-prefix-009", "The EL prefix wins over a shorter competing prefix",
                  "vat", None, "EL012345670", "EL012345670", "GR", VALID, VALID,
                  ["dispatch", "prefix", "synthetic", "valid"], ["eu-vies-number-structure", "gr-aade-afm"]),
    validate_case("dispatch-implicit-target-010",
                  "A dispatcher with a single implicit target routes an unprefixed value",
                  "siren", None, "01234567" + luhn_check_digit("01234567"),
                  "01234567" + luhn_check_digit("01234567"), "FR", VALID, VALID,
                  ["dispatch", "implicit", "synthetic", "valid"], SIREN_SRC),
    validate_case("dispatch-input-too-long-011",
                  "An input above the safety limit is unsupported, never invalid",
                  "siren", "FR", "1" * 1025, "1" * 1025, "FR",
                  step("unsupported", "input_too_long"), UNSUP_FMT_NOT_RUN,
                  ["limits", "security", "synthetic", "unsupported"], SIREN_SRC),
    canon_case("dispatch-canonicalize-input-too-long-012",
               "Canonicalization of an over long input keeps the raw value and context",
               "siren", "FR", "1" * 1025, "1" * 1025, "FR", "unsupported", "input_too_long",
               ["canonicalize", "limits", "security", "synthetic"], SIREN_SRC),
    canon_case("dispatch-canonicalize-unknown-kind-013",
               "Canonicalization of an unknown kind keeps the raw value",
               "unknown_kind", None, " X ", " X ", None, "unsupported", "unsupported_kind",
               ["canonicalize", "dispatch", "synthetic"]),
    canon_case("dispatch-canonicalize-country-mismatch-014",
               "A contradicting country is invalid at canonicalization time",
               "vat", "FR", BE, BE, "FR", "invalid", "country_mismatch",
               ["canonicalize", "dispatch", "synthetic"], BE_SRC),
    validate_case("dispatch-prefix-must-start-the-value-016",
                  "A prefix appearing inside the value never routes",
                  "vat", None, "0BE0123456749", "0BE0123456749", None,
                  step("unsupported", "missing_country_code"), UNSUP_FMT_NOT_RUN,
                  ["dispatch", "prefix", "synthetic", "unsupported"], BE_SRC),
    validate_case("dispatch-boundary-input-limit-015",
                  "An input of exactly 1024 bytes is still processed",
                  "siren", "FR", "1" * 1024, "1" * 1024, "FR",
                  step("invalid", "invalid_length", "fr.siren.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "limits", "synthetic"], SIREN_SRC),
]
write(os.path.join(ROOT, "conformance", "global", "dispatch.jsonl"), dispatch)

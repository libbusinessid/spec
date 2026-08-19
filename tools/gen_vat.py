#!/usr/bin/env python3
"""VAT conformance corpus for BE, DE, FR and GR."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gen_conformance import (INVALID_FMT_NOT_RUN, UNSUP_FMT_NOT_RUN, VALID, canon_case, step,
                             validate_case, write)  # noqa: E402
from vectors import be_check, el_check, fr_vat_key, luhn_ok, luhn_check_digit  # noqa: E402

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
BE_SRC = ["be-fps-finance-vat", "eu-vies-number-structure"]
DE_SRC = ["de-bzst-ustid", "eu-vies-number-structure"]
FR_SRC = ["eu-vies-number-structure", "fr-dgfip-vat"]
GR_SRC = ["eu-vies-number-structure", "gr-aade-afm"]

# ---------------------------------------------------------------- Belgium
BE1 = "01234567" + be_check("01234567")   # 0123456749
BE2 = "10000000" + be_check("10000000")   # 1000000021
BE3 = "01111111" + be_check("01111111")
be = [
    validate_case("vat-be-valid-001", "Synthetic Belgian VAT number with a correct modulo 97 check",
                  "vat", None, "BE" + BE1, "BE" + BE1, "BE", VALID, VALID,
                  ["mod97", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-valid-002", "Second synthetic Belgian VAT number starting with 1",
                  "vat", None, "BE" + BE2, "BE" + BE2, "BE", VALID, VALID,
                  ["mod97", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-valid-003", "Third synthetic Belgian VAT number",
                  "vat", None, "BE" + BE3, "BE" + BE3, "BE", VALID, VALID,
                  ["mod97", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-normalization-004", "Spaces and dots are removed by canonicalization",
                  "vat", None, "BE 0123.456.749", "BE" + BE1, "BE", VALID, VALID,
                  ["normalization", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-lowercase-005", "A lower case prefix is upper cased by canonicalization",
                  "vat", None, "be0123456749", "BE" + BE1, "BE", VALID, VALID,
                  ["normalization", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-country-context-006", "The country context selects the target and adds the prefix",
                  "vat", "BE", BE1, "BE" + BE1, "BE", VALID, VALID,
                  ["country", "normalization", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-legacy-nine-digits-007", "A legacy nine digit number is completed by a leading zero",
                  "vat", None, "BE123456749", "BE" + BE1, "BE", VALID, VALID,
                  ["legacy", "normalization", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-alias-kind-008", "The vat_number alias resolves to the canonical vat kind",
                  "vat_number", None, "BE" + BE1, "BE" + BE1, "BE", VALID, VALID,
                  ["alias", "synthetic", "valid"], BE_SRC, expected_kind="vat"),
    validate_case("vat-be-empty-010", "An empty value with a Belgian context is too short",
                  "vat", "BE", "", "BE", "BE",
                  step("invalid", "invalid_length", "vat.be.length"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], BE_SRC),
    validate_case("vat-be-too-short-011", "Nine digits that cannot be padded stay too short",
                  "vat", None, "BE0123456", "BE0123456", "BE",
                  step("invalid", "invalid_length", "vat.be.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], BE_SRC),
    validate_case("vat-be-too-long-012", "Eleven digits are too long",
                  "vat", None, "BE" + BE1 + "0", "BE" + BE1 + "0", "BE",
                  step("invalid", "invalid_length", "vat.be.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], BE_SRC),
    validate_case("vat-be-letter-013", "A letter in the digit part is refused",
                  "vat", None, "BE012345674A", "BE012345674A", "BE",
                  step("invalid", "invalid_characters", "vat.be.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], BE_SRC),
    validate_case("vat-be-enterprise-prefix-014", "An enterprise number starting with 2 is refused",
                  "vat", None, "BE2123456749", "BE2123456749", "BE",
                  step("invalid", "invalid_format", "vat.be.enterprise_prefix"), INVALID_FMT_NOT_RUN,
                  ["invalid", "structure", "synthetic"], BE_SRC),
    validate_case("vat-be-boundary-lowest-015", "The lowest enterprise number in the accepted range",
                  "vat", None, "BE" + "00000000" + be_check("00000000"), "BE00000000" + be_check("00000000"), "BE",
                  VALID, VALID, ["boundary", "synthetic", "valid"], BE_SRC),
    validate_case("vat-be-boundary-highest-016", "The highest enterprise number in the accepted range",
                  "vat", None, "BE" + "19999999" + be_check("19999999"), "BE19999999" + be_check("19999999"), "BE",
                  VALID, VALID, ["boundary", "synthetic", "valid"], BE_SRC),
]
for offset in (1, 2, 50, 96):
    wrong = "%02d" % ((int(be_check("01234567")) + offset) % 97)
    if wrong == be_check("01234567"):
        continue
    be.append(validate_case(
        f"vat-be-checksum-mutation-{offset:02d}-02{offset:02d}",
        f"Mutating the Belgian check digits by {offset} breaks the modulo 97 check",
        "vat", None, "BE01234567" + wrong, "BE01234567" + wrong, "BE", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], BE_SRC))
for position in range(8):
    body = list("01234567")
    body[position] = str((int(body[position]) + 1) % 10)
    mutated = "".join(body)
    if mutated[0] not in "01":
        continue
    be.append(validate_case(
        f"vat-be-body-mutation-{position}-03{position}",
        f"Mutating the Belgian enterprise digit at position {position} breaks the check",
        "vat", None, "BE" + mutated + be_check("01234567"), "BE" + mutated + be_check("01234567"), "BE",
        VALID, step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], BE_SRC))
be += [
    canon_case("vat-be-canonicalize-040", "Canonicalization adds the BE prefix",
               "vat", "BE", "0123.456.749", "BE" + BE1, "BE", "valid", "ok",
               ["canonicalize", "normalization", "synthetic"], BE_SRC),
    canon_case("vat-be-canonicalize-legacy-041", "Canonicalization pads a legacy nine digit number",
               "vat", None, "BE 123 456 749", "BE" + BE1, "BE", "valid", "ok",
               ["canonicalize", "legacy", "synthetic"], BE_SRC),
    validate_case("vat-be-strict-profile-042", "The strict_current profile does not change the Belgian rules",
                  "vat", None, "BE" + BE1, "BE" + BE1, "BE", VALID, VALID,
                  ["profile", "synthetic", "valid"], BE_SRC, profile="strict_current"),
]
write(os.path.join(ROOT, "conformance", "vat", "be.jsonl"), be)

# ---------------------------------------------------------------- Germany
de = [
    validate_case("vat-de-format-valid-001", "A well formed German VAT number stays checksum unsupported",
                  "vat", None, "DE123456789", "DE123456789", "DE", VALID,
                  step("unsupported", "unsupported_checksum"),
                  ["synthetic", "unsupported", "valid"], DE_SRC),
    validate_case("vat-de-format-valid-002", "A second well formed German VAT number",
                  "vat", None, "DE 000 000 000", "DE000000000", "DE", VALID,
                  step("unsupported", "unsupported_checksum"),
                  ["normalization", "synthetic", "unsupported", "valid"], DE_SRC),
    validate_case("vat-de-country-context-003", "The country context adds the DE prefix",
                  "vat", "DE", "123456789", "DE123456789", "DE", VALID,
                  step("unsupported", "unsupported_checksum"),
                  ["country", "synthetic", "unsupported"], DE_SRC),
    validate_case("vat-de-empty-010", "An empty value with a German context is too short",
                  "vat", "DE", "", "DE", "DE",
                  step("invalid", "invalid_length", "vat.de.length"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], DE_SRC),
    validate_case("vat-de-too-short-011", "Eight digits are too short",
                  "vat", None, "DE12345678", "DE12345678", "DE",
                  step("invalid", "invalid_length", "vat.de.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], DE_SRC),
    validate_case("vat-de-too-long-012", "Ten digits are too long",
                  "vat", None, "DE1234567890", "DE1234567890", "DE",
                  step("invalid", "invalid_length", "vat.de.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], DE_SRC),
    validate_case("vat-de-letter-013", "A letter in the digit part is refused",
                  "vat", None, "DE12345678A", "DE12345678A", "DE",
                  step("invalid", "invalid_characters", "vat.de.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], DE_SRC),
    validate_case("vat-de-strict-profile-014", "The strict_current profile does not change the German rules",
                  "vat", None, "DE123456789", "DE123456789", "DE", VALID,
                  step("unsupported", "unsupported_checksum"),
                  ["profile", "strict", "synthetic", "unsupported"], DE_SRC, profile="strict_current"),
    canon_case("vat-de-canonicalize-020", "Canonicalization adds the DE prefix",
               "vat", "DE", "123 456 789", "DE123456789", "DE", "valid", "ok",
               ["canonicalize", "synthetic"], DE_SRC),
]
write(os.path.join(ROOT, "conformance", "vat", "de.jsonl"), de)

# ---------------------------------------------------------------- France
SIREN1 = "01234567" + luhn_check_digit("01234567")
SIREN2 = "12345678" + luhn_check_digit("12345678")
FR1 = "FR" + fr_vat_key(SIREN1) + SIREN1
FR2 = "FR" + fr_vat_key(SIREN2) + SIREN2
BADSIREN = SIREN1[:8] + str((int(SIREN1[8]) + 1) % 10)
assert not luhn_ok(BADSIREN)
fr = [
    validate_case("vat-fr-valid-001", "Synthetic French VAT number with a numeric computation key",
                  "vat", None, FR1, FR1, "FR", VALID, VALID,
                  ["mod97", "synthetic", "valid"], FR_SRC),
    validate_case("vat-fr-valid-002", "Second synthetic French VAT number",
                  "vat", None, FR2, FR2, "FR", VALID, VALID,
                  ["mod97", "synthetic", "valid"], FR_SRC),
    validate_case("vat-fr-normalization-003", "Spaces are removed by canonicalization",
                  "vat", None, "FR " + fr_vat_key(SIREN1) + " " + SIREN1, FR1, "FR", VALID, VALID,
                  ["normalization", "synthetic", "valid"], FR_SRC),
    validate_case("vat-fr-country-context-004", "The country context adds the FR prefix",
                  "vat", "FR", fr_vat_key(SIREN1) + SIREN1, FR1, "FR", VALID, VALID,
                  ["country", "synthetic", "valid"], FR_SRC),
    validate_case("vat-fr-alphanumeric-key-005",
                  "An alphanumeric key follows an unpublished scheme and is never invalid",
                  "vat", None, "FRK7" + SIREN1, "FRK7" + SIREN1, "FR", VALID,
                  step("unsupported", "checksum_not_published"),
                  ["legacy", "synthetic", "unsupported"], FR_SRC),
    validate_case("vat-fr-alphanumeric-key-strict-006",
                  "The strict_current profile only accepts the numeric computation key",
                  "vat", None, "FRK7" + SIREN1, "FRK7" + SIREN1, "FR",
                  step("invalid", "invalid_characters", "vat.fr.key_characters"), INVALID_FMT_NOT_RUN,
                  ["profile", "strict", "synthetic"], FR_SRC, profile="strict_current"),
    validate_case("vat-fr-numeric-key-strict-007", "A numeric key is accepted by the strict_current profile",
                  "vat", None, FR1, FR1, "FR", VALID, VALID,
                  ["profile", "strict", "synthetic", "valid"], FR_SRC, profile="strict_current"),
    validate_case("vat-fr-empty-010", "An empty value with a French context is too short",
                  "vat", "FR", "", "FR", "FR",
                  step("invalid", "invalid_length", "vat.fr.length"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], FR_SRC),
    validate_case("vat-fr-too-short-011", "Twelve characters are too short",
                  "vat", None, FR1[:12], FR1[:12], "FR",
                  step("invalid", "invalid_length", "vat.fr.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], FR_SRC),
    validate_case("vat-fr-too-long-012", "Fourteen characters are too long",
                  "vat", None, FR1 + "0", FR1 + "0", "FR",
                  step("invalid", "invalid_length", "vat.fr.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], FR_SRC),
    validate_case("vat-fr-siren-letter-013", "A letter inside the SIREN part is refused",
                  "vat", None, "FR" + fr_vat_key(SIREN1) + SIREN1[:8] + "A",
                  "FR" + fr_vat_key(SIREN1) + SIREN1[:8] + "A", "FR",
                  step("invalid", "invalid_characters", "vat.fr.siren_characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], FR_SRC),
    validate_case("vat-fr-key-symbol-014", "A non alphanumeric key is refused",
                  "vat", None, "FR*7" + SIREN1, "FR*7" + SIREN1, "FR",
                  step("invalid", "invalid_characters", "vat.fr.key_characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], FR_SRC),
    validate_case("vat-fr-alphanumeric-key-broken-siren-016",
                  "A proven SIREN failure wins over an unknown key scheme",
                  "vat", None, "FRK7" + BADSIREN, "FRK7" + BADSIREN, "FR", VALID,
                  step("invalid", "invalid_checksum"),
                  ["checksum", "composition", "invalid", "synthetic"], FR_SRC),
    validate_case("vat-fr-embedded-siren-checksum-015",
                  "The reused SIREN checksum rejects a broken embedded SIREN",
                  "vat", None, "FR" + fr_vat_key(BADSIREN) + BADSIREN,
                  "FR" + fr_vat_key(BADSIREN) + BADSIREN, "FR", VALID,
                  step("invalid", "invalid_checksum"),
                  ["checksum", "composition", "invalid", "synthetic"], FR_SRC),
]
for offset in (1, 2, 40, 96):
    wrong = "%02d" % ((int(fr_vat_key(SIREN1)) + offset) % 97)
    if wrong == fr_vat_key(SIREN1):
        continue
    fr.append(validate_case(
        f"vat-fr-checksum-mutation-{offset:02d}-02{offset:02d}",
        f"Mutating the French key by {offset} breaks the modulo 97 relation",
        "vat", None, "FR" + wrong + SIREN1, "FR" + wrong + SIREN1, "FR", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], FR_SRC))
fr += [
    canon_case("vat-fr-canonicalize-030", "Canonicalization adds the FR prefix",
               "vat", "FR", fr_vat_key(SIREN1) + " " + SIREN1, FR1, "FR", "valid", "ok",
               ["canonicalize", "synthetic"], FR_SRC),
]
write(os.path.join(ROOT, "conformance", "vat", "fr.jsonl"), fr)

# ------------------------------------------------------------------ Greece
GR1 = "01234567" + el_check("01234567")
GR2 = "09999999" + el_check("09999999")
GR3 = "00000000" + el_check("00000000")
gr = [
    validate_case("vat-gr-valid-el-001", "Synthetic Greek VAT number written with the EL prefix",
                  "vat", None, "EL" + GR1, "EL" + GR1, "GR", VALID, VALID,
                  ["synthetic", "valid", "weighted"], GR_SRC),
    validate_case("vat-gr-valid-gr-002", "The same number written with the GR prefix canonicalizes to EL",
                  "vat", None, "GR " + GR1, "EL" + GR1, "GR", VALID, VALID,
                  ["normalization", "synthetic", "valid", "weighted"], GR_SRC),
    validate_case("vat-gr-valid-003", "Second synthetic Greek VAT number",
                  "vat", None, "EL" + GR2, "EL" + GR2, "GR", VALID, VALID,
                  ["synthetic", "valid", "weighted"], GR_SRC),
    validate_case("vat-gr-boundary-zero-004", "The all zero body maps the remainder 0 to the check digit 0",
                  "vat", None, "EL" + GR3, "EL" + GR3, "GR", VALID, VALID,
                  ["boundary", "synthetic", "valid", "weighted"], GR_SRC),
    validate_case("vat-gr-country-alias-005", "The EL country token is an alias of the ISO country GR",
                  "vat", "EL", GR1, "EL" + GR1, "GR", VALID, VALID,
                  ["alias", "country", "synthetic", "valid"], GR_SRC),
    validate_case("vat-gr-country-context-006", "The ISO country GR selects the same target",
                  "vat", "GR", GR1, "EL" + GR1, "GR", VALID, VALID,
                  ["country", "synthetic", "valid"], GR_SRC),
    validate_case("vat-gr-empty-010", "An empty value with a Greek context is too short",
                  "vat", "GR", "", "EL", "GR",
                  step("invalid", "invalid_length", "vat.gr.length"), INVALID_FMT_NOT_RUN,
                  ["empty", "invalid", "synthetic"], GR_SRC),
    validate_case("vat-gr-too-short-011", "Eight digits are too short",
                  "vat", None, "EL" + GR1[:8], "EL" + GR1[:8], "GR",
                  step("invalid", "invalid_length", "vat.gr.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], GR_SRC),
    validate_case("vat-gr-too-long-012", "Ten digits are too long",
                  "vat", None, "EL" + GR1 + "0", "EL" + GR1 + "0", "GR",
                  step("invalid", "invalid_length", "vat.gr.length"), INVALID_FMT_NOT_RUN,
                  ["boundary", "invalid", "length", "synthetic"], GR_SRC),
    validate_case("vat-gr-letter-013", "A letter in the digit part is refused",
                  "vat", None, "EL" + GR1[:8] + "A", "EL" + GR1[:8] + "A", "GR",
                  step("invalid", "invalid_characters", "vat.gr.characters"), INVALID_FMT_NOT_RUN,
                  ["characters", "invalid", "synthetic"], GR_SRC),
]
for digit in range(10):
    value = GR1[:8] + str(digit)
    if value == GR1:
        continue
    gr.append(validate_case(
        f"vat-gr-checksum-mutation-{digit}-02{digit}",
        f"Mutating the Greek check digit to {digit} breaks the weighted modulo 11 check",
        "vat", None, "EL" + value, "EL" + value, "GR", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], GR_SRC))
for position in range(8):
    body = list(GR1)
    body[position] = str((int(body[position]) + 1) % 10)
    value = "".join(body)
    if el_check(value[:8]) == value[8]:
        continue
    gr.append(validate_case(
        f"vat-gr-body-mutation-{position}-03{position}",
        f"Mutating the Greek digit at position {position} breaks the weighted check",
        "vat", None, "EL" + value, "EL" + value, "GR", VALID,
        step("invalid", "invalid_checksum"),
        ["checksum", "invalid", "mutation", "synthetic"], GR_SRC))
gr += [
    validate_case("vat-gr-strict-profile-039", "The strict_current profile does not change the Greek rules",
                  "vat", None, "EL" + GR1, "EL" + GR1, "GR", VALID, VALID,
                  ["profile", "strict", "synthetic", "valid"], GR_SRC, profile="strict_current"),
    canon_case("vat-gr-canonicalize-040", "Canonicalization rewrites the GR prefix into EL",
               "vat", None, "gr " + GR1, "EL" + GR1, "GR", "valid", "ok",
               ["canonicalize", "normalization", "synthetic"], GR_SRC),
]
write(os.path.join(ROOT, "conformance", "vat", "gr.jsonl"), gr)

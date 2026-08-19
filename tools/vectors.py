#!/usr/bin/env python3
"""Independent reference implementations of the published check algorithms used
by the LibBusinessID pilot rules.

This module is deliberately written from the published algorithm descriptions
and never consults the Go compiler or the Go reference interpreter. It is the
independent oracle used to author the conformance expectations.
"""


def luhn_ok(digits: str) -> bool:
    total = 0
    for i, c in enumerate(reversed(digits)):
        d = int(c)
        if i % 2 == 1:
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


def luhn_check_digit(body: str) -> str:
    for d in "0123456789":
        if luhn_ok(body + d):
            return d
    raise AssertionError


def be_check(first8: str) -> str:
    return "%02d" % (97 - int(first8) % 97)


def fr_vat_key(siren: str) -> str:
    return "%02d" % ((12 + 3 * (int(siren) % 97)) % 97)


def el_check(first8: str) -> str:
    weights = [256, 128, 64, 32, 16, 8, 4, 2]
    total = sum(int(c) * w for c, w in zip(first8, weights))
    return str((total % 11) % 10)


def lei_expand(s: str) -> str:
    out = ""
    for c in s:
        if c.isdigit():
            out += c
        else:
            out += str(ord(c) - 55)
    return out


def lei_ok(lei: str) -> bool:
    return int(lei_expand(lei)) % 97 == 1


def lei_check_digits(body18: str) -> str:
    return "%02d" % (98 - int(lei_expand(body18) + "00") % 97)


if __name__ == "__main__":
    print("SIREN 01234567_:", luhn_check_digit("01234567"))
    print("SIREN 12345678_:", luhn_check_digit("12345678"))
    print("SIREN 00000000_:", luhn_check_digit("00000000"))
    print("SIREN 000000004 luhn:", luhn_ok("000000004"))
    print("BE 01234567:", be_check("01234567"))
    print("BE 01111111:", be_check("01111111"))
    print("BE 10000000:", be_check("10000000"))
    for siren in ("012345670", "000000000", "123456782"):
        print("FR VAT key for", siren, ":", fr_vat_key(siren))
    print("FR remainder table:", [(12 + 3 * r) % 97 for r in range(97)])
    print("EL 01234567:", el_check("01234567"))
    print("EL 00000000:", el_check("00000000"))
    print("EL 09999999:", el_check("09999999"))
    print("LEI 000000000000000000 ->", lei_check_digits("000000000000000000"))
    print("LEI 000000ABCDEF123456 ->", lei_check_digits("000000ABCDEF123456"))
    print("LEI 5493001KJTIIGC8Y1R12 ok:", lei_ok("5493001KJTIIGC8Y1R12"))

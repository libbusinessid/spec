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

# National register check algorithms, written from the descriptions recorded in
# the `source` blocks of the rules. They never consult the Go compiler or the Go
# reference interpreter, which is what keeps the expected results independent.


def ico_check(first7: str) -> str:
    """Czech and Slovak ICO: weights 8..2, remainder modulo 11."""
    total = sum(int(c) * w for c, w in zip(first7, (8, 7, 6, 5, 4, 3, 2)))
    return str([1, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1][total % 11])


def dk_ok(cvr: str) -> bool:
    """Danish CVR: the weighted sum of all eight digits vanishes modulo 11."""
    total = sum(int(c) * w for c, w in zip(cvr, (2, 7, 6, 5, 4, 3, 2, 1)))
    return total % 11 == 0


def pt_check(first8: str) -> str:
    """Portuguese NIPC: weights 9..2, a remainder below two yielding zero."""
    total = sum(int(c) * w for c, w in zip(first8, (9, 8, 7, 6, 5, 4, 3, 2)))
    return str([0, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1][total % 11])


def ee_check(first7: str) -> str:
    """Estonian registrikood: weights 1..7, recomputed with 3..9 on a ten."""
    total = sum(int(c) * w for c, w in zip(first7, (1, 2, 3, 4, 5, 6, 7)))
    if total % 11 == 10:
        total = sum(int(c) * w for c, w in zip(first7, (3, 4, 5, 6, 7, 8, 9)))
    return str(total % 11 % 10)


def lt_ok(code: str) -> bool:
    """Lithuanian code: weights 1..9 vanishing, rotated to 3..9,1,2 on a ten."""
    total = sum(int(c) * w for c, w in zip(code, (1, 2, 3, 4, 5, 6, 7, 8, 9)))
    if total % 11 == 10:
        total = sum(int(c) * w for c, w in zip(code, (3, 4, 5, 6, 7, 8, 9, 1, 2)))
        return total % 11 % 10 == 0
    return total % 11 == 0


def ro_check(body: str) -> str:
    """Romanian CUI: weights aligned on the right of the digits before the check."""
    w = (7, 5, 3, 2, 1, 7, 5, 3, 2)[-len(body):]
    return str(sum(int(c) * x for c, x in zip(body, w)) * 10 % 11 % 10)


def bg_check(first8: str) -> str:
    """Bulgarian BULSTAT: weights 1..8, sent through 3..10 on a ten."""
    total = sum(int(c) * w for c, w in zip(first8, range(1, 9)))
    if total % 11 == 10:
        total = sum(int(c) * w for c, w in zip(first8, range(3, 11)))
    return str(total % 11 % 10)


def fi_check(first7: str):
    """Finnish Y-tunnus: a remainder of one is never issued."""
    total = sum(int(c) * w for c, w in zip(first7, (7, 9, 10, 5, 8, 4, 2)))
    m = total % 11
    if m == 1:
        return None
    return str(0 if m == 0 else 11 - m)


def lv_check(first10: str):
    """Latvian legal entity: three minus the sum, a ten never being issued."""
    total = sum(int(c) * w for c, w in zip(first10, (9, 1, 4, 8, 3, 10, 2, 5, 7, 6)))
    c = (3 - total) % 11
    return None if c == 10 else str(c)


def es_cif_digit(body7: str) -> str:
    """Spanish CIF: doubling the odd positions of the body, which is Luhn."""
    total = 0
    for i, c in enumerate(body7, start=1):
        d = int(c)
        if i % 2 == 1:
            d *= 2
            d = d // 10 + d % 10
        total += d
    return str((10 - total % 10) % 10)


def es_cif_letter(body7: str) -> str:
    return "JABCDEFGHI"[int(es_cif_digit(body7))]


# VAT check algorithms, from the descriptions recorded in the source blocks.


def _weighted(body, weights):
    return sum(int(c) * w for c, w in zip(body, weights))


def vat_fi_check(first7: str):
    m = _weighted(first7, (7, 9, 10, 5, 8, 4, 2)) % 11
    return None if m == 1 else str(0 if m == 0 else 11 - m)


def vat_si_check(first7: str):
    m = _weighted(first7, (8, 7, 6, 5, 4, 3, 2)) % 11
    c = 11 - m
    if c == 10:
        return None
    return str(0 if c == 11 else c)


def vat_pt_check(first8: str) -> str:
    m = _weighted(first8, (9, 8, 7, 6, 5, 4, 3, 2)) % 11
    return str(11 - m if m >= 2 else 0)


def vat_pl_check(first9: str):
    m = _weighted(first9, (6, 5, 7, 2, 3, 4, 5, 6, 7)) % 11
    return None if m == 10 else str(m)


def vat_ee_check(first8: str) -> str:
    return str((10 - _weighted(first8, (3, 7, 1, 3, 7, 1, 3, 7)) % 10) % 10)


def vat_mt_check(first6: str) -> str:
    return f"{37 - _weighted(first6, (3, 4, 6, 7, 8, 9)) % 37:02d}"


def vat_lu_check(first6: str) -> str:
    return f"{int(first6) % 89:02d}"


def vat_dk_ok(body8: str) -> bool:
    return _weighted(body8, (2, 7, 6, 5, 4, 3, 2, 1)) % 11 == 0


def vat_nl_check(first8: str):
    m = sum(int(c) * w for c, w in zip(first8, (9, 8, 7, 6, 5, 4, 3, 2))) % 11
    return None if m == 10 else str(m)


def vat_hu_check(first7: str) -> str:
    total = sum(int(c) * w for c, w in zip(first7, (9, 7, 3, 1, 9, 7, 3)))
    return str((10 - total % 10) % 10)


def vat_sk_ok(body10: str) -> bool:
    return int(body10) % 11 == 0


def vat_cz_check(first7: str) -> str:
    total = sum(int(c) * w for c, w in zip(first7, (8, 7, 6, 5, 4, 3, 2)))
    return str([1, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1][total % 11])


def vat_lv_check(first10: str):
    total = sum(int(c) * w for c, w in zip(first10, (9, 1, 4, 8, 3, 10, 2, 5, 7, 6)))
    c = (3 - total) % 11
    return None if c == 10 else str(c)


def vat_no_check(first8: str):
    m = sum(int(c) * w for c, w in zip(first8, (3, 2, 7, 6, 5, 4, 3, 2))) % 11
    c = 11 - m
    if c == 10:
        return None
    return str(0 if c == 11 else c)


def gb_ok(nine: str) -> bool:
    """UK VAT: the last two digits folded into the weighted sum, modulo 97."""
    ws = sum(int(c) * w for c, w in zip(nine, (8, 7, 6, 5, 4, 3, 2, 10, 1)))
    return ws % 97 in (0, 42)


def is_check(first8: str):
    """Icelandic ten digit form, the same shape as the Norwegian check."""
    m = sum(int(c) * w for c, w in zip(first8, (3, 2, 7, 6, 5, 4, 3, 2))) % 11
    c = 11 - m
    if c == 10:
        return None
    return str(0 if c == 11 else c)

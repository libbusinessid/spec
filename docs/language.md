# The LibBusinessID authoring language

The rules of LibBusinessID are written in HCL v2 native syntax. HCL provides the
syntax, the expression forms and the positional diagnostics; LibBusinessID
defines every piece of semantics.

The language is an **authoring** language. No production engine parses it: the
compiler `businessidc` resolves every symbolic reference and emits the typed IR
described in [`ir.md`](ir.md).

## 1. Compilation unit

There is **no import mechanism**. `businessidc` walks `rules/` recursively,
collects every `*.hcl` file, sorts them by POSIX relative path and analyses them
as a single compilation unit with a single global symbol table.

- Hidden directories, `dist` and `vendor` are skipped.
- Symbolic links are refused, at the root and at any depth.
- A rule can therefore never depend on the file system order, on the current
  directory or on a file outside `rules/`.

Sources are UTF-8 without byte order mark.

## 2. Symbols

Five symbol categories exist:

```text
canonicalizer.<namespace>.<name>
format.<namespace>.<name>
checksum.<namespace>.<name>
dispatcher.<kind>
identifier.<kind>.<country_or_GLOBAL>
```

A reusable declaration carries two labels, the namespace and the name:

```hcl
format "fr" "siren" {
  # ...
}
```

A reference is structural and typed:

```hcl
rule = format.fr.siren
```

Symbols are immutable. Duplicates and cycles are compilation errors. Text
interpolation such as `"FR.${rule.fr.siren.format}"` is refused: it can never be
used to compose a rule.

A canonical kind matches `[a-z][a-z0-9_]{0,63}` so that
`identifier.<kind>.<country>` stays writable; a kind alias may also hold a
hyphen (see decision ND-008 in [`normative-decisions.md`](normative-decisions.md)).
A country label is exactly `[A-Z]{2}`, or the literal `GLOBAL`.

## 3. Types

The typechecker knows:

| Type | Produced by |
|---|---|
| `StringExpr` | `value()`, `subject()`, `country_code()`, the view constructors, a string literal |
| `IntExpr` | the integer constructors of a checksum rule |
| `Predicate` | the predicates |
| `CanonicalizationStep` | the canonicalization steps |
| `Assertion` | `require(...)` and `use_format` |
| `ChecksumRule` | the checksum constructors |

There is no dynamic type: every expression is fully typed before emission.

## 4. Execution context

A rule reads only:

- `input.value`, through `value()` and `subject()`;
- `input.country_code`, through `country_code()`;
- `input.kind`, through the dispatcher that selected the definition;
- `options.profile`, through `profile_is(...)`;
- the value current during canonicalization;
- the captures produced by the format rule that declares them.

No system clock, environment variable, locale, network or randomness is
reachable.

## 5. Unicode and ASCII

Identifiers are evaluated by Unicode code points, but the V1 `digits`, `letters`
and `alphanumeric` classes are ASCII only. `uppercase_ascii()` maps only `a..z`.
The frozen `whitespace_v1` table is reproduced in [`ir.md`](ir.md) section 7 and
a runtime never delegates it to its own Unicode tables.

## 6. Canonicalizers

```hcl
canonicalizer "vat" "be" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
    when(
      all(length_eq(value(), 11), ascii_digits(slice_from(value(), 2))),
      insert(2, "0"),
    ),
  ]
}
```

The available steps are `trim_whitespace`, `remove_whitespace`,
`uppercase_ascii`, `remove_chars`, `replace_prefix`, `prepend`, `append`,
`insert`, `left_pad`, `prepend_country_if_missing` and `when`.

Inside a canonicalizer, `value()` is the value **current at that point**:
`when(...)` observes the result of the previous steps. `subject()` is not
available.

Canonicalization must be idempotent. `businessidc lint` checks that property on
every conformance case and on generated probes.

An impossible operation is a compile time error, never an `invalid` result. An
out of range `insert` and a `left_pad` shorter than the current value simply
leave the value unchanged; nothing is ever truncated.

## 7. String views and captures

| Constructor | Result |
|---|---|
| `value()` | the current canonical value |
| `subject()` | the subject of the enclosing program |
| `country_code()` | the canonical country of the selected target |
| `slice(expr, start, end)` | the code points in `[start, end)` |
| `slice_from(expr, start)` | from `start` to the end |
| `slice_to(expr, end)` | before `end` |
| `before_first(expr, delimiter)` | the part before the first delimiter |
| `after_first(expr, delimiter)` | the part after the first delimiter |
| `strip_prefix(expr, prefix)` | without the exact prefix, otherwise absent |
| `concat(expr...)` | the concatenation |
| `capture.<name>` | a named view of the enclosing format rule |

An operation that cannot succeed yields a typed **absent** value. A predicate
reading an absent value returns `false`, except `is_absent`.

The default subject of a format or checksum rule is `value()`. A declaration can
replace it with `subject = <StringExpr>`; the compiler resolves that attribute
before the rest of the block. When the rule is reused through `use_format` or
`apply_checksum`, `subject()` is the view supplied by the caller.

## 8. Predicates

```text
is_empty  is_absent  equals  length_eq  length_in  length_between
ascii_digits  ascii_upper_letters  ascii_alphanumeric  ascii_charset
starts_with  ends_with  prefix_in  char_at_in  contains
all  any  not  profile_is
```

There is no generic regular expression in V1. A new primitive is accepted only
when its semantics is identical and testable on the four engines.

## 9. Format rules

A format rule is an ordered sequence of assertions. The first failing assertion
determines the reason code and the message key.

```hcl
format "fr" "siren" {
  checks = [
    require(not(is_empty(subject())), "empty", "fr.siren.empty"),
    require(length_eq(subject(), 9), "invalid_length", "fr.siren.length"),
    require(ascii_digits(subject()), "invalid_characters", "fr.siren.characters"),
  ]
}
```

`require(predicate, reason_code, message_key)` is the only construct producing a
format invalidity. Its reason code is restricted to `empty`, `invalid_length`,
`invalid_characters`, `invalid_format` and `country_mismatch`.

A rule can declare captures and reuse another rule on a view:

```hcl
format "euid" "fr" {
  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(contains(subject(), "."), "invalid_format", "euid.fr.separator"),
  ]

  use_format {
    rule  = format.fr.siren
    input = capture.registration
  }
}
```

The checks of the parent run **before** the `use_format` blocks, in declaration
order. The compiler detects cycles and the reason codes of the called rule are
preserved unchanged.

## 10. Checksum rules

A checksum rule returns `valid`, `invalid` with `invalid_checksum`, or
`unsupported` with a stable reason.

```hcl
checksum "vat" "be" {
  rule = compare_slice(
    complement(mod_digits(slice(subject(), 2, 10), 97), 97),
    subject(), 10, 12,
  )
}
```

For a variant without a published algorithm:

```hcl
checksum "vat" "fr" {
  rule = all_checks(
    apply_checksum(checksum.fr.siren, slice_from(subject(), 4)),
    choose(
      when_checksum(ascii_digits(slice(subject(), 2, 4)), /* key check */),
      unsupported_checksum("checksum_not_published"),
    ),
  )
}
```

The available constructors are `luhn`, `iso7064_mod97_10`, `digits_to_integer`,
`mod_digits`, `weighted_sum`, `modulo`, `complement`, `remainder_map`,
`compare_digit`, `compare_slice`, `choose`, `when_checksum`, `all_checks`,
`any_check`, `unsupported_checksum` and `apply_checksum`.

`when_checksum` is only accepted as a direct branch of `choose`. `all_checks`
returns the first `invalid`, otherwise the first `unsupported`, otherwise
`valid`; `any_check` returns `valid` as soon as one branch is valid, otherwise
the first `unsupported`, otherwise the first `invalid`. Both orders keep a
proven failure stronger than an unknown algorithm, and an unknown algorithm
stronger than a silent acceptance.

`digits_to_integer` is only accepted when the compiler can prove the operand
holds at most 18 digits. For longer identifiers, only the digit by digit
`mod_digits` family is conforming.

## 11. Dispatchers and definitions

Selecting a definition is deliberately separated from canonicalizing it.

```hcl
dispatcher "vat" {
  aliases           = ["vat_id", "vat_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  country_aliases = {
    "EL" = "GR"
  }

  target {
    country_code      = "GR"
    accepted_prefixes = ["EL", "GR"]
    canonical_prefix  = "EL"
    identifier        = identifier.vat.GR
  }
}
```

A pre-canonicalizer is restricted to `trim_whitespace`, `remove_whitespace`,
`uppercase_ascii` and `remove_chars` with a constant list. It can never add,
replace or interpret a prefix.

Kind aliases, country aliases and value prefixes are three separate spaces.
Duplicates after normalization, contradictory aliases, two targets for the same
country and two mappings of the **same prefix value** to different targets are
compilation errors. Different prefixes of the same length are of course allowed.

A `GLOBAL` dispatcher holds exactly one global target, without prefix and
without country alias. A well formed country context is kept, normalized, in the
result but never routes.

```hcl
identifier "vat" "BE" {
  canonicalizer   = canonicalizer.vat.be
  format          = format.vat.be
  checksum        = checksum.vat.be
  default_profile = "compatible"

  source {
    id               = "be-fps-finance-vat"
    url              = "https://..."
    authority        = "Service public federal Finances"
    title            = "Numero de TVA belge"
    accessed_at      = "2026-08-18"
    jurisdiction     = "BE"
    language         = "fr"
    notes            = "..."
    license_or_terms = "..."
  }
}
```

A definition declares either a `checksum` reference or an explicit absence:

```hcl
no_checksum {
  reason_code = "unsupported_checksum"
  notes       = "why no algorithm is applied"
}
```

Every definition must be referenced by exactly one dispatcher target of the same
kind and the same country. Orphan definitions and inconsistencies are refused.

## 12. Profiles

`compatible` is the normative default. It accepts the current variants and the
documented historical variants that can still legitimately appear.
`strict_current` is opt-in and only restricts the accepted variants: it must
never change the canonicalization shared by both profiles, so a profile
difference always lives in a format rule, never in a canonicalizer.

## 13. Provenance

Every rejecting rule must be linked to at least one source, holding an `id`, a
`url`, an `authority`, a `title`, an `accessed_at` date, a `jurisdiction`, a
`language`, `notes`, `license_or_terms` and an optional `archive_url`.

Official primary sources are preferred. A third party implementation such as
`python-stdnum` can serve as a cross-check, never as the sole authority when the
official documentation exists.

A rule change must explain the previous semantics, the new semantics, the false
positive and false negative risk, the new conformance cases, and the date and
source of the change.

# Synthetic rule set exercising every V1 operation of the IR.
#
# It is never published: it exists only so that the reference interpreter and
# the bundle validator cover 100 % of the operation dispatch.

canonicalizer "probe" "dispatch" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars(["-"]),
  ]
}

canonicalizer "probe" "full" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "/"]),
    replace_prefix("ZZ", "XX"),
    prepend_country_if_missing(),
    when(
      all(
        length_eq(value(), 4),
        any(starts_with(value(), "XX"), starts_with(value(), "YY")),
        not(is_empty(value())),
        ascii_charset(value(), "0123456789XY"),
      ),
      insert(2, "0"),
      left_pad(7, "9"),
      prepend("P"),
      append("S"),
    ),
  ]
}

format "probe" "generic" {
  subject = value()

  capture "body" {
    value = slice_from(subject(), 2)
  }

  capture "head" {
    value = slice_to(subject(), 2)
  }

  capture "tail" {
    value = after_first(subject(), "XX")
  }

  capture "front" {
    value = before_first(concat(subject(), "|", subject()), "|")
  }

  capture "stripped" {
    value = strip_prefix(subject(), "XX")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "probe.empty"),
    require(not(is_absent(capture.body)), "invalid_format", "probe.body"),
    require(length_between(subject(), 3, 32), "invalid_length", "probe.length"),
    require(length_in(subject(), [10, 11, 12, 13, 14, 15, 16]), "invalid_length", "probe.length_in"),
    require(ascii_upper_letters(capture.head), "invalid_characters", "probe.head"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "probe.alnum"),
    require(ascii_digits(capture.body), "invalid_characters", "probe.digits"),
    require(equals(capture.head, "XX"), "invalid_format", "probe.prefix"),
    require(starts_with(subject(), "XX"), "invalid_format", "probe.starts"),
    require(ends_with(subject(), "7"), "invalid_format", "probe.ends"),
    require(prefix_in(subject(), ["XX", "YY"]), "invalid_format", "probe.prefix_in"),
    require(char_at_in(subject(), 2, "0123456789"), "invalid_characters", "probe.char_at"),
    require(contains(subject(), "X"), "invalid_format", "probe.contains"),
    require(equals(country_code(), "XX"), "invalid_format", "probe.country"),
    require(
      any(profile_is("compatible"), profile_is("strict_current")),
      "invalid_format",
      "probe.profile",
    ),
    require(not(equals(capture.front, capture.tail)), "invalid_format", "probe.views"),
    require(not(is_absent(capture.stripped)), "invalid_format", "probe.stripped"),
  ]

  use_format {
    rule  = format.probe.body
    input = capture.body
  }
}

format "probe" "body" {
  checks = [
    require(ascii_digits(subject()), "invalid_characters", "probe.body.digits"),
    require(length_eq(subject(), 8), "invalid_length", "probe.body.length"),
  ]
}

checksum "probe" "body" {
  rule = luhn(subject())
}

checksum "probe" "generic" {
  subject = slice_from(value(), 2)

  rule = choose(
    when_checksum(
      is_empty(slice(subject(), 0, 0)),
      all_checks(
        any_check(
          compare_digit(
            remainder_map(
              modulo(
                weighted_sum(slice(subject(), 0, 4), [1, 2, 3, 4], "left", "digit_value"),
                11,
              ),
              [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
            ),
            subject(), 7,
          ),
          compare_slice(
            complement(mod_digits(slice(subject(), 0, 4), 97), 97),
            subject(), 6, 8,
          ),
          compare_constant(
            modulo(weighted_sum(slice(subject(), 0, 4), [1], "cycle", "digit_value"), 5),
            0,
          ),
          choose(
            when_checksum(
              integer_is(modulo(weighted_sum(slice(subject(), 0, 4), [1], "cycle", "digit_value"), 11), 10),
              compare_constant(modulo(digits_to_integer(slice(subject(), 0, 4)), 7), 0),
            ),
            compare_constant(modulo(digits_to_integer(slice(subject(), 0, 4)), 3), 0),
          ),
          compare_digit(
            modulo(digits_to_integer(slice(subject(), 0, 4)), 10),
            subject(), 0,
          ),
          compare_digit(
            modulo(
              weighted_sum(slice(subject(), 0, 4), [1, 2], "cycle", "alnum_base36"),
              10,
            ),
            subject(), 1,
          ),
          compare_digit(
            modulo(
              weighted_sum(slice(subject(), 0, 4), [3, 5, 7, 11, 13], "right", "digit_value"),
              10,
            ),
            subject(), 2,
          ),
          apply_checksum(checksum.probe.body, subject()),
          iso7064_mod97_10(concat(subject(), country_code())),
        ),
      ),
    ),
    unsupported_checksum("checksum_not_published"),
  )
}

dispatcher "probe" {
  aliases           = ["probe_alias"]
  pre_canonicalizer = canonicalizer.probe.dispatch

  country_aliases = {
    "ZZ" = "XX"
  }

  target {
    country_code                     = "XX"
    accepted_prefixes                = ["XX", "YY"]
    canonical_prefix                 = "XX"
    identifier                       = identifier.probe.XX
    allow_unprefixed_without_country = true
  }
}

identifier "probe" "XX" {
  canonicalizer   = canonicalizer.probe.full
  format          = format.probe.generic
  checksum        = checksum.probe.generic
  default_profile = "compatible"

  source {
    id               = "probe-source"
    url              = "https://example.invalid/probe"
    authority        = "EntID"
    title            = "Synthetic operation probe"
    accessed_at      = "2026-08-18"
    jurisdiction     = "XX"
    language         = "en"
    notes            = "Covers every V1 operation of the IR; never published."
    license_or_terms = "Apache-2.0"
    tier = "primary"
  }
}

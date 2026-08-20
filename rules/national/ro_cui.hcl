# Romania - unique registration code (CUI)
#
# The registration part of the Romania EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "ro" "cui" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "ro" "cui" {
  checks = [
    require(not(is_empty(subject())), "empty", "ro.cui.empty"),
    require(length_between(subject(), 2, 10), "invalid_length", "ro.cui.length"),
    require(ascii_digits(subject()), "invalid_characters", "ro.cui.characters"),
  ]
}

checksum "ro" "cui" {
  rule = choose(
    when_checksum(
      length_eq(subject(), 2),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 1), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 1,
      ),
    ),
    when_checksum(
      length_eq(subject(), 3),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 2), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 2,
      ),
    ),
    when_checksum(
      length_eq(subject(), 4),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 3), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 3,
      ),
    ),
    when_checksum(
      length_eq(subject(), 5),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 4), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 4,
      ),
    ),
    when_checksum(
      length_eq(subject(), 6),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 5), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 5,
      ),
    ),
    when_checksum(
      length_eq(subject(), 7),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 6), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 6,
      ),
    ),
    when_checksum(
      length_eq(subject(), 8),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 7), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 7,
      ),
    ),
    when_checksum(
      length_eq(subject(), 9),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 8), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 8,
      ),
    ),
    when_checksum(
      length_eq(subject(), 10),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(subject(), 0, 9), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        subject(), 9,
      ),
    ),
  )
}

dispatcher "cui" {
  aliases           = ["ro_cui", "cif"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "RO"
    identifier                       = identifier.cui.RO
    allow_unprefixed_without_country = true
  }
}

identifier "cui" "RO" {
  canonicalizer   = canonicalizer.ro.cui
  format          = format.ro.cui
  checksum        = checksum.ro.cui
  default_profile = "compatible"

  source {
    id               = "ro-onrc-cui"
    url              = "https://www.onrc.ro"
    authority        = "Oficiul National al Registrului Comertului (ONRC)"
    title            = "Registrul comertului - codul unic de inregistrare"
    accessed_at      = "2026-08-20"
    jurisdiction     = "RO"
    language         = "ro"
    notes            = "The register publishes the CUI as the unique registration code of a company. It does not publish the algorithm closing it."
    license_or_terms = "Romanian public sector information"
    tier             = "primary"
  }
  source {
    id               = "ro-cui-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "RO"
    language         = "en"
    notes            = "Weights 7, 5, 3, 2, 1, 7, 5, 3, 2 aligned on the right of the digits preceding the check digit, the sum multiplied by ten, taken modulo eleven then modulo ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

# Romania - EUID
#
# The registration part is the CUI, which runs from two to ten digits. Its last
# digit closes a weighted modulo 11 over the others, and the weights are aligned
# on the right: a short number uses the tail of the weight list, not its head.
#
# The rule needs one branch per length because a view cannot be taken from the
# start of a value to one position before its end - `slice` takes fixed bounds.
# Each branch is the same computation on a different length.

canonicalizer "euid" "ro" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "ro" {
  capture "register" {
    value = before_first(after_first(subject(), "RO"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.ro.empty"),
    require(starts_with(subject(), "RO"), "invalid_format", "euid.ro.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.ro.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.ro.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.ro.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.ro.register_characters"),
    require(length_between(capture.registration, 2, 10), "invalid_length", "euid.ro.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.ro.registration_characters"),
  ]
}

checksum "euid" "ro" {
  rule = choose(
    when_checksum(
      length_eq(after_first(subject(), "."), 2),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 1), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 1,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 3),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 2), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 2,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 4),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 3), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 3,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 5),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 4), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 4,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 6),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 5), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 5,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 7),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 6), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 6,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 8),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 7), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 7,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 9),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 8), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 8,
      ),
    ),
    when_checksum(
      length_eq(after_first(subject(), "."), 10),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(after_first(subject(), "."), 0, 9), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        after_first(subject(), "."), 9,
      ),
    ),
  )
}

identifier "euid" "RO" {
  canonicalizer   = canonicalizer.euid.ro
  format          = format.euid.ro
  checksum        = checksum.euid.ro
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

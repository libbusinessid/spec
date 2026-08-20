# Romania - VAT identification number
#
# Two to ten digits, the CUI of the commercial register. The weights are aligned
# on the right, so a short number uses the tail of the list, and the rule needs a
# branch per length because a view cannot stop one position before the end.

canonicalizer "vat" "ro" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "ro" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.ro.empty"),
    require(length_between(subject(), 4, 12), "invalid_length", "vat.ro.length"),
    require(starts_with(subject(), "RO"), "invalid_format", "vat.ro.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.ro.characters"),
  ]
}

checksum "vat" "ro" {
  rule = choose(
    when_checksum(
      length_eq(slice_from(subject(), 2), 2),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 1), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 1,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 3),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 2), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 2,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 4),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 3), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 3,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 5),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 4), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 4,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 6),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 5), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 5,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 7),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 6), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 6,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 8),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 7), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 7,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 9),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 8,
      ),
    ),
    when_checksum(
      length_eq(slice_from(subject(), 2), 10),
      compare_digit(
        remainder_map(
          modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 9), [70, 50, 30, 20, 10, 70, 50, 30, 20], "right", "digit_value"), 11),
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
        ),
        slice_from(subject(), 2), 9,
      ),
    ),
  )
}

identifier "vat" "RO" {
  canonicalizer   = canonicalizer.vat.ro
  format          = format.vat.ro
  checksum        = checksum.vat.ro
  default_profile = "compatible"

  source {
    id               = "eu-vies-number-structure"
    url              = "https://ec.europa.eu/taxation_customs/vies/"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "VIES - VAT number structure per Member State"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "Published length and prefix of each Member State VAT identification number."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
    tier             = "primary"
  }

  source {
    id               = "ro-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "RO"
    language         = "en"
    notes            = "Weights 7, 5, 3, 2, 1, 7, 5, 3, 2 aligned on the right of the digits before the check, the sum multiplied by ten, taken modulo eleven then modulo ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

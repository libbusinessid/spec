# Bulgaria - VAT identification number
#
# Nine digits, the BULSTAT identifier of a legal person.
#
# The ten digit form is an EGN, the personal number of a natural person. It is
# not accepted: a VAT number of an individual identifies that individual, and
# validating it is not what this corpus is for.

canonicalizer "vat" "bg" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "bg" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.bg.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.bg.length"),
    require(starts_with(subject(), "BG"), "invalid_format", "vat.bg.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.bg.characters"),
  ]
}

checksum "vat" "bg" {
  rule = choose(
    when_checksum(
      integer_is(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [1, 2, 3, 4, 5, 6, 7, 8], "left", "digit_value"), 11), 10),
      compare_digit(remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [3, 4, 5, 6, 7, 8, 9, 10], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]), slice_from(subject(), 2), 8),
    ),
    compare_digit(remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [1, 2, 3, 4, 5, 6, 7, 8], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]), slice_from(subject(), 2), 8),
  )
}

identifier "vat" "BG" {
  canonicalizer   = canonicalizer.vat.bg
  format          = format.vat.bg
  checksum        = checksum.vat.bg
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
    id               = "bg-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "BG"
    language         = "en"
    notes            = "Weights 1 to 8 over the first eight digits, sent through weights 3 to 10 when the remainder reaches ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

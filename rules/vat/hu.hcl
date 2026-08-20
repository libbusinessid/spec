# Hungary - VAT identification number (adoszam)
#
# Eight digits, the last completing the weighted sum to the next multiple of ten.

canonicalizer "vat" "hu" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "hu" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.hu.empty"),
    require(length_eq(subject(), 10), "invalid_length", "vat.hu.length"),
    require(starts_with(subject(), "HU"), "invalid_format", "vat.hu.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.hu.characters"),
  ]
}

checksum "vat" "hu" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 7), [9, 7, 3, 1, 9, 7, 3], "left", "digit_value"), 10), [0, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    slice_from(subject(), 2), 7,
  )
}

identifier "vat" "HU" {
  canonicalizer   = canonicalizer.vat.hu
  format          = format.vat.hu
  checksum        = checksum.vat.hu
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
    id               = "hu-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "HU"
    language         = "en"
    notes            = "Weights 9, 7, 3, 1 repeated over the first seven digits, the check digit completing the sum to the next multiple of ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

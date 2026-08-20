# Norway - VAT identification number (organisasjonsnummer with MVA)
#
# Nine digits. A remainder of one leaves no check digit, so such a number is
# never issued and the remainder table sends that case where no digit follows.

canonicalizer "vat" "no" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "no" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.no.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.no.length"),
    require(starts_with(subject(), "NO"), "invalid_format", "vat.no.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.no.characters"),
  ]
}

checksum "vat" "no" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [3, 2, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11), [0, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    slice_from(subject(), 2), 8,
  )
}

identifier "vat" "NO" {
  canonicalizer   = canonicalizer.vat.no
  format          = format.vat.no
  checksum        = checksum.vat.no
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
    id               = "no-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "NO"
    language         = "en"
    notes            = "Weights 3, 2, 7, 6, 5, 4, 3, 2 over the first eight digits, the check digit being eleven minus the remainder, zero when that reaches eleven, and never issued when it reaches ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

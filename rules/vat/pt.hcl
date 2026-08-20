# Portugal - VAT identification number (NIF)
#
# Nine digits, the same number as the NIPC of the commercial register.

canonicalizer "vat" "pt" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "pt" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.pt.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.pt.length"),
    require(starts_with(subject(), "PT"), "invalid_format", "vat.pt.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.pt.characters"),
  ]
}

checksum "vat" "pt" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [9, 8, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11), [0, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    slice_from(subject(), 2), 8,
  )
}

identifier "vat" "PT" {
  canonicalizer   = canonicalizer.vat.pt
  format          = format.vat.pt
  checksum        = checksum.vat.pt
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
    id               = "pt-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PT"
    language         = "en"
    notes            = "Weights 9 to 2 over the first eight digits, the check digit being eleven minus the remainder when that is at least two, and zero otherwise."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

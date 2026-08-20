# Estonia - VAT identification number (KMKR number)
#
# Nine digits, closed by a modulo ten complement rather than the modulo eleven
# the register number uses.

canonicalizer "vat" "ee" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "ee" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.ee.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.ee.length"),
    require(starts_with(subject(), "EE"), "invalid_format", "vat.ee.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.ee.characters"),
  ]
}

checksum "vat" "ee" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [3, 7, 1, 3, 7, 1, 3, 7], "left", "digit_value"), 10), [0, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    slice_from(subject(), 2), 8,
  )
}

identifier "vat" "EE" {
  canonicalizer   = canonicalizer.vat.ee
  format          = format.vat.ee
  checksum        = checksum.vat.ee
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
    id               = "ee-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EE"
    language         = "en"
    notes            = "Weights 3, 7, 1 repeated over the first eight digits, the check digit completing the sum to the next multiple of ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

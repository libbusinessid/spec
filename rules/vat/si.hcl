# Slovenia - VAT identification number (davcna stevilka)
#
# Eight digits. A remainder of one leaves no check digit, exactly as in Finland,
# and the same table expresses it.

canonicalizer "vat" "si" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "si" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.si.empty"),
    require(length_eq(subject(), 10), "invalid_length", "vat.si.length"),
    require(starts_with(subject(), "SI"), "invalid_format", "vat.si.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.si.characters"),
  ]
}

checksum "vat" "si" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 7), [8, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11), [0, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    slice_from(subject(), 2), 7,
  )
}

identifier "vat" "SI" {
  canonicalizer   = canonicalizer.vat.si
  format          = format.vat.si
  checksum        = checksum.vat.si
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
    id               = "si-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SI"
    language         = "en"
    notes            = "Weights 8 to 2 over the first seven digits, the check digit being eleven minus the remainder, zero when that reaches eleven, and never issued when it reaches ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

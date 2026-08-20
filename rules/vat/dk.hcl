# Denmark - VAT identification number (momsnummer)
#
# The eight digits of the CVR number, whose weighted sum vanishes modulo eleven
# rather than carrying a separate check digit.

canonicalizer "vat" "dk" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "dk" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.dk.empty"),
    require(length_eq(subject(), 10), "invalid_length", "vat.dk.length"),
    require(starts_with(subject(), "DK"), "invalid_format", "vat.dk.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.dk.characters"),
  ]
}

checksum "vat" "dk" {
  rule = compare_constant(
    modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [2, 7, 6, 5, 4, 3, 2, 1], "left", "digit_value"), 11),
    0,
  )
}

identifier "vat" "DK" {
  canonicalizer   = canonicalizer.vat.dk
  format          = format.vat.dk
  checksum        = checksum.vat.dk
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
    id               = "dk-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DK"
    language         = "en"
    notes            = "Weights 2, 7, 6, 5, 4, 3, 2, 1 over the eight digits, the whole sum vanishing modulo eleven."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

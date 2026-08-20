# Poland - VAT identification number (NIP)
#
# Ten digits. A remainder of ten leaves no check digit, so the table sends it
# where no digit can follow.

canonicalizer "vat" "pl" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "pl" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.pl.empty"),
    require(length_eq(subject(), 12), "invalid_length", "vat.pl.length"),
    require(starts_with(subject(), "PL"), "invalid_format", "vat.pl.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.pl.characters"),
  ]
}

checksum "vat" "pl" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 9), [6, 5, 7, 2, 3, 4, 5, 6, 7], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]),
    slice_from(subject(), 2), 9,
  )
}

identifier "vat" "PL" {
  canonicalizer   = canonicalizer.vat.pl
  format          = format.vat.pl
  checksum        = checksum.vat.pl
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
    id               = "pl-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PL"
    language         = "en"
    notes            = "Weights 6, 5, 7, 2, 3, 4, 5, 6, 7 over the first nine digits, the check digit being the remainder modulo eleven, a remainder of ten never being issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

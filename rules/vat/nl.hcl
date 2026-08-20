# Netherlands - VAT identification number (btw-identificatienummer)
#
# Twelve characters: nine digits, the letter B, then a two digit sub number. The
# check closes the first nine digits, and a remainder of ten is never issued.

canonicalizer "vat" "nl" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "nl" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.nl.empty"),
    require(length_eq(subject(), 14), "invalid_length", "vat.nl.length"),
    require(starts_with(subject(), "NL"), "invalid_format", "vat.nl.prefix"),
    require(ascii_digits(slice_from(subject(), 12)), "invalid_characters", "vat.nl.characters"),
    require(char_at_in(subject(), 11, "B"), "invalid_format", "vat.nl.b_marker"),
    require(ascii_digits(slice(subject(), 2, 11)), "invalid_characters", "vat.nl.body_characters"),
    require(ascii_digits(slice(subject(), 12, 14)), "invalid_characters", "vat.nl.sub_characters"),
  ]
}

checksum "vat" "nl" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [9, 8, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]),
    slice_from(subject(), 2), 8,
  )
}

identifier "vat" "NL" {
  canonicalizer   = canonicalizer.vat.nl
  format          = format.vat.nl
  checksum        = checksum.vat.nl
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
    id               = "nl-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "NL"
    language         = "en"
    notes            = "Weights 9 to 2 over the first eight digits, the check digit being the remainder modulo eleven, a remainder of ten never being issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

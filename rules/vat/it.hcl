# Italy - VAT identification number (partita IVA)
#
# Eleven digits closed by a Luhn digit, the same number the registro delle
# imprese issues.

canonicalizer "vat" "it" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "it" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.it.empty"),
    require(length_eq(subject(), 13), "invalid_length", "vat.it.length"),
    require(starts_with(subject(), "IT"), "invalid_format", "vat.it.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.it.characters"),
  ]
}

checksum "vat" "it" {
  rule = luhn(slice_from(subject(), 2))
}

identifier "vat" "IT" {
  canonicalizer   = canonicalizer.vat.it
  format          = format.vat.it
  checksum        = checksum.vat.it
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
    id               = "it-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "IT"
    language         = "en"
    notes            = "Eleven digits, the last a Luhn check over all eleven."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

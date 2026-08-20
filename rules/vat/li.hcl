# Liechtenstein - VAT identification number
#
# Five digits. The administration publishes no check algorithm over them, so no
# checksum is applied rather than one being guessed.

canonicalizer "vat" "li" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "li" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.li.empty"),
    require(length_eq(subject(), 7), "invalid_length", "vat.li.length"),
    require(starts_with(subject(), "LI"), "invalid_format", "vat.li.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.li.characters"),
  ]
}

identifier "vat" "LI" {
  canonicalizer   = canonicalizer.vat.li
  format          = format.vat.li
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The administration publishes the five digit number and no algorithm over it."
  }

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
}

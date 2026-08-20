# Austria - VAT identification number (UID)
#
# ATU followed by eight characters: seven digits and a check digit.
#
# The check doubles alternate digits and sums the digits of each result, which is
# the Luhn computation, but it then adds four before taking the complement. That
# constant shifts the sum outside what a weighted sum can express, and the IR has
# no addition. The step reports unsupported rather than risking a false negative,
# and the format is still asserted.

canonicalizer "vat" "at" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "at" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.at.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.at.length"),
    require(starts_with(subject(), "AT"), "invalid_format", "vat.at.prefix"),
    require(ascii_digits(slice_from(subject(), 3)), "invalid_characters", "vat.at.characters"),
    require(char_at_in(subject(), 2, "U"), "invalid_format", "vat.at.u_marker"),
  ]
}

identifier "vat" "AT" {
  canonicalizer   = canonicalizer.vat.at
  format          = format.vat.at
  default_profile = "compatible"

  no_checksum {
    reason_code = "unsupported_checksum"
    notes       = "The published check adds a constant to a Luhn style sum before taking its complement, and IR v1 has no addition to express that shift."
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

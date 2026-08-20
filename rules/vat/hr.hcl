# Croatia - VAT identification number (OIB)
#
# Eleven digits closed by the iterative ISO 7064 MOD 11,10 procedure, which
# carries a data dependent state from one digit to the next.
#
# IR v1 has no iterative primitive, exactly as for the German VAT number, so the
# step reports unsupported rather than risking a false negative. The format is
# still asserted.

canonicalizer "vat" "hr" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "hr" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.hr.empty"),
    require(length_eq(subject(), 13), "invalid_length", "vat.hr.length"),
    require(starts_with(subject(), "HR"), "invalid_format", "vat.hr.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.hr.characters"),
  ]
}

identifier "vat" "HR" {
  canonicalizer   = canonicalizer.vat.hr
  format          = format.vat.hr
  default_profile = "compatible"

  no_checksum {
    reason_code = "unsupported_checksum"
    notes       = "The ISO 7064 MOD 11,10 procedure is iterative and stateful, and IR v1 reserves no capability able to express it."
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

# Germany - VAT identification number (Umsatzsteuer-Identifikationsnummer)
#
# The German VAT number is DE followed by nine digits. Its check digit follows
# the iterative MOD 11,10 procedure of DIN ISO 7064. That procedure carries a
# data dependent state from one digit to the next and cannot be expressed with
# the IR v1 operation set, whose arithmetic is limited to weighted sums,
# digit-by-digit moduli and constant remainder tables.
#
# Adding an iterative primitive would require a new capability ID, its
# implementation in the four engines and its publication, in that order
# (spec.md section 7.4). Until then the checksum step reports `unsupported`,
# never `invalid`: refusing a valid identifier is the most serious defect of
# the project.

canonicalizer "vat" "de" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "de" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.de.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.de.length"),
    require(starts_with(subject(), "DE"), "invalid_format", "vat.de.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.de.characters"),
  ]
}

identifier "vat" "DE" {
  canonicalizer   = canonicalizer.vat.de
  format          = format.vat.de
  default_profile = "compatible"

  no_checksum {
    reason_code = "unsupported_checksum"
    notes       = "The DIN ISO 7064 MOD 11,10 procedure is iterative and stateful; IR v1 reserves no capability able to express it, so the checksum stays unsupported instead of risking a false negative."
  }

  source {
    id               = "de-bzst-ustid"
    url              = "https://www.bzst.de/DE/Unternehmen/Identifikationsnummern/Umsatzsteuer-Identifikationsnummer/umsatzsteuer-identifikationsnummer_node.html"
    authority        = "Bundeszentralamt fuer Steuern (BZSt)"
    title            = "Umsatzsteuer-Identifikationsnummer"
    accessed_at      = "2026-08-18"
    jurisdiction     = "DE"
    language         = "de"
    notes            = "DE followed by nine digits. The published check digit procedure is the iterative MOD 11,10 of DIN ISO 7064."
    license_or_terms = "Public sector information published by the German federal tax administration"
  }

  source {
    id               = "eu-vies-number-structure"
    url              = "https://ec.europa.eu/taxation_customs/vies/"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "VIES - VAT number structure per Member State"
    accessed_at      = "2026-08-18"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "Cross-check of the published length and prefix of each Member State VAT identification number."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
  }
}

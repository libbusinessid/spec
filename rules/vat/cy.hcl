# Cyprus - VAT identification number
#
# Eight digits closed by a letter.
#
# The check maps the digits at even positions through a table that is not a
# weighted sum - one, zero, five, seven, nine, thirteen and upward - and compares
# the result against a letter derived from a remainder modulo twenty six. Neither
# the mapping nor the letter comparison is expressible in IR v1, so the step
# reports unsupported.

canonicalizer "vat" "cy" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "cy" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.cy.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.cy.length"),
    require(starts_with(subject(), "CY"), "invalid_format", "vat.cy.prefix"),
    require(char_at_in(subject(), 10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"), "invalid_format", "vat.cy.check_letter"),
    require(ascii_digits(slice(subject(), 2, 10)), "invalid_characters", "vat.cy.body_characters"),
  ]
}

identifier "vat" "CY" {
  canonicalizer   = canonicalizer.vat.cy
  format          = format.vat.cy
  default_profile = "compatible"

  no_checksum {
    reason_code = "unsupported_checksum"
    notes       = "The check maps alternate digits through a non linear table and compares a remainder modulo twenty six against a letter, neither of which IR v1 can express."
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

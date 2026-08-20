# Ireland - VAT identification number
#
# Seven digits closed by a letter, and optionally a second letter for a group
# registration. An older layout put a letter or a symbol in second position.
#
# The check derives a letter from a weighted sum modulo twenty three - W for a
# remainder of zero, otherwise the letter at that offset in the alphabet. The IR
# has no mapping from a number to a letter, exactly as for the Spanish letter
# check, so the step reports unsupported while the shape is asserted.

canonicalizer "vat" "ie" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "ie" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.ie.empty"),
    require(length_between(subject(), 10, 11), "invalid_length", "vat.ie.length"),
    require(starts_with(subject(), "IE"), "invalid_format", "vat.ie.prefix"),
    require(ascii_alphanumeric(slice_from(subject(), 2)), "invalid_characters", "vat.ie.characters"),
    require(
      any(
        // Seven digits and a check letter.
        all(
          length_eq(subject(), 10),
          ascii_digits(slice(subject(), 2, 9)),
          char_at_in(subject(), 9, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        // Seven digits, a check letter and a group letter.
        all(
          length_eq(subject(), 11),
          ascii_digits(slice(subject(), 2, 9)),
          char_at_in(subject(), 9, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
          char_at_in(subject(), 10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        // The older layout: a digit, a letter, five digits and a check letter.
        all(
          length_eq(subject(), 10),
          ascii_digits(slice(subject(), 2, 3)),
          char_at_in(subject(), 3, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
          ascii_digits(slice(subject(), 4, 9)),
          char_at_in(subject(), 9, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
      ),
      "invalid_format",
      "vat.ie.shape",
    ),
  ]
}

identifier "vat" "IE" {
  canonicalizer   = canonicalizer.vat.ie
  format          = format.vat.ie
  default_profile = "compatible"

  no_checksum {
    reason_code = "unsupported_checksum"
    notes       = "The check is a letter derived from a weighted sum modulo twenty three, and IR v1 has no mapping from a number to a letter."
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

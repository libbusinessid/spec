# Sweden - VAT identification number (momsregistreringsnummer)
#
# Twelve digits: the ten of the organisationsnummer, closed by their Luhn, and a
# two digit branch number which is never zero.

canonicalizer "vat" "se" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "se" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.se.empty"),
    require(length_eq(subject(), 14), "invalid_length", "vat.se.length"),
    require(starts_with(subject(), "SE"), "invalid_format", "vat.se.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.se.characters"),
    require(
      not(all(char_at_in(subject(), 12, "0"), char_at_in(subject(), 13, "0"))),
      "invalid_format",
      "vat.se.branch",
    ),
  ]
}

checksum "vat" "se" {
  rule = luhn(slice(slice_from(subject(), 2), 0, 10))
}

identifier "vat" "SE" {
  canonicalizer   = canonicalizer.vat.se
  format          = format.vat.se
  checksum        = checksum.vat.se
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
    id               = "se-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SE"
    language         = "en"
    notes            = "The first ten digits are the organisationsnummer closed by a Luhn digit; the last two are a branch number running from one to ninety nine."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

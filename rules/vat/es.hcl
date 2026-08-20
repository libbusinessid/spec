# Spain - VAT identification number (NIF)
#
# The CIF form, which designates a legal person: a leading letter, seven digits
# and a check character.
#
# The DNI and NIE forms identify natural persons and are not accepted, as in the
# register rule. Where the check is a letter it encodes the same digit through
# the alphabet JABCDEFGHI, and turning it back into a digit needs a mapping the
# IR does not have, so those numbers report unsupported.

canonicalizer "vat" "es" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "es" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.es.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.es.length"),
    require(starts_with(subject(), "ES"), "invalid_format", "vat.es.prefix"),
    require(ascii_alphanumeric(slice_from(subject(), 2)), "invalid_characters", "vat.es.characters"),
    require(char_at_in(subject(), 2, "ABCDEFGHJNPQRSUVW"), "invalid_format", "vat.es.legal_entity"),
    require(ascii_digits(slice(subject(), 3, 10)), "invalid_characters", "vat.es.body"),
  ]
}

checksum "vat" "es" {
  rule = choose(
    when_checksum(
      char_at_in(subject(), 10, "0123456789"),
      luhn(slice(subject(), 3, 11)),
    ),
    unsupported_checksum("checksum_not_published"),
  )
}

identifier "vat" "ES" {
  canonicalizer   = canonicalizer.vat.es
  format          = format.vat.es
  checksum        = checksum.vat.es
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
    id               = "es-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "ES"
    language         = "en"
    notes            = "The check doubles the odd positions of the body and sums the digits of each result, which is the Luhn computation. A letter check encodes the same digit through the alphabet JABCDEFGHI."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

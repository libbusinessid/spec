# EORI number
#
# The identifier customs authorities of the European Union use for an economic
# operator: a two letter country code followed by up to fifteen alphanumeric
# characters, whose shape each Member State fixes for itself.
#
# The regulation states the country code and the maximum length and leaves the
# national part to each authority, so nothing beyond that shape can be asserted
# for every issuer, and no check algorithm is published across them.

canonicalizer "global" "eori" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
  ]
}

format "global" "eori" {
  checks = [
    require(not(is_empty(subject())), "empty", "eori.empty"),
    require(length_between(subject(), 3, 17), "invalid_length", "eori.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "eori.characters"),
    require(ascii_upper_letters(slice(subject(), 0, 2)), "invalid_format", "eori.country_prefix"),
  ]
}

dispatcher "eori" {
  aliases           = ["eori_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    identifier = identifier.eori.GLOBAL
  }
}

identifier "eori" "GLOBAL" {
  canonicalizer   = canonicalizer.global.eori
  format          = format.global.eori
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The national part of an EORI is fixed by each Member State, and no algorithm is published across issuers."
  }

  source {
    id               = "eu-eori-number"
    url              = "https://taxation-customs.ec.europa.eu/customs-4/customs-procedures-import-and-export/customs-procedures/economic-operators-registration-and-identification-number-eori_en"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "Economic Operators Registration and Identification number (EORI)"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "A two letter country code followed by a national part of at most fifteen alphanumeric characters."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
    tier             = "primary"
  }
}

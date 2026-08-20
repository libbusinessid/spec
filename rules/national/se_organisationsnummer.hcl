# Sweden - organisationsnummer
#
# The registration part of the Sweden EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "se" "organisationsnummer" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "se" "organisationsnummer" {
  checks = [
    require(not(is_empty(subject())), "empty", "se.organisationsnummer.empty"),
    require(length_eq(subject(), 10), "invalid_length", "se.organisationsnummer.length"),
    require(ascii_digits(subject()), "invalid_characters", "se.organisationsnummer.characters"),
    require(
      char_at_in(subject(), 2, "23456789"),
      "invalid_format",
      "se.organisationsnummer.group",
    ),
  ]
}

checksum "se" "organisationsnummer" {
  rule = luhn(subject())
}

dispatcher "organisationsnummer" {
  aliases           = ["se_organisationsnummer"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "SE"
    identifier                       = identifier.organisationsnummer.SE
    allow_unprefixed_without_country = true
  }
}

identifier "organisationsnummer" "SE" {
  canonicalizer   = canonicalizer.se.organisationsnummer
  format          = format.se.organisationsnummer
  checksum        = checksum.se.organisationsnummer
  default_profile = "compatible"

  source {
    id               = "se-skatteverket-organisationsnummer"
    url              = "https://docs.swedenconnect.se/technical-framework/mirror/skv/skv709-8.pdf"
    authority        = "Skatteverket (Swedish Tax Agency)"
    title            = "Organisationsnummer, SKV 709 utgava 8"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SE"
    language         = "sv"
    notes            = "Ten digits under SFS 1974:174. The third digit is always at least two so the number cannot be confused with a personnummer, and the last is a check digit. The brochure names the check digit but does not state the algorithm computing it."
    license_or_terms = "Swedish public sector information"
    tier             = "primary"
  }
  source {
    id               = "se-organisationsnummer-luhn"
    url              = "https://sv.wikipedia.org/wiki/Organisationsnummer"
    authority        = "Wikipedia"
    title            = "Organisationsnummer - kontrollsiffra"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SE"
    language         = "sv"
    notes            = "States that the check digit follows the Luhn algorithm over the ten digits. Skatteverket does not publish the algorithm, so this entry records where it was read rather than an authority for it."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}

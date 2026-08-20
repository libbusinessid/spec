# Sweden - EUID
#
# The registration part is the organisationsnummer: ten digits whose third is
# always at least two, so that it cannot be mistaken for a personnummer, and
# whose last is a check digit.
#
# Skatteverket states the format but not the algorithm that closes it. The check
# digit is a Luhn over the ten digits, carried here on a secondary source. The
# constraint on the third digit comes from the authority itself and is enforced,
# which the historical Go library did not do.

canonicalizer "euid" "se" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "se" {
  capture "register" {
    value = before_first(after_first(subject(), "SE"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.se.empty"),
    require(starts_with(subject(), "SE"), "invalid_format", "euid.se.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.se.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.se.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.se.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.se.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.se.organisationsnummer
    input = capture.registration
  }
}

checksum "euid" "se" {
  rule = apply_checksum(checksum.se.organisationsnummer, after_first(subject(), "."))
}

identifier "euid" "SE" {
  canonicalizer   = canonicalizer.euid.se
  format          = format.euid.se
  checksum        = checksum.euid.se
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

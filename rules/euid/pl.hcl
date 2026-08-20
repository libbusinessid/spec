# Poland - EUID
#
# The registration part is the ten digit KRS number of the national court
# register.

canonicalizer "euid" "pl" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "pl" {
  capture "register" {
    value = before_first(after_first(subject(), "PL"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.pl.empty"),
    require(starts_with(subject(), "PL"), "invalid_format", "euid.pl.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.pl.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.pl.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.pl.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.pl.register_characters"),
    require(length_eq(capture.registration, 10), "invalid_length", "euid.pl.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.pl.registration_characters"),
  ]
}

identifier "euid" "PL" {
  canonicalizer   = canonicalizer.euid.pl
  format          = format.euid.pl
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "pl-krs-number"
    url              = "https://prs.ms.gov.pl"
    authority        = "Ministerstwo Sprawiedliwosci"
    title            = "Krajowy Rejestr Sadowy"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PL"
    language         = "pl"
    notes            = "The register publishes the ten digit KRS number."
    license_or_terms = "Polish public sector information"
    tier             = "primary"
  }
}

# Netherlands - EUID
#
# The registration part is the eight digit KVK number of the trade register.

canonicalizer "euid" "nl" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "nl" {
  capture "register" {
    value = before_first(after_first(subject(), "NL"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.nl.empty"),
    require(starts_with(subject(), "NL"), "invalid_format", "euid.nl.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.nl.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.nl.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.nl.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.nl.register_characters"),
    require(length_eq(capture.registration, 8), "invalid_length", "euid.nl.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.nl.registration_characters"),
  ]
}

identifier "euid" "NL" {
  canonicalizer   = canonicalizer.euid.nl
  format          = format.euid.nl
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "nl-kvk-nummer"
    url              = "https://www.kvk.nl"
    authority        = "Kamer van Koophandel (KVK)"
    title            = "Handelsregister - KVK-nummer"
    accessed_at      = "2026-08-20"
    jurisdiction     = "NL"
    language         = "nl"
    notes            = "The chamber publishes the eight digit KVK number."
    license_or_terms = "Dutch public sector information"
    tier             = "primary"
  }
}

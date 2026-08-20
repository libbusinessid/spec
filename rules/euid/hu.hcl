# Hungary - EUID
#
# The registration part is the cegjegyzekszam, ten digits printed as
# RR-TT-NNNNNN. The dashes are presentation, so the canonicalizer removes them;
# the dot separating the register from the number is structural and is kept.

canonicalizer "euid" "hu" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars(["-"]),
    prepend_country_if_missing(),
  ]
}

format "euid" "hu" {
  capture "register" {
    value = before_first(after_first(subject(), "HU"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.hu.empty"),
    require(starts_with(subject(), "HU"), "invalid_format", "euid.hu.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.hu.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.hu.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.hu.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.hu.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.hu.cegjegyzekszam
    input = capture.registration
  }
}

identifier "euid" "HU" {
  canonicalizer   = canonicalizer.euid.hu
  format          = format.euid.hu
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "hu-ceginformacio"
    url              = "https://www.e-cegjegyzek.hu"
    authority        = "Igazsagugyi Miniszterium"
    title            = "Cegjegyzek"
    accessed_at      = "2026-08-20"
    jurisdiction     = "HU"
    language         = "hu"
    notes            = "The register publishes the ten digit cegjegyzekszam, printed with dashes."
    license_or_terms = "Hungarian public sector information"
    tier             = "primary"
  }
}

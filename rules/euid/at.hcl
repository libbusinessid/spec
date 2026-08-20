# Austria - EUID
#
# The registration part is the Firmenbuchnummer: digits closed by a single
# letter, which identifies the check character of the register rather than a
# computed digit. The shape is checked per length because the position of the
# letter depends on it.

canonicalizer "euid" "at" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "at" {
  capture "register" {
    value = before_first(after_first(subject(), "AT"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.at.empty"),
    require(starts_with(subject(), "AT"), "invalid_format", "euid.at.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.at.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.at.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.at.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.at.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.at.firmenbuchnummer
    input = capture.registration
  }
}

identifier "euid" "AT" {
  canonicalizer   = canonicalizer.euid.at
  format          = format.euid.at
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The trailing letter is part of the register number rather than a digit computed from the others, and the register publishes no algorithm to verify it."
  }

  source {
    id               = "at-firmenbuch-nummer"
    url              = "https://www.justiz.gv.at"
    authority        = "Bundesministerium fuer Justiz"
    title            = "Firmenbuch"
    accessed_at      = "2026-08-20"
    jurisdiction     = "AT"
    language         = "de"
    notes            = "The register publishes the Firmenbuchnummer as digits followed by a single letter."
    license_or_terms = "Austrian public sector information"
    tier             = "primary"
  }
}

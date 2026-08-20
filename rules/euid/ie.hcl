# Ireland - EUID
#
# The registration part is the CRO number, five to seven digits.

canonicalizer "euid" "ie" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "ie" {
  capture "register" {
    value = before_first(after_first(subject(), "IE"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.ie.empty"),
    require(starts_with(subject(), "IE"), "invalid_format", "euid.ie.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.ie.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.ie.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.ie.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.ie.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.ie.cro_number
    input = capture.registration
  }
}

identifier "euid" "IE" {
  canonicalizer   = canonicalizer.euid.ie
  format          = format.euid.ie
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "ie-cro-number"
    url              = "https://www.cro.ie"
    authority        = "Companies Registration Office"
    title            = "CRO company number"
    accessed_at      = "2026-08-20"
    jurisdiction     = "IE"
    language         = "en"
    notes            = "The office publishes the company number as five to seven digits."
    license_or_terms = "Irish public sector information"
    tier             = "primary"
  }
}

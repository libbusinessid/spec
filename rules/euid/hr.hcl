# Croatia - EUID
#
# The registration part is the eight digit MBS of the court register.

canonicalizer "euid" "hr" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "hr" {
  capture "register" {
    value = before_first(after_first(subject(), "HR"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.hr.empty"),
    require(starts_with(subject(), "HR"), "invalid_format", "euid.hr.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.hr.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.hr.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.hr.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.hr.register_characters"),
    require(length_eq(capture.registration, 8), "invalid_length", "euid.hr.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.hr.registration_characters"),
  ]
}

identifier "euid" "HR" {
  canonicalizer   = canonicalizer.euid.hr
  format          = format.euid.hr
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "hr-sudreg-mbs"
    url              = "https://sudreg.pravosudje.hr"
    authority        = "Ministarstvo pravosuda i uprave"
    title            = "Sudski registar"
    accessed_at      = "2026-08-20"
    jurisdiction     = "HR"
    language         = "hr"
    notes            = "The court register publishes the eight digit MBS."
    license_or_terms = "Croatian public sector information"
    tier             = "primary"
  }
}

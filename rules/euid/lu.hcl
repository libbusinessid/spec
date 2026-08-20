# Luxembourg - EUID
#
# The registration part is the RCSL number: the letter B followed by four to
# six digits.

canonicalizer "euid" "lu" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "lu" {
  capture "register" {
    value = before_first(after_first(subject(), "LU"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.lu.empty"),
    require(starts_with(subject(), "LU"), "invalid_format", "euid.lu.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.lu.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.lu.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.lu.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.lu.register_characters"),
    require(length_between(capture.registration, 5, 7), "invalid_length", "euid.lu.registration_length"),
    require(starts_with(capture.registration, "B"), "invalid_format", "euid.lu.registration_prefix"),
    require(ascii_digits(slice_from(capture.registration, 1)), "invalid_characters", "euid.lu.registration_characters"),
  ]
}

identifier "euid" "LU" {
  canonicalizer   = canonicalizer.euid.lu
  format          = format.euid.lu
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "lu-rcsl-number"
    url              = "https://www.lbr.lu"
    authority        = "Luxembourg Business Registers"
    title            = "Registre de commerce et des societes"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LU"
    language         = "fr"
    notes            = "The register publishes a number formed of the letter B followed by digits."
    license_or_terms = "Luxembourg public sector information"
    tier             = "primary"
  }
}

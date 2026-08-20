# Malta - EUID
#
# The registration part is the company number: the letter C followed by four to
# six digits.

canonicalizer "euid" "mt" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "mt" {
  capture "register" {
    value = before_first(after_first(subject(), "MT"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.mt.empty"),
    require(starts_with(subject(), "MT"), "invalid_format", "euid.mt.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.mt.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.mt.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.mt.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.mt.register_characters"),
    require(length_between(capture.registration, 5, 7), "invalid_length", "euid.mt.registration_length"),
    require(starts_with(capture.registration, "C"), "invalid_format", "euid.mt.registration_prefix"),
    require(ascii_digits(slice_from(capture.registration, 1)), "invalid_characters", "euid.mt.registration_characters"),
  ]
}

identifier "euid" "MT" {
  canonicalizer   = canonicalizer.euid.mt
  format          = format.euid.mt
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "mt-mbr-number"
    url              = "https://mbr.mt"
    authority        = "Malta Business Registry"
    title            = "Company registration number"
    accessed_at      = "2026-08-20"
    jurisdiction     = "MT"
    language         = "en"
    notes            = "The registry publishes a number formed of the letter C followed by digits."
    license_or_terms = "Maltese public sector information"
    tier             = "primary"
  }
}

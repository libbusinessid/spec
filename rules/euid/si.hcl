# Slovenia - EUID
#
# The registration part is the seven digit maticna stevilka of the business
# register.

canonicalizer "euid" "si" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "si" {
  capture "register" {
    value = before_first(after_first(subject(), "SI"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.si.empty"),
    require(starts_with(subject(), "SI"), "invalid_format", "euid.si.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.si.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.si.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.si.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.si.register_characters"),
    require(length_eq(capture.registration, 7), "invalid_length", "euid.si.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.si.registration_characters"),
  ]
}

identifier "euid" "SI" {
  canonicalizer   = canonicalizer.euid.si
  format          = format.euid.si
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "si-ajpes-maticna"
    url              = "https://www.ajpes.si"
    authority        = "Agencija Republike Slovenije za javnopravne evidence in storitve (AJPES)"
    title            = "Poslovni register Slovenije"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SI"
    language         = "sl"
    notes            = "The register publishes the seven digit maticna stevilka."
    license_or_terms = "Slovenian public sector information"
    tier             = "primary"
  }
}

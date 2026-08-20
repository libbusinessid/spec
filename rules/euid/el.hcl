# Greece - EUID
#
# The registration part is the twelve digit GEMI number of the general
# commercial registry.

canonicalizer "euid" "el" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "el" {
  capture "register" {
    value = before_first(after_first(subject(), "EL"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.el.empty"),
    require(starts_with(subject(), "EL"), "invalid_format", "euid.el.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.el.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.el.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.el.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.el.register_characters"),
    require(length_eq(capture.registration, 12), "invalid_length", "euid.el.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.el.registration_characters"),
  ]
}

identifier "euid" "EL" {
  canonicalizer   = canonicalizer.euid.el
  format          = format.euid.el
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "el-gemi-number"
    url              = "https://www.businessportal.gr"
    authority        = "Geniko Emporiko Mitroo (GEMI)"
    title            = "General Commercial Registry"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GR"
    language         = "el"
    notes            = "The registry publishes the twelve digit GEMI number."
    license_or_terms = "Greek public sector information"
    tier             = "primary"
  }
}

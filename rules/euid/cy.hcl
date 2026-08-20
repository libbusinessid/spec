# Cyprus - EUID
#
# The registration part is the six digit HE number of the registrar of
# companies. Leading zeros are significant.

canonicalizer "euid" "cy" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "cy" {
  capture "register" {
    value = before_first(after_first(subject(), "CY"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.cy.empty"),
    require(starts_with(subject(), "CY"), "invalid_format", "euid.cy.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.cy.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.cy.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.cy.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.cy.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.cy.he_number
    input = capture.registration
  }
}

identifier "euid" "CY" {
  canonicalizer   = canonicalizer.euid.cy
  format          = format.euid.cy
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "cy-drcor-he"
    url              = "https://www.companies.gov.cy"
    authority        = "Department of Registrar of Companies and Intellectual Property"
    title            = "Registrar of Companies"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CY"
    language         = "el"
    notes            = "The registrar publishes the six digit HE number."
    license_or_terms = "Cypriot public sector information"
    tier             = "primary"
  }
}

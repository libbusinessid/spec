# Cyprus - registration number
#
# The registration part of the Cyprus EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "cy" "he_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/", " "]),
  ]
}

format "cy" "he_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "cy.he_number.empty"),
    require(length_eq(subject(), 6), "invalid_length", "cy.he_number.length"),
    require(ascii_digits(subject()), "invalid_characters", "cy.he_number.characters"),
  ]
}

dispatcher "he_number" {
  aliases           = ["cy_he_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "CY"
    identifier                       = identifier.he_number.CY
    allow_unprefixed_without_country = true
  }
}

identifier "he_number" "CY" {
  canonicalizer   = canonicalizer.cy.he_number
  format          = format.cy.he_number
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

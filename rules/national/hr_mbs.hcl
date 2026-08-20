# Croatia - court register number (MBS)
#
# The registration part of the Croatia EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "hr" "mbs" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "hr" "mbs" {
  checks = [
    require(not(is_empty(subject())), "empty", "hr.mbs.empty"),
    require(length_eq(subject(), 8), "invalid_length", "hr.mbs.length"),
    require(ascii_digits(subject()), "invalid_characters", "hr.mbs.characters"),
  ]
}

dispatcher "mbs" {
  aliases           = ["hr_mbs"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "HR"
    identifier                       = identifier.mbs.HR
    allow_unprefixed_without_country = true
  }
}

identifier "mbs" "HR" {
  canonicalizer   = canonicalizer.hr.mbs
  format          = format.hr.mbs
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

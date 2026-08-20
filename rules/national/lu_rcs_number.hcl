# Luxembourg - RCS number
#
# The registration part of the Luxembourg EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "lu" "rcs_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "lu" "rcs_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "lu.rcs_number.empty"),
    require(length_between(subject(), 5, 7), "invalid_length", "lu.rcs_number.length"),
    require(starts_with(subject(), "B"), "invalid_format", "lu.rcs_number.prefix"),
    require(ascii_digits(slice_from(subject(), 1)), "invalid_characters", "lu.rcs_number.characters"),
  ]
}

dispatcher "rcs_number" {
  aliases           = ["lu_rcs_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "LU"
    identifier                       = identifier.rcs_number.LU
    allow_unprefixed_without_country = true
  }
}

identifier "rcs_number" "LU" {
  canonicalizer   = canonicalizer.lu.rcs_number
  format          = format.lu.rcs_number
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

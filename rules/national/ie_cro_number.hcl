# Ireland - CRO number
#
# The registration part of the Ireland EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "ie" "cro_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "ie" "cro_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "ie.cro_number.empty"),
    require(length_between(subject(), 5, 7), "invalid_length", "ie.cro_number.length"),
    require(ascii_digits(subject()), "invalid_characters", "ie.cro_number.characters"),
  ]
}

dispatcher "cro_number" {
  aliases           = ["ie_cro_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "IE"
    identifier                       = identifier.cro_number.IE
    allow_unprefixed_without_country = true
  }
}

identifier "cro_number" "IE" {
  canonicalizer   = canonicalizer.ie.cro_number
  format          = format.ie.cro_number
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

# Poland - KRS number
#
# The registration part of the Poland EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "pl" "krs" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "pl" "krs" {
  checks = [
    require(not(is_empty(subject())), "empty", "pl.krs.empty"),
    require(length_eq(subject(), 10), "invalid_length", "pl.krs.length"),
    require(ascii_digits(subject()), "invalid_characters", "pl.krs.characters"),
  ]
}

dispatcher "krs" {
  aliases           = ["pl_krs"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "PL"
    identifier                       = identifier.krs.PL
    allow_unprefixed_without_country = true
  }
}

identifier "krs" "PL" {
  canonicalizer   = canonicalizer.pl.krs
  format          = format.pl.krs
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "pl-krs-number"
    url              = "https://prs.ms.gov.pl"
    authority        = "Ministerstwo Sprawiedliwosci"
    title            = "Krajowy Rejestr Sadowy"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PL"
    language         = "pl"
    notes            = "The register publishes the ten digit KRS number."
    license_or_terms = "Polish public sector information"
    tier             = "primary"
  }
}
